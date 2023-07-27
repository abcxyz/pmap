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

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pmap/apis/v1alpha1"

	"github.com/abcxyz/pmap/internal/testhelper"
	"github.com/abcxyz/pmap/pkg/server"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testGithubCommitValue          = "test-github-commit"
	testGithubRepoValue            = "test-github-repo"
	testWorkflowValue              = "test-workflow"
	testWorkflowShaValue           = "test-workflow-sha"
	testWorkflowTriggeredTimeValue = "2023-04-25T17:44:57+00:00"
	testWorkflowRunID              = "5050509831"
	testWorkflowRunAttempt         = "1"
)

var (
	// Global integration test config.
	cfg *config
	// Global BQ client for integration test.
	bqClient *bigquery.Client
	// Global GCS client for integration test.
	gcsClient *storage.Client
)

type attributes struct {
	ProcessErr string
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		ctx := context.Background()

		if strings.ToLower(os.Getenv("TEST_INTEGRATION")) != "true" {
			log.Printf("skipping (not integration)")
			// Not integration test. Exit.
			return 0
		}

		// set up global test config.
		c, err := newTestConfig(ctx)
		if err != nil {
			log.Printf("Failed to parse integration test config: %v", err)
			return 2
		}
		cfg = c

		// set up global bq client.
		bc, err := bigquery.NewClient(ctx, cfg.ProjectID)
		if err != nil {
			log.Printf("failed to create bigquery client: %v", err)
			return 2
		}
		defer bc.Close()
		bqClient = bc

		// set up global gcs client.
		sc, err := storage.NewClient(ctx)
		if err != nil {
			log.Printf("failed to create gcs client: %v", err)
			return 2
		}
		gcsClient = sc
		defer sc.Close()

		return m.Run()
	}())
}

