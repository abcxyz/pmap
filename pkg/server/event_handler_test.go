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

package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/pkg/pmaperrors"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEventHandler_NewHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup fake storage client.
	hc, done := newTestServer(testHandleObjectRead(t, []byte("test")))
	defer done()
	c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
	if err != nil {
		t.Fatalf("failed to creat GCS storage client %v", err)
	}

	cases := []struct {
		name             string
		opts             []Option
		successMessenger Messenger
		failureMessenger Messenger
		wantErr          string
	}{
		{
			name:             "success",
			successMessenger: &NoopMessenger{},
			failureMessenger: &NoopMessenger{},
		},
		{
			name:             "success_without_failure_event_messenger",
			successMessenger: &NoopMessenger{},
		},
		{
			name:    "missing_success_event_messenger",
			wantErr: "successMessenger cannot be nil",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, gotErr := NewHandler(ctx, []Processor[*structpb.Struct]{}, tc.successMessenger, WithStorageClient(c))

			if diff := testutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
		})
	}
}

func TestEventHandler_HttpHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name               string
		pubsubMessageBytes []byte
		gcsObjectBytes     []byte
		wantStatusCode     int
		wantRespBodySubstr string
	}{
		{
			name: "success",
			pubsubMessageBytes: testToJSON(t, &PubSubMessage{
				Message: struct {
					Data       []byte            `json:"data,omitempty"`
					Attributes map[string]string `json:"attributes"`
				}{
					Attributes: map[string]string{
						"bucketId":      "foo",
						"objectId":      "pmap-test/gh-prefix/dir1/dir2/bar",
						"payloadFormat": "JSON_API_V1",
					},
					Data: []byte(`{
						"metadata": {
							"github-commit": "test-github-commit",
							"github-workflow-triggered-timestamp": "2023-04-25T17:44:57+00:00",
							"github-workflow-sha": "test-workflow-sha",
							"github-workflow": "test-workflow",
							"github-repo": "test-github-repo",
							"github-run-id": "5050509831",
							"github-run-attempt": "1"
						}
					}`),
				},
			}),
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			wantStatusCode:     http.StatusCreated,
			wantRespBodySubstr: "OK",
		},
		{
			name:               "invalid_request_body",
			pubsubMessageBytes: []byte(`}"`),
			gcsObjectBytes:     nil,
			wantStatusCode:     http.StatusBadRequest,
			wantRespBodySubstr: "invalid character",
		},
		{
			name: "failed_handle_event",
			pubsubMessageBytes: testToJSON(t, &PubSubMessage{
				Message: struct {
					Data       []byte            `json:"data,omitempty"`
					Attributes map[string]string `json:"attributes"`
				}{
					Attributes: map[string]string{
						"bucketId":      "foo",
						"objectId":      "pmap-test/gh-prefix/dir1/dir2/bar2",
						"payloadFormat": "JSON_API_V1",
					},
					Data: []byte(`{
						"metadata": {
							"git-commit": "test-github-commit",
							"git-workflow-triggered-timestamp": "2023-04-25T17:44:57+00:00",
							"git-workflow-sha": "test-workflow-sha",
							"git-workflow": "test-workflow",
							"git-repo": "test-github-repo",
							"github-run-id": "5050509831",
							"github-run-attempt": "1",
						}
					}`),
				},
			}),
			wantStatusCode:     http.StatusInternalServerError,
			wantRespBodySubstr: "failed to get GCS object",
		},
		{
			name: "invalid_pubsubmessage_data",
			pubsubMessageBytes: testToJSON(t, &PubSubMessage{
				Message: struct {
					Data       []byte            `json:"data,omitempty"`
					Attributes map[string]string `json:"attributes"`
				}{
					Attributes: map[string]string{
						"bucketId":      "foo",
						"objectId":      "pmap-test/gh-prefix/dir1/dir2/bar",
						"payloadFormat": "JSON_API_V1",
					},
					Data: []byte(`{
						"metadata": {
							"key" : 12
						}
					}`),
				},
			}),
			wantStatusCode:     http.StatusInternalServerError,
			wantRespBodySubstr: "failed to unmarshal payloadMetadata",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup fake storage client.
			hc, done := newTestServer(testHandleObjectRead(t, tc.gcsObjectBytes))
			defer done()
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}

			h, err := NewHandler(ctx, []Processor[*structpb.Struct]{&testProcessor{}}, &NoopMessenger{}, WithStorageClient(c))
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.pubsubMessageBytes))
			resp := httptest.NewRecorder()
			h.HTTPHandler().ServeHTTP(resp, req)

			if resp.Code != tc.wantStatusCode {
				t.Errorf("Process %+v: StatusCode got: %d want: %d", tc.name, resp.Code, tc.wantStatusCode)
			}

			if !strings.Contains(resp.Body.String(), tc.wantRespBodySubstr) {
				t.Errorf("Process %+v: expect ResponseBody: %s to contain: %s", tc.name, resp.Body.String(), tc.wantRespBodySubstr)
			}
		})
	}
}

