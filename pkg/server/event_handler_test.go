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
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEventHandler_HttpHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name               string
		requestBody        []byte
		responseData       []byte
		wantStatusCode     int
		wantRespBodySubstr string
	}{
		{
			name: "success",
			requestBody: []byte(`
			{
				"message": {
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			responseData: []byte(`foo: bar
isOK: true`),
			wantStatusCode:     http.StatusCreated,
			wantRespBodySubstr: "OK",
		},
		{
			name:               "invalid_request_body",
			requestBody:        []byte(`}`),
			responseData:       nil,
			wantStatusCode:     http.StatusBadRequest,
			wantRespBodySubstr: "invalid character",
		},
		{
			name: "failed_handle_event",
			requestBody: []byte(`
			{
				"message": {
					"attributes": {
						"bucketId": "foo",
						"objectId": "bar2"
					}
				},
				"subscription": "test_subscription"
			}
			`),
			wantStatusCode:     http.StatusInternalServerError,
			wantRespBodySubstr: "failed to get GCS object",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup fake storage client.
			hc, done := newTestServer(handleObjectRead(t, tc.responseData))
			defer done()
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}

			h, err := NewHandler(ctx, []Processor[*structpb.Struct]{&successProcessor{}}, WithClient[structpb.Struct](c))
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.requestBody))
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
		name          string
		message       PubSubMessage
		responseData  []byte
		processors    []Processor[*structpb.Struct]
		wantErrSubstr string
	}{
		{
			name: "success",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar"},
				},
			},
			responseData: []byte(`foo: bar
isOK: true`),
			processors: []Processor[*structpb.Struct]{&successProcessor{}},
		},
		{
			name: "missing_bucket_id",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"objectId": "bar"},
				},
			},
			wantErrSubstr: "bucket ID not found",
		},
		{
			name: "missing_object_id",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo"},
				},
			},
			wantErrSubstr: "object ID not found",
		},
		{
			name: "bucket_not_exist",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo2", "objectId": "bar"},
				},
			},
			wantErrSubstr: "failed to create GCS object reader",
		},
		{
			name: "object_not_exist",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar2"},
				},
			},
			wantErrSubstr: "failed to create GCS object reader",
		},
		{
			name: "invalid_yaml_format",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar"},
				},
			},
			responseData:  []byte(`foo, bar`),
			wantErrSubstr: "failed to unmarshal object yaml",
		},
		{
			name: "failed_process",
			message: PubSubMessage{
				Message: struct {
					Data []byte "json:\"data,omitempty\""
					Attributes map[string]string "json:\"attributes\""
				}{
					Attributes: map[string]string{"bucketId": "foo", "objectId": "bar"},
				},
			},
			responseData: []byte(`foo: bar
isOK: true`),
			processors:    []Processor[*structpb.Struct]{&failProcessor{}},
			wantErrSubstr: "failed to process object",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Create fake http client for storage client.
			hc, done := newTestServer(handleObjectRead(t, tc.responseData))
			defer done()

			// Setup test handler with fake storage client.
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}
			h, err := NewHandler(ctx, tc.processors, WithClient[structpb.Struct](c))
			if err != nil {
				t.Fatalf("failed to create event handler %v", err)
			}

			// Run test.
			gotErr := h.Handle(ctx, tc.message)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
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

// Returns a fake http func that writes the data in http response.
func handleObjectRead(t *testing.T, data []byte) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/foo/bar" {
			http.Error(w, "injected error", http.StatusNotFound)
		}
		_, err := w.Write(data)
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
