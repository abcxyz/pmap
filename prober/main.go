// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/internal/testhelper"
	"github.com/abcxyz/pmap/pkg/server"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	proberGithubCommitValue    = "prober-test-github-commit"
	proberGithubRepoValue      = "prober-test-github-repo"
	proberWorkflowValue        = "prober-test-workflow"
	proberWorkflowShaValue     = "prober-test-workflow-sha"
	proberWorkflowRunID        = "prober-test-workflow-id"
	proberWorkflowRunAttempt   = "1"
	proberGCSNamePrefix        = "//storage.googleapis.com"
	proberMappingTraceIDPrefix = "prober-mapping"
	proberResourceProvider     = "gcp"
	proberResourceContact      = "pmap-prober@abcxyz.dev"
)

var (
	cfg       *config
	bqClient  *bigquery.Client
	gcsClient *storage.Client
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer done()

	if err := realMain(ctx); err != nil {
		done()
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain(ctx context.Context) error {
	logging := logging.FromContext(ctx)
	// create a global config
	c, err := newTestConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse integration test config: %w", err)
	}
	cfg = c

	// create a global gcs client
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create gcs client: %w", err)
	}
	gcsClient = sc
	defer sc.Close()

	// create a global bigquery client
	bc, err := bigquery.NewClient(ctx, c.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create bigquery client: %w", err)
	}
	bqClient = bc
	defer bc.Close()

	ts := strconv.FormatInt(time.Now().Unix(), 10)

	var probeErr error
	if err := probeMapping(ctx, ts); err != nil {
		probeErr = errors.Join(probeErr, fmt.Errorf("prober failed for mapping service: %w", err))
	}

	// TODO: add probePolicy() to probe policy service
	// errors.Join() is used here so we can join the errors from probePolicy and return them together.
	// if err := probePolicy(ctx, ts); err != nil {
	// 	probeErr = errors.Join(probeErr, fmt.Errorf("prober failed for policy service: %w", err))
	// } else {
	// 	logging.Info("Policy probe failed.")
	// }
	if probeErr == nil {
		logging.Info("Mapping probe successed")
	}

	return probeErr
}

// probeMapping probe the mapping service by uploading file, query the bigquery table
// and compare the result.
func probeMapping(ctx context.Context, timestamp string) error {
	logging := logging.FromContext(ctx)
	logging.Info("Mapping probe started")

	traceID := fmt.Sprintf("%s-%s", proberMappingTraceIDPrefix, timestamp)
	logging.Infof("using traceID: %s", traceID)

	data := []byte(fmt.Sprintf(`
resource:
  name: %s/%s
  provider: %s
annotations:
  traceID: %s
contacts:
  email:
  - %s
`, proberGCSNamePrefix, cfg.GCSBucketID, proberResourceProvider, traceID, proberResourceContact))

	filepath := fmt.Sprintf("%s/%s-%s", cfg.ProberMappingServiceName, cfg.ProberMappingFilePrefix, timestamp)

	if err := testhelper.UploadGCSFile(ctx, gcsClient, cfg.GCSBucketID, filepath, bytes.NewReader(data), getProberGCSMetadata()); err != nil {
		return fmt.Errorf("failed to uploaded mapping object: %w", err)
	}

	queryString := fmt.Sprintf("SELECT data FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, cfg.MappingTableID)
	queryString += ` WHERE JSON_VALUE(data.payload.annotations.traceID) = ?`

	bqQuery := bqClient.Query(queryString)
	bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID}}

	gotBQEntry, err := getFirstMatchedBQEntry(ctx, bqQuery, cfg)
	if err != nil {
		return fmt.Errorf("failed to get match bigquery result: %w", err)
	}

	wantResourceMapping := &v1alpha1.ResourceMapping{
		Resource: &v1alpha1.Resource{
			Provider: proberResourceProvider,
			Name:     fmt.Sprintf("%s/%s", proberGCSNamePrefix, cfg.GCSBucketID),
		},
		Contacts: &v1alpha1.Contacts{Email: []string{proberResourceContact}},
	}

	gotPmapEvent := &v1alpha1.PmapEvent{}
	if err := protojson.Unmarshal([]byte(gotBQEntry), gotPmapEvent); err != nil {
		return fmt.Errorf("failed to unmarshal bigquery result to pmapevent: %w", err)
	}

	resourceMapping := &v1alpha1.ResourceMapping{}
	if err := gotPmapEvent.GetPayload().UnmarshalTo(resourceMapping); err != nil {
		return fmt.Errorf("failed to unmarshal to resource mapping: %w", err)
	}

	cmpOpts := []cmp.Option{
		protocmp.Transform(),
		protocmp.IgnoreFields(&v1alpha1.ResourceMapping{}, "annotations"),
	}

	var diffErr error
	if diff := cmp.Diff(wantResourceMapping, resourceMapping, cmpOpts...); diff != "" {
		diffErr = errors.Join(diffErr, fmt.Errorf("resourcemapping(ignore annotation) unexpected diff (-want,+got):\n%s", diff))
	}

	wantGithubSource := &v1alpha1.GitHubSource{
		RepoName:           proberGithubRepoValue,
		Commit:             proberGithubCommitValue,
		Workflow:           proberWorkflowValue,
		WorkflowSha:        proberWorkflowShaValue,
		WorkflowRunId:      proberWorkflowRunID,
		WorkflowRunAttempt: 1,
	}

	if diff := cmp.Diff(wantGithubSource, gotPmapEvent.GetGithubSource(), protocmp.Transform()); diff != "" {
		diffErr = errors.Join(diffErr, fmt.Errorf("githubSource unexpected diff (-want, +got):\n%s", diff))
	}

	// check CAIS annotation exist to make sure CAIS is working
	if _, ok := resourceMapping.GetAnnotations().GetFields()[v1alpha1.AnnotationKeyAssetInfo].GetStructValue().AsMap()["ancestors"]; !ok {
		diffErr = errors.Join(diffErr, fmt.Errorf("ancestors is blank in resourcemapping.annotations"))
	}
	if _, ok := resourceMapping.GetAnnotations().GetFields()[v1alpha1.AnnotationKeyAssetInfo].GetStructValue().AsMap()["iamPolicies"]; !ok {
		diffErr = errors.Join(diffErr, fmt.Errorf("iamPolicies is blank in resourcemapping.annotations"))
	}

	return diffErr
}

// getProberGCSMetadata returns the metadata of an object that being uploaded to GCS.
func getProberGCSMetadata() map[string]string {
	return map[string]string{
		server.MetadataKeyGitHubCommit:       proberGithubCommitValue,
		server.MetadataKeyGitHubRepo:         proberGithubRepoValue,
		server.MetadataKeyWorkflow:           proberWorkflowValue,
		server.MetadataKeyWorkflowSha:        proberWorkflowShaValue,
		server.MetadataKeyWorkflowRunAttempt: proberWorkflowRunAttempt,
		server.MetadataKeyWorkflowRunID:      proberWorkflowRunID,
	}
}

// getFirstMatchedBQEntryWithRetries query BigQuery table to find and return the matching entry.
// If no result is found, query will be retried with the retry config.
func getFirstMatchedBQEntry(ctx context.Context, bqQuery *bigquery.Query, cfg *config) (string, error) {
	entry, err := testhelper.GetFirstMatchedBQEntryWithRetries(ctx, bqQuery, cfg.QueryRetryWaitDuration, cfg.QueryRetryLimit)
	if err != nil {
		return "", fmt.Errorf("failed to get matched bq entry: %w", err)
	}
	return entry.Data, nil
}