func TestEventHandler_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                 string
		notification         *pubsub.Message
		gcsObjectBytes       []byte
		githubSourceBytes    []byte
		processors           []Processor[*structpb.Struct]
		successMessenger     *testMessenger
		failureMessenger     *testMessenger
		wantErrSubstr        string
		wantPmapEvent        *v1alpha1.PmapEvent
		wantFailuerPmapEvent *v1alpha1.PmapEvent
		wantAttr             map[string]string
	}{
		{
			name: "success",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(
				`
resource:
  name: //pubsub.googleapis.com/projects/test-project/topics/test-topic
  provider: gcp
annotations:
  labels:
    env: dev
contacts:
  email:
  - test.gmail.com
`),
			processors: []Processor[*structpb.Struct]{&testProcessor{}},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantPmapEvent: &v1alpha1.PmapEvent{
				GithubSource: &v1alpha1.GitHubSource{
					RepoName:                   "test-github-repo",
					Commit:                     "test-github-commit",
					Workflow:                   "test-workflow",
					WorkflowSha:                "test-workflow-sha",
					WorkflowTriggeredTimestamp: timestamppb.New(time.Date(2023, time.April, 25, 17, 44, 57, 0, time.UTC)),
					WorkflowRunId:              "5050509831",
					WorkflowRunAttempt:         1,
					FilePath:                   "dir1/dir2/bar",
				},
			},
		},
		{
			name: "failed_send_downstream",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			processors: []Processor[*structpb.Struct]{&testProcessor{}},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
				returnErr:    fmt.Errorf("always fail"),
			},
			wantErrSubstr: "failed to send succuss event downstream",
			wantPmapEvent: &v1alpha1.PmapEvent{
				GithubSource: &v1alpha1.GitHubSource{
					RepoName:                   "test-github-repo",
					Commit:                     "test-github-commit",
					Workflow:                   "test-workflow",
					WorkflowSha:                "test-workflow-sha",
					WorkflowTriggeredTimestamp: timestamppb.New(time.Date(2023, time.April, 25, 17, 44, 57, 0, time.UTC)),
					WorkflowRunId:              "5050509831",
					WorkflowRunAttempt:         1,
					FilePath:                   "dir1/dir2/bar",
				},
			},
		},
		{
			name: "missing_bucket_id",
			notification: &pubsub.Message{
				Attributes: map[string]string{"objectId": "pmap-test/gh-prefix/dir1/dir2/bar"},
				Data:       testGCSMetadataBytes(),
			},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "bucket ID not found",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "failed_parsing_timestamp",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data: []byte(`{
									"metadata": {
									  "github-commit": "test-github-commit",
									  "github-workflow-triggered-timestamp": "2023",
									  "github-workflow-sha": "test-workflow-sha",
									  "github-workflow": "test-workflow",
									  "github-repo": "test-github-repo",
									  "github-run-id": "5050509831",
									  "github-run-attempt": "1"
									}
								  }`),
			},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "failed to parse date",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "missing_object_id",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo"},
				Data:       testGCSMetadataBytes(),
			},
			successMessenger: &testMessenger{},
			wantErrSubstr:    "object ID not found",
		},
		{
			name: "bucket_not_exist",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo2", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       testGCSMetadataBytes(),
			},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "failed to create GCS object reader",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "invalid_yaml_format",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(`foo, bar`),
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "failed to unmarshal object yaml",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "invalid_object_metadata",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       []byte("}"),
			},
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "failed to parse metadata",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "failed_process_not_processErr",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			processors: []Processor[*structpb.Struct]{&testProcessor{fmt.Errorf("always fail")}},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantErrSubstr: "always fail",
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "failed_process_with_processErr",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			processors: []Processor[*structpb.Struct]{&testProcessor{pmaperrors.New("user facing error")}},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			wantPmapEvent: &v1alpha1.PmapEvent{},
		},
		{
			name: "failed_process_and_send_with_processErr",
			notification: &pubsub.Message{
				Attributes: map[string]string{"bucketId": "foo", "objectId": "pmap-test/gh-prefix/dir1/dir2/bar", "payloadFormat": "JSON_API_V1"},
				Data:       testGCSMetadataBytes(),
			},
			gcsObjectBytes: []byte(`foo: bar
isOK: true`),
			processors: []Processor[*structpb.Struct]{&testProcessor{pmaperrors.New("user facing error")}},
			successMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
			},
			failureMessenger: &testMessenger{
				gotPmapEvent: &v1alpha1.PmapEvent{},
				returnErr:    fmt.Errorf("always fail"),
			},
			wantErrSubstr: "failed to send failure event downstream",
			wantPmapEvent: &v1alpha1.PmapEvent{},
			wantFailuerPmapEvent: &v1alpha1.PmapEvent{
				GithubSource: &v1alpha1.GitHubSource{
					RepoName:                   "test-github-repo",
					Commit:                     "test-github-commit",
					Workflow:                   "test-workflow",
					WorkflowSha:                "test-workflow-sha",
					WorkflowTriggeredTimestamp: timestamppb.New(time.Date(2023, time.April, 25, 17, 44, 57, 0, time.UTC)),
					WorkflowRunId:              "5050509831",
					WorkflowRunAttempt:         1,
					FilePath:                   "dir1/dir2/bar",
				},
			},
			wantAttr: map[string]string{
				AttrKeyProcessErr: "failed to process object: pmap process err: user facing error",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Create fake http client for storage client.
			hc, done := newTestServer(testHandleObjectRead(t, tc.gcsObjectBytes))
			defer done()

			// Setup test handler with fake storage client.
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}
			opts := []Option{
				WithStorageClient(c),
				WithFailureMessenger(tc.failureMessenger),
			}
			h, err := NewHandler(ctx, tc.processors, tc.successMessenger, opts...)
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}

			// Run test.
			gotErr := h.Handle(ctx, *tc.notification)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}

			cmpOpts := []cmp.Option{
				protocmp.Transform(),
				protocmp.IgnoreFields(&v1alpha1.PmapEvent{}, "payload"),
			}
			if diff := cmp.Diff(tc.wantPmapEvent, tc.successMessenger.getPmapEvent(), cmpOpts...); diff != "" {
				t.Errorf("successMessenger got unexpected pmapEvent diff (-want, +got):\n%s", diff)
			}
			if tc.failureMessenger != nil {
				if diff := cmp.Diff(tc.wantFailuerPmapEvent, tc.failureMessenger.getPmapEvent(), cmpOpts...); diff != "" {
					t.Errorf("failureMessenger got unexpected pmapEvent diff (-want, +got):\n%s", diff)
				}

				if diff := cmp.Diff(tc.wantAttr, tc.failureMessenger.getAttr()); diff != "" {
					t.Errorf("failureMessenger got unexpected attribute diff (-want, +got):\n%s", diff)
				}
			}
		})
	}
}

