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
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	// Global integration test config.
	cfg *config
	// Global BQ client for integration test.
	bqClient *bigquery.Client
	// Global GCS client for integration test.
	gcsClient *storage.Client
)

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
		name                string
		resourceName        string
		bigqueryTable       string
		wantCAISProcessed   bool
		wantResourceMapping *v1alpha1.ResourceMapping
	}{
		{
			name:              "mapping_success_event",
			resourceName:      fmt.Sprintf("//storage.googleapis.com/%s", cfg.GCSStaticBucket),
			bigqueryTable:     cfg.MappingTableID,
			wantCAISProcessed: true,
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     fmt.Sprintf("//storage.googleapis.com/%s", cfg.GCSStaticBucket),
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"group@example.com"}},
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
			gcsObject := fmt.Sprintf("mapping/%s/traceID-%s.yaml", cfg.ObjectPrefix, traceID)
			// Upload data to GCS, this should trigger the pmap event handler via GCS notification behind the scenes.
			if err := testUploadFile(ctx, t, cfg.GCSBucketID, gcsObject, bytes.NewReader(data)); err != nil {
				t.Fatalf("failed to upload object %s to bucket %s: %v", gcsObject, cfg.GCSBucketID, err)
			}

			// Check if the file uploaded exists in BigQuery.
			queryString := fmt.Sprintf("SELECT data FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += ` WHERE JSON_VALUE(data.payload.annotations.traceID) = ?`
			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID.String()}}

			gotPmapEvent := testGetFirstPmapEventWithRetries(ctx, t, bqQuery, cfg)

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
		})
	}
}

// TestPolicyEventHandling tests the entire flow from uploading policy data to GCS, triggering pmap event handler, to writing
// it to a BigQuery table.
func TestPolicyEventHandling(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		bigqueryTable string
		wantPolicyID  string
	}{
		{
			name:          "policy_success_event",
			bigqueryTable: cfg.PolicyTableID,
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

			data := []byte(fmt.Sprintf(`
policy_id: fake-policy-123
annotations:
  traceID: %s
deletion_timeline:
  - 356 days
  - 1 day
`, traceID.String()))
			gcsObject := fmt.Sprintf("policy/%s/traceID-%s.yaml", cfg.ObjectPrefix, traceID)
			// Upload data to GCS, this should trigger the pmap event handler via GCS notification behind the scenes.
			if err := testUploadFile(ctx, t, cfg.GCSBucketID, gcsObject, bytes.NewReader(data)); err != nil {
				t.Fatalf("failed to upload object %s to bucket %s: %v", gcsObject, cfg.GCSBucketID, err)
			}

			// Check if the file uploaded exists in BigQuery.
			queryString := fmt.Sprintf("SELECT data FROM `%s.%s.%s`", cfg.ProjectID, cfg.BigQueryDataSetID, tc.bigqueryTable)
			queryString += `WHERE JSON_VALUE(data.payload.value.annotations.traceID) = ?`
			bqQuery := bqClient.Query(queryString)
			bqQuery.Parameters = []bigquery.QueryParameter{{Value: traceID.String()}}

			gotPmapEvent := testGetFirstPmapEventWithRetries(ctx, t, bqQuery, cfg)

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
		})
	}
}

// testGetFirstPmapEventWithRetries queries the DB and get the matched PmapEvent.
// If no matched PmapEvent was found, the query will be retried with the specified retry inputs.
func testGetFirstPmapEventWithRetries(ctx context.Context, tb testing.TB, bqQuery *bigquery.Query, cfg *config) *v1alpha1.PmapEvent {
	tb.Helper()

	b := retry.NewConstant(cfg.QueryRetryWaitDuration)
	var pmapEvents []*v1alpha1.PmapEvent
	if err := retry.Do(ctx, retry.WithMaxRetries(cfg.QueryRetryLimit, b), func(ctx context.Context) error {
		results, err := queryPmapEvents(ctx, bqQuery)
		if err != nil {
			tb.Logf("failed to query pmap events: %v", err)
			return err
		}

		// Early exit retry if queried pmap event already found.
		if len(results) > 0 {
			pmapEvents = results
			return nil
		}
		tb.Log("Matching entry not found, retrying...")
		return retry.RetryableError(fmt.Errorf("no matching pmap event found in bigquery after timeout"))
	}); err != nil {
		tb.Fatalf("Retry failed: %v.", err)
	}
	return pmapEvents[0]
}

// queryPmapEvents queries the BQ and checks if pmap events matched the query exists or not and return the results.
func queryPmapEvents(ctx context.Context, query *bigquery.Query) ([]*v1alpha1.PmapEvent, error) {
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
	var pmapEvents []*v1alpha1.PmapEvent
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}
		value, ok := row[0].(string)
		if !ok {
			return nil, fmt.Errorf("failed to convert query (%T) to string: %w", value[0], err)
		}
		var pmapEvent v1alpha1.PmapEvent
		if err := protojson.Unmarshal([]byte(value), &pmapEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bq row to pmapEvent: %w", err)
		}
		pmapEvents = append(pmapEvents, &pmapEvent)
	}
	return pmapEvents, nil
}

// testUploadFile uploads an object to the GCS bucket and
// automatically delete the object when the tests finish.
func testUploadFile(ctx context.Context, tb testing.TB, bucket, object string, data io.Reader) error {
	tb.Helper()

	tb.Cleanup(func() {
		o := gcsClient.Bucket(bucket).Object(object)

		if err := o.Delete(ctx); err != nil {
			tb.Logf("failed to delete gcs object(%q).Delete: %v", object, err)
		}
	})

	// TODO: #41 set up GCS upload retry.
	o := gcsClient.Bucket(bucket).Object(object)

	// For an object that does not yet exist, set the DoesNotExist precondition.
	o = o.If(storage.Conditions{DoesNotExist: true})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	if _, err := io.Copy(wc, data); err != nil {
		return fmt.Errorf("failed to copy bytes: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}
