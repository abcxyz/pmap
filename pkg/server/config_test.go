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

	"github.com/abcxyz/pkg/testutil"
	"google.golang.org/api/option"
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

func TestConfig_CreateFailureMessenger(t *testing.T) {
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
				SuccessTopicID: testSuccessTopicID,
			},
			wantErrSubstr: "invalid configuration",
		},
		{
			name: "missing_failure_topic",
			cfg: &HandlerConfig{
				ProjectID:      testProjectID,
				SuccessTopicID: testSuccessTopicID,
			},
			wantErrSubstr: "FAILURE_TOPIC_ID is empty and requires a value",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Use fake PubSub grpc connection to create the messenger.
			msger, err := CreateFailureMessenger(ctx, tc.cfg, option.WithGRPCConn(newTestPubSubGrpcConn(ctx, t)))
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