// TestMappingEventHandling tests the entire flow from uploading a mapping data
// to GCS, triggering pmap event handler, to writing it to a BigQuery table.
// TODO(#56): add validation logic for github source attributes as object metadata.
func TestMappingEventHandling(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                 string
		resourceName         string
		bigqueryTable        string
		wantCAISProcessed    bool
		wantGithubSource     *v1alpha1.GitHubSource
		wantResourceMapping  *v1alpha1.ResourceMapping
		wantProcessErrSubStr string
	}{
		{
			name:              "mapping_success_event_scoped_resource",
			resourceName:      fmt.Sprintf("//artifactregistry.googleapis.com/projects/%s/locations/us-central1/repositories/%s", cfg.ProjectID, cfg.StaticARRepo),
			bigqueryTable:     cfg.MappingTableID,
			wantCAISProcessed: true,
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     fmt.Sprintf("//artifactregistry.googleapis.com/projects/%s/locations/us-central1/repositories/%s", cfg.ProjectID, cfg.StaticARRepo),
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"group@example.com"}},
			},
			wantGithubSource: &v1alpha1.GitHubSource{
				RepoName:                   testGithubRepoValue,
				Commit:                     testGithubCommitValue,
				Workflow:                   testWorkflowValue,
				WorkflowSha:                testWorkflowShaValue,
				WorkflowTriggeredTimestamp: testParseTime(t, testWorkflowTriggeredTimeValue),
				WorkflowRunId:              testWorkflowRunID,
				WorkflowRunAttempt:         1,
			},
		},
		{
			name:              "mapping_success_event_unscoped_resource",
			resourceName:      fmt.Sprintf("//storage.googleapis.com/%s", cfg.StaticGCSBucket),
			bigqueryTable:     cfg.MappingTableID,
			wantCAISProcessed: true,
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     fmt.Sprintf("//storage.googleapis.com/%s", cfg.StaticGCSBucket),
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"group@example.com"}},
			},
			wantGithubSource: &v1alpha1.GitHubSource{
				RepoName:                   testGithubRepoValue,
				Commit:                     testGithubCommitValue,
				Workflow:                   testWorkflowValue,
				WorkflowSha:                testWorkflowShaValue,
				WorkflowTriggeredTimestamp: testParseTime(t, testWorkflowTriggeredTimeValue),
				WorkflowRunId:              testWorkflowRunID,
				WorkflowRunAttempt:         1,
			},
		},
		{
			name:          "mapping_failure_event",
			resourceName:  fmt.Sprintf("//pubsub.googleapis.com/projects/%s/topics/%s", cfg.ProjectID, "non_existent_topic"),
			bigqueryTable: cfg.MappingFailureTableID,
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     fmt.Sprintf("//pubsub.googleapis.com/projects/%s/topics/%s", cfg.ProjectID, "non_existent_topic"),
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"group@example.com"}},
			},
			wantGithubSource: &v1alpha1.GitHubSource{
				RepoName:                   testGithubRepoValue,
				Commit:                     testGithubCommitValue,
				Workflow:                   testWorkflowValue,
				WorkflowSha:                testWorkflowShaValue,
				WorkflowTriggeredTimestamp: testParseTime(t, testWorkflowTriggeredTimeValue),
				WorkflowRunId:              testWorkflowRunID,
				WorkflowRunAttempt:         1,
			},
			wantProcessErrSubStr: "failed to validate and enrich",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			traceID, err := rand.Int(rand.Reader, big.NewInt(100000))
			if err != nil {
				t.Fatalf("failed to generate random int")
			}
			t.Logf("using trace ID %s", traceID.String())

			filePath := fmt.Sprintf("test-dir/traceID-%s.yaml", traceID)
			tc.wantGithubSource.FilePath = filePath

			data := []byte(fmt.Sprintf(`
resource:
  name: %s
  provider: gcp
annotations:
  traceID: %s
contacts:
  email:
  - group@example.com
`, tc.resourceName, traceID.String()))
			gcsObject := fmt.Sprintf("mapping/%s/gh-prefix/%s", cfg.ObjectPrefix, filePath)
			// Upload data to GCS, this should trigger the pmap event handler via GCS notification behind the scenes.
			if err := testUploadFile(ctx, t, cfg.GCSBucketID, gcsObject, bytes.NewReader(data)); err != nil {
				t.Fatalf("failed to upload object %s to bucket %s: %v", gcsObject, cfg.GCSBucketID, err)
			}

			// Check if the file uploaded exists in BigQuery.
			queryString := fmt.Sprintf("SELECT * FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += ` WHERE JSON_VALUE(data.payload.annotations.traceID) = ?`
			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID.String()}}

			gotBQEntry := testGetFirstMatchedBQEntry(ctx, t, bqQuery, cfg)

			gotPmapEvent := &v1alpha1.PmapEvent{}
			if err := protojson.Unmarshal([]byte(gotBQEntry.Data), gotPmapEvent); err != nil {
				t.Fatalf("failed to unmarshal BQEntry.Data to pmapevent: %v", err)
			}

			gotAttributes := &attributes{}
			if err := json.Unmarshal([]byte(gotBQEntry.Attributes), gotAttributes); err != nil {
				t.Fatalf("failed to unmarshal BQEntry.Attributes to Attributes: %v", err)
			}

			resourceMapping := &v1alpha1.ResourceMapping{}
			if err := gotPmapEvent.GetPayload().UnmarshalTo(resourceMapping); err != nil {
				t.Fatalf("failed to unmarshal to resource mapping: %v", err)
			}

			cmpOpts := []cmp.Option{
				protocmp.Transform(),
				protocmp.IgnoreFields(&v1alpha1.ResourceMapping{}, "annotations"),
			}
			if diff := cmp.Diff(tc.wantResourceMapping, resourceMapping, cmpOpts...); diff != "" {
				t.Errorf("resourcemapping(ignore annotation) unexpected diff (-want,+got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantGithubSource, gotPmapEvent.GetGithubSource(), cmpOpts...); diff != "" {
				t.Errorf("githubSource unexpected diff (-want, +got):\n%s", diff)
			}

			// Resources that don't exist won't pass the validation of CAIS processor,
			// therefore, additional metadata including 'ancestors' and 'iamPolicies' won't get attached.
			if tc.wantCAISProcessed {
				if _, ok := resourceMapping.GetAnnotations().GetFields()[v1alpha1.AnnotationKeyAssetInfo].GetStructValue().AsMap()["ancestors"]; !ok {
					t.Errorf("ancestors is blank in resourcemapping.annotations")
				}
				if _, ok := resourceMapping.GetAnnotations().GetFields()[v1alpha1.AnnotationKeyAssetInfo].GetStructValue().AsMap()["iamPolicies"]; !ok {
					t.Errorf("iamPolicies is blank in resourcemapping.annotations")
				}
			}
			if !strings.Contains(gotAttributes.ProcessErr, tc.wantProcessErrSubStr) {
				t.Errorf("case %s expect %s to contain %s", tc.name, gotAttributes.ProcessErr, tc.wantProcessErrSubStr)
			}
		})
	}
}

