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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEventHandler_Handle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name           string
		requestBody    []byte
		processors     []Processor[*structpb.Struct]
		wantStatusCode int
		wantRespBody   string
	}{
		{
			name: "success",
			requestBody: []byte(`
			{
				"message": {
					"data": "some_data",
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			processors:     []Processor[*structpb.Struct]{&successProcessor{}},
			wantStatusCode: http.StatusCreated,
			wantRespBody:   successMessage,
		},
		{
			name: "invalid_request_body",
			requestBody: []byte(`
				"message": {
					"data": "some_data",
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			processors:     []Processor[*structpb.Struct]{},
			wantStatusCode: http.StatusBadRequest,
			wantRespBody:   errUnmarshallingMessage,
		},
		{
			name: "failed_retrieve_gcs_object",
			requestBody: []byte(`
			{
				"message": {
					"data": "some_data",
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar2"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			processors:     []Processor[*structpb.Struct]{},
			wantStatusCode: http.StatusInternalServerError,
			wantRespBody:   errGettingGCSObject,
		},
		{
			name: "failed_process_gcs_object",
			requestBody: []byte(`
			{
				"message": {
					"data": "some_data",
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			processors:     []Processor[*structpb.Struct]{&failProcessor{}},
			wantStatusCode: http.StatusInternalServerError,
			wantRespBody:   errProcessingObject,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup fake storage client.
			hc, done := newTestServer(handleObjectReadYaml(t))
			defer done()
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}

			h, err := NewHandler(ctx, tc.processors, WithClient[structpb.Struct](c))
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.requestBody))
			resp := httptest.NewRecorder()
			h.Handle().ServeHTTP(resp, req)

			if resp.Code != tc.wantStatusCode {
				t.Errorf("Process %+v: StatusCode got: %d want: %d", tc.name, resp.Code, tc.wantStatusCode)
			}

			if strings.TrimSpace(resp.Body.String()) != tc.wantRespBody {
				t.Errorf("Process %+v: ResponseBody got: %s want: %s", tc.name, resp.Body.String(), tc.wantRespBody)
			}
		})
	}
}

func TestEventHandler_getGCSObjectProto(t *testing.T) {
	t.Parallel()

	// A nil struct used for testing.
	var nilStruct *structpb.Struct

	cases := []struct {
		name            string
		message         PubSubMessage
		invalidYaml     bool
		wantObjectProto proto.Message
		wantErrSubstr   string
	}{
		{
			name: "success",
			message: PubSubMessage{
				Message: struct {
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar"},
				},
			},
			invalidYaml: false,
			wantObjectProto: &structpb.Struct{
				Fields: map[string]*structpb.Value{"foo": structpb.NewStringValue("bar"), "isOK": structpb.NewBoolValue(true)},
			},
		},
		{
			name: "bucket_not_exist",
			message: PubSubMessage{
				Message: struct {
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo2", "objectId": "bar"},
				},
			},
			invalidYaml:     false,
			wantObjectProto: nilStruct,
			wantErrSubstr:   "failed to create GCS object reader",
		},
		{
			name: "object_not_exist",
			message: PubSubMessage{
				Message: struct {
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar2"},
				},
			},
			invalidYaml:     false,
			wantObjectProto: nilStruct,
			wantErrSubstr:   "failed to create GCS object reader",
		},
		{
			name: "invalid_yaml_format",
			message: PubSubMessage{
				Message: struct {
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar"},
				},
			},
			invalidYaml:     true,
			wantObjectProto: nilStruct,
			wantErrSubstr:   "failed to unmarshal object yaml",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Create fake http client for storage client.
			var fakeFunc func(w http.ResponseWriter, r *http.Request)
			if tc.invalidYaml {
				fakeFunc = handleObjectReadInvalidYaml(t)
			} else {
				fakeFunc = handleObjectReadYaml(t)
			}
			hc, done := newTestServer(fakeFunc)
			defer done()

			// Setup test handler with fake storage client.
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}
			h, err := NewHandler(ctx, []Processor[*structpb.Struct]{}, WithClient[structpb.Struct](c))
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}

			// Run test.
			gotP, gotErr := h.getGCSObjectProto(ctx, tc.message)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if diff := cmp.Diff(tc.wantObjectProto, gotP, protocmp.Transform()); diff != "" {
				t.Errorf("Process(%+v) got diff (-want, +got): %v", tc.name, diff)
			}
		})
	}
}

func TestUnmarshalYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		yb            []byte
		wantStruct    *structpb.Struct
		wantErrSubstr string
	}{
		{
			name: "success",
			yb: []byte(`foo: bar
isOK: true`),
			wantStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{"foo": structpb.NewStringValue("bar"), "isOK": structpb.NewBoolValue(true)},
			},
		},
		{
			name: "invalid_yaml",
			yb: []byte(`foo, bar,
isOK: true`),
			wantStruct:    &structpb.Struct{},
			wantErrSubstr: "failed to unmarshal yaml",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := &structpb.Struct{}

			// Run test.
			gotErr := unmarshalYAML(tc.yb, v)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if diff := cmp.Diff(tc.wantStruct, v, protocmp.Transform()); diff != "" {
				t.Errorf("Process(%+v) got diff (-want, +got): %v", tc.name, diff)
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

// Returns a fake http func that writes a valid yaml bytes in http response.
func handleObjectReadYaml(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/foo/bar" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
		_, err := w.Write([]byte(`foo: bar
isOK: true`))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}
}

// Returns a fake http func that writes an invalid yaml bytes in http response.
func handleObjectReadInvalidYaml(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/foo/bar" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
		_, err := w.Write([]byte(`foo, bar,
		isOK: true`))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}
}

type failProcessor struct{}

func (p *failProcessor) Process(_ context.Context, m *structpb.Struct) error {
	return fmt.Errorf("always fail")
}

type successProcessor struct{}

func (p *successProcessor) Process(_ context.Context, m *structpb.Struct) error {
	m.Fields["processed"] = structpb.NewBoolValue(true)
	return nil
}