// Creates a fake http client.
func newTestServer(handler func(w http.ResponseWriter, r *http.Request)) (*http.Client, func()) {
	ts := httptest.NewTLSServer(http.HandlerFunc(handler))
	// Need insecure TLS option for testing.
	// #nosec G402
	tlsConf := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{
		TLSClientConfig: tlsConf,
		DialTLS: func(netw, addr string) (net.Conn, error) {
			return tls.Dial("tcp", ts.Listener.Addr().String(), tlsConf)
		},
	}
	return &http.Client{Transport: tr}, func() {
		tr.CloseIdleConnections()
		ts.Close()
	}
}

// Returns fake metadata that include github resource info.
func testGCSMetadataBytes() []byte {
	return []byte(`{
		"metadata": {
		  "github-commit": "test-github-commit",
		  "github-workflow-triggered-timestamp": "2023-04-25T17:44:57+00:00",
		  "github-workflow-sha": "test-workflow-sha",
		  "github-workflow": "test-workflow",
		  "github-repo": "test-github-repo",
		  "github-run-id": "5050509831",
		  "github-run-attempt": "1"
		}
	  }`)
}

// Returns a fake http func that writes the data in http response.
func testHandleObjectRead(t *testing.T, data []byte) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// This is for getting object info
		case "/foo/pmap-test/gh-prefix/dir1/dir2/bar":
			_, err := w.Write(data)
			if err != nil {
				t.Fatalf("failed to write response for object info: %v", err)
			}
		default:
			http.Error(w, "injected error", http.StatusNotFound)
		}
	}
}

func testToJSON(tb testing.TB, in any) []byte {
	tb.Helper()

	b, err := json.Marshal(in)
	if err != nil {
		tb.Fatal(err)
	}

	return b
}

type testProcessor struct {
	returnErr error
}

func (p *testProcessor) Process(_ context.Context, m *structpb.Struct) error {
	if p.returnErr == nil {
		m.Fields["processed"] = structpb.NewBoolValue(true)
		return nil
	}
	return p.returnErr
}

type testMessenger struct {
	gotPmapEvent *v1alpha1.PmapEvent
	gotAttr      map[string]string
	returnErr    error
}

func (m *testMessenger) Send(_ context.Context, data []byte, attr map[string]string) error {
	if m == nil {
		return nil
	}
	err := protojson.Unmarshal(data, m.gotPmapEvent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to PmapEvent: %w", err)
	}
	m.gotAttr = attr
	return m.returnErr
}

func (m *testMessenger) getPmapEvent() *v1alpha1.PmapEvent {
	return m.gotPmapEvent
}

func (m *testMessenger) getAttr() map[string]string {
	return m.gotAttr
}
