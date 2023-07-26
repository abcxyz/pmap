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
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/pkg/server"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	testGithubCommitValue  = "prober-test-github-commit"
	testGithubRepoValue    = "prober-test-github-repo"
	testWorkflowValue      = "prober-test-workflow"
	testWorkflowShaValue   = "prober-test-workflow-sha"
	testWorkflowRunID      = "prober-test-workflow-id"
	testWorkflowRunAttempt = "1"
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

	ts := time.Now().Format(time.RFC3339)

	// TODO: add probePolicy to probe policy service
	var probeErr error
	if err := probeMapping(ctx, ts); err != nil {
		probeErr = errors.Join(probeErr, fmt.Errorf("prober failed for mapping service: %w", err))
	} else {
		log.Print("Mapping probe succeed")
	}

	return probeErr
}

// probeMapping probe the mapping service by uploading file, query the bigquery table
// and compare the result.
func probeMapping(ctx context.Context, timestamp string) error {
	log.Print("Mapping probe started")
	traceID := fmt.Sprintf("prober-mapping-%s", timestamp)
	log.Printf("using traceID: %s", traceID)
	data := []byte(fmt.Sprintf(`
resource:
  name: //storage.googleapis.com/%s
  provider: gcp
annotations:
  traceID: %s
contacts:
  email:
  - prober@pmap.com
`, cfg.GCSBucketID, traceID))

	filepath := fmt.Sprintf("mapping/prober-files/timestamp-%s", timestamp)

	if err := uploadFile(ctx, cfg.GCSBucketID, filepath, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("failed to uploaded mapping object: %w", err)
	}

	queryString := fmt.Sprintf("SELECT data FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, cfg.MappingTableID)
	queryString += ` WHERE JSON_VALUE(data.payload.annotations.traceID) = ?`

	bqQuery := bqClient.Query(queryString)
	bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID}}

	gotBQEntry, err := getFirstMatchedBQEntryWithRetries(ctx, bqQuery, cfg)
	if err != nil {
		return fmt.Errorf("failed to get match bigquery result: %w", err)
	}

	wantResourceMapping := &v1alpha1.ResourceMapping{
		Resource: &v1alpha1.Resource{
			Provider: "gcp",
			Name:     fmt.Sprintf("//storage.googleapis.com/%s", cfg.GCSBucketID),
		},
		Contacts: &v1alpha1.Contacts{Email: []string{"prober@pmap.com"}},
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
		RepoName:           testGithubRepoValue,
		Commit:             testGithubCommitValue,
		Workflow:           testWorkflowValue,
		WorkflowSha:        testWorkflowShaValue,
		WorkflowRunId:      testWorkflowRunID,
		WorkflowRunAttempt: 1,
	}

	if diff := cmp.Diff(wantGithubSource, gotPmapEvent.GetGithubSource(), cmpOpts...); diff != "" {
		diffErr = errors.Join(diffErr, fmt.Errorf("githubSource unexpected diff (-want, +got):\n%s", diff))
	}

	return diffErr
}

// uploadFile uploads a object to the GCS bucket.
func uploadFile(ctx context.Context, bucket, object string, data io.Reader) error {
	o := gcsClient.Bucket(bucket).Object(object)
	o = o.If(storage.Conditions{DoesNotExist: true})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	wc.Metadata = map[string]string{
		server.MetadataKeyGitHubCommit:       testGithubCommitValue,
		server.MetadataKeyGitHubRepo:         testGithubRepoValue,
		server.MetadataKeyWorkflow:           testWorkflowValue,
		server.MetadataKeyWorkflowSha:        testWorkflowShaValue,
		server.MetadataKeyWorkflowRunAttempt: testWorkflowRunAttempt,
		server.MetadataKeyWorkflowRunID:      testWorkflowRunID,
	}

	if _, err := io.Copy(wc, data); err != nil {
		return fmt.Errorf("failed to copy bytes: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// getFirstMatchedBQEntryWithRetries query BigQuery table to find and return the matching entry.
// If no result is found, query will be retried with the retry config.
func getFirstMatchedBQEntryWithRetries(ctx context.Context, bqQuery *bigquery.Query, cfg *config) (string, error) {
	b := retry.NewConstant(cfg.QueryRetryWaitDuration)
	var entry string
	if err := retry.Do(ctx, retry.WithMaxRetries(cfg.QueryRetryLimit, b), func(ctx context.Context) error {
		results, err := queryBQEntries(ctx, bqQuery)
		if err != nil {
			return err
		}

		// Early exit retry if queried pmap event already found.
		if len(results) > 0 {
			entry = results[0]
			return nil
		}
		log.Print("Matching entry not found, retrying...")
		return retry.RetryableError(fmt.Errorf("no matching pmap event found in bigquery after timeout"))
	}); err != nil {
		return "", fmt.Errorf("retry failed: %w", err)
	}
	return entry, nil
}

// queryBQEntries queries the BQ and checks if a bigqury entry matched the query exists or not and return the results.
func queryBQEntries(ctx context.Context, query *bigquery.Query) ([]string, error) {
	job, err := query.Run(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to run query: %w", err))
	}

	if status, err := job.Wait(ctx); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to wait for query: %w", err))
	} else if err = status.Err(); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("query failed: %w", err))
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to read job: %w", err))
	}

	var entries []string
	for {
		var entry []bigquery.Value
		err := it.Next(&entry)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get next entry: %w", err)
		}
		v, ok := entry[0].(string)
		if !ok {
			return nil, fmt.Errorf("failed to parse %v to string: %w", entry[0], err)
		}
		entries = append(entries, v)
	}
	return entries, nil
}
