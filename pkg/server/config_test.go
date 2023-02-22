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
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/testutil"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	testProjectID      = "test-project-id"
	testSuccessTopicID = "test-success-topic-id"
	testFailureTopicID = "test-failure-topic-id"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *HandlerConfig
		wantErr string
	}{
		{
			name: "success",
			cfg: &HandlerConfig{
				Port:           "8080",
				ProjectID:      testProjectID,
				SuccessTopicID: testSuccessTopicID,
				FailureTopicID: testFailureTopicID,
			},
		},
		{
			name: "missing_project_id",
			cfg: &HandlerConfig{
				SuccessTopicID: testSuccessTopicID,
			},
			wantErr: `PROJECT_ID is empty and requires a value`,
		},
		{
			name: "missing_success_event_topic_id",
			cfg: &HandlerConfig{
				ProjectID: testProjectID,
			},
			wantErr: `SUCCESS_TOPIC_ID is empty and requires a value`,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Process(%+v) got unexpected err: %s", tc.name, diff)
			}
		})
	}
}

func TestConfig_FromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cfg           *HandlerConfig
		wantErrSubstr string
	}{
		{
			name: "success",
			cfg: &HandlerConfig{
				ProjectID:      testProjectID,
				SuccessTopicID: testSuccessTopicID,
				FailureTopicID: testFailureTopicID,
			},
		},
		{
			name:          "nil_config",
			wantErrSubstr: "nil config",
		},
		{
			name: "invalid_config",
			cfg: &HandlerConfig{
				SuccessTopicID: "test_topic",
			},
			wantErrSubstr: "invalid configuration",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Use fake PubSub grpc connection to create the messengers.
			o := FromConfig(tc.cfg, option.WithGRPCConn(newTestPubSubGrpcConn(ctx, t)))
			msger, err := NewPubSubMessenger(ctx, testProjectID, testSuccessTopicID, option.WithGRPCConn(newTestPubSubGrpcConn(ctx, t)))
			if err != nil {
				t.Fatalf("failed to create new PubSubMessenger: %v", err)
			}

			// Setup fake storage client.
			hc, done := newTestServer(handleObjectRead(t, nil))
			defer done()
			c, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
			if err != nil {
				t.Fatalf("failed to creat GCS storage client %v", err)
			}

			h, err := NewHandler(ctx, []Processor[*structpb.Struct]{&successProcessor{}}, msger, o, WithStorageClient(c))
			if diff := testutil.DiffErrString(err, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected err: %s", tc.name, diff)
			}
			if h != nil {
				if err := h.Cleanup(); err != nil {
					t.Fatalf("Process(%+v) failed to cleanup: %v", tc.name, err)
				}
			}
		})
	}
}

func TestConfig_CreateSuccessMessenger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cfg           *HandlerConfig
		wantErrSubstr string
	}{
		{
			name: "success",
			cfg: &HandlerConfig{
				ProjectID:      testProjectID,
				SuccessTopicID: testSuccessTopicID,
			},
		},
		{
			name:          "nil_config",
			wantErrSubstr: "nil config",
		},
		{
			name: "invalid_config",
			cfg: &HandlerConfig{
				SuccessTopicID: "test_topic",
			},
			wantErrSubstr: "invalid configuration",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Use fake PubSub grpc connection to create the messenger.
			msger, err := CreateSuccessMessenger(ctx, tc.cfg, option.WithGRPCConn(newTestPubSubGrpcConn(ctx, t)))
			if diff := testutil.DiffErrString(err, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected err: %s", tc.name, diff)
			}
			if msger != nil {
				if err := msger.Cleanup(); err != nil {
					t.Fatalf("Process(%+v) failed to cleanup: %v", tc.name, err)
				}
			}
		})
	}
}