// TestPolicyEventHandling tests the entire flow from uploading policy data to GCS, triggering pmap event handler, to writing
// it to a BigQuery table.
func TestPolicyEventHandling(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		bigqueryTable    string
		wantPolicyID     string
		wantGithubSource *v1alpha1.GitHubSource
	}{
		{
			name:          "policy_success_event",
			bigqueryTable: cfg.PolicyTableID,
			wantGithubSource: &v1alpha1.GitHubSource{
				RepoName:                   testGithubRepoValue,
				Commit:                     testGithubCommitValue,
				Workflow:                   testWorkflowValue,
				WorkflowSha:                testWorkflowShaValue,
				WorkflowTriggeredTimestamp: testParseTime(t, testWorkflowTriggeredTimeValue),
				WorkflowRunId:              testWorkflowRunID,
				WorkflowRunAttempt:         1,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			traceID, err := rand.Int(rand.Reader, big.NewInt(100000))
			if err != nil {
				t.Fatalf("failed to generate random int")
			}
			t.Logf("using trace ID %s", traceID.String())

			filePath := fmt.Sprintf("test-dir/traceID-%s.yaml", traceID)
			tc.wantGithubSource.FilePath = filePath

			data := []byte(fmt.Sprintf(`
policy_id: fake-policy-123
annotations:
  traceID: %s
deletion_timeline:
  - 356 days
  - 1 day
`, traceID.String()))
			gcsObject := fmt.Sprintf("policy/%s/gh-prefix/%s", cfg.ObjectPrefix, filePath)
			// Upload data to GCS, this should trigger the pmap event handler via GCS notification behind the scenes.
			if err := testUploadFile(ctx, t, cfg.GCSBucketID, gcsObject, bytes.NewReader(data)); err != nil {
				t.Fatalf("failed to upload object %s to bucket %s: %v", gcsObject, cfg.GCSBucketID, err)
			}

			// Check if the file uploaded exists in BigQuery.
			queryString := fmt.Sprintf("SELECT * FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += `WHERE JSON_VALUE(data.payload.value.annotations.traceID) = ?`
			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID.String()}}

			gotBQEntry := testGetFirstMatchedBQEntry(ctx, t, bqQuery, cfg)

			gotPmapEvent := &v1alpha1.PmapEvent{}
			if err := protojson.Unmarshal([]byte(gotBQEntry.Data), gotPmapEvent); err != nil {
				t.Fatalf("failed to unmarshal BQEntry.Data to pmapevent: %v", err)
			}

			gotPayload := &structpb.Struct{}
			if err = gotPmapEvent.GetPayload().UnmarshalTo(gotPayload); err != nil {
				t.Fatalf("failed to unmarshal to gotPayload: %v", err)
			}

			wantPayload := &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"annotations": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"traceID": structpb.NewNumberValue(float64(traceID.Int64())),
						},
					}),
					"deletion_timeline": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("356 days"), structpb.NewStringValue("1 day")}}),
					"policy_id":         structpb.NewStringValue("fake-policy-123"),
				},
			}

			if diff := cmp.Diff(wantPayload, gotPayload, protocmp.Transform()); diff != "" {
				t.Errorf("gotPayload unexpected diff (-want,+got):\n%s", diff)
			}
			cmpOpts := []cmp.Option{
				protocmp.Transform(),
			}

			if diff := cmp.Diff(tc.wantGithubSource, gotPmapEvent.GetGithubSource(), cmpOpts...); diff != "" {
				t.Errorf("githubSource unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}
}

// TestMappingReusableWorkflowCall tests the mapping file-copy reusable workflow has successfully
// uploaded the file to gcs and triggered pmap event handler, and write the corresponding entry
// into bigquery.
func TestMappingReusableWorkflowCall(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                string
		bigqueryTable       string
		traceID             string
		wantResourceMapping *v1alpha1.ResourceMapping
	}{
		{
			name:          "mapping-copy-success-event",
			bigqueryTable: cfg.MappingTableID,
			traceID:       "fake-pmap-dev-manual-test-mapping",
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     fmt.Sprintf("//storage.googleapis.com/%s", "pmap-static-ci-bucket-7faa"),
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap.mapping@gmail.com"}},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			t.Logf("using trace ID %s", tc.traceID)
			t.Logf("using workflow run ID %s", cfg.WorkflowRunID)

			// Check if the file uploaded exists in BigQuery.
			// We use workflowRunId here as the unique identifier as this is the test the
			// reusable workflow and the traceID is written in the yaml file, which won't
			// be unique if we don't change the yaml file. But workflowRunID is unique for
			// all workflow runs, and using workflowRunId to query, we can make sure the
			// reusable workflow uploaded the correct GCS object metadata, and later we only
			// need to diff the enties in the yaml file.
			queryString := fmt.Sprintf("SELECT * FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += `WHERE JSON_VALUE(data.githubSource.workflowRunId) = ?`

			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: cfg.WorkflowRunID}}

			gotBQEntry := testGetFirstMatchedBQEntry(ctx, t, bqQuery, cfg)

			gotPmapEvent := &v1alpha1.PmapEvent{}
			if err := protojson.Unmarshal([]byte(gotBQEntry.Data), gotPmapEvent); err != nil {
				t.Fatalf("failed to unmarshal BQEntry.Data to pmapevent: %v", err)
			}

			gotResourceMapping := &v1alpha1.ResourceMapping{}
			if err := gotPmapEvent.GetPayload().UnmarshalTo(gotResourceMapping); err != nil {
				t.Fatalf("failed to unmarshal to resource mapping: %v", err)
			}

			cmpOpts := []cmp.Option{
				protocmp.Transform(),
				protocmp.IgnoreFields(&v1alpha1.ResourceMapping{}, "annotations"),
			}

			if diff := cmp.Diff(tc.wantResourceMapping, gotResourceMapping, cmpOpts...); diff != "" {
				t.Errorf("gotPayload unexpected diff (-want,+got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.traceID, gotResourceMapping.Annotations.AsMap()["traceID"]); diff != "" {
				t.Errorf("traceID got unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}

// TestMappingReusableWorkflowCall tests the policy file-copy reusable workflow has successfully
// uploaded the file to gcs and triggered pmap event handler, and write the corresponding entry
// into bigquery.
func TestPolicyReusableWorkflowCall(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		bigqueryTable string
		traceID       string
		wantPayload   *structpb.Struct
	}{
		{
			name:          "policy-copy-success-event",
			bigqueryTable: cfg.PolicyTableID,
			traceID:       "fake-pmap-dev-manual-test-policy",
			wantPayload: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"annotations": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"traceID": structpb.NewStringValue("fake-pmap-dev-manual-test-policy"),
						},
					}),
					"deletion_timeline": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("356 days"), structpb.NewStringValue("1 day")}}),
					"policy_id":         structpb.NewStringValue("fake-policy-123"),
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			t.Logf("using trace ID %s", tc.traceID)
			t.Logf("using workflow run ID %s", cfg.WorkflowRunID)

			// Check if the file uploaded exists in BigQuery.
			// We use workflowRunId here as the unique identifier as this is the test the
			// reusable workflow and the traceID is written in the yaml file, which won't
			// be unique if we don't change the yaml file. But workflowRunID is unique for
			// all workflow runs, and using workflowRunId to query, we can make sure the
			// reusable workflow uploaded the correct GCS object metadata, and later we only
			// need to diff the enties in the yaml file.
			queryString := fmt.Sprintf("SELECT * FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += `WHERE JSON_VALUE(data.githubSource.workflowRunId) = ?`

			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: cfg.WorkflowRunID}}

			gotBQEntry := testGetFirstMatchedBQEntry(ctx, t, bqQuery, cfg)

			gotPmapEvent := &v1alpha1.PmapEvent{}
			if err := protojson.Unmarshal([]byte(gotBQEntry.Data), gotPmapEvent); err != nil {
				t.Fatalf("failed to unmarshal BQEntry.Data to pmapevent: %v", err)
			}

			gotPayload := &structpb.Struct{}
			if err := gotPmapEvent.GetPayload().UnmarshalTo(gotPayload); err != nil {
				t.Fatalf("failed to unmarshal to resource mapping: %v", err)
			}

			if diff := cmp.Diff(tc.wantPayload, gotPayload, protocmp.Transform()); diff != "" {
				fmt.Printf("gotPayload unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}

// testGetFirstMatchedBQEntryWithRetries queries the DB and get the matched BQEntry.
// If no matched BQEntry was found, the query will be retried with the specified retry inputs.
func testGetFirstMatchedBQEntry(ctx context.Context, tb testing.TB, bqQuery *bigquery.Query, cfg *config) *testhelper.BQEntry {
	tb.Helper()

	entry, err := testhelper.GetFirstMatchedBQEntryWithRetries(ctx, bqQuery, cfg.QueryRetryWaitDuration, cfg.QueryRetryLimit)
	if err != nil {
		tb.Fatalf("failed to get matched bq entry: %v", err)
	}

	return entry
}

// testUploadFile uploads an object to the GCS bucket and
// automatically delete the object when the tests finish.
func testUploadFile(ctx context.Context, tb testing.TB, bucket, object string, data io.Reader) error {
	tb.Helper()

	metadata := map[string]string{
		server.MetadataKeyGitHubCommit:               testGithubCommitValue,
		server.MetadataKeyGitHubRepo:                 testGithubRepoValue,
		server.MetadataKeyWorkflow:                   testWorkflowValue,
		server.MetadataKeyWorkflowSha:                testWorkflowShaValue,
		server.MetadataKeyWorkflowTriggeredTimestamp: testWorkflowTriggeredTimeValue,
		server.MetadataKeyWorkflowRunAttempt:         testWorkflowRunAttempt,
		server.MetadataKeyWorkflowRunID:              testWorkflowRunID,
	}

	if err := testhelper.UploadGCSFile(ctx, gcsClient, bucket, object, data, metadata); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func testParseTime(tb testing.TB, ts string) *timestamppb.Timestamp {
	tb.Helper()
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		tb.Fatal(err)
	}
	return timestamppb.New(t)
}
