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

	"cloud.google.com/go/pubsub/pstest"
	"github.com/abcxyz/pkg/testutil"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	injectedPublishError = "injected publish error"
	serverProjectID      = "test-project-id"
	serverTopicID        = "test-topic-id"
)

func TestPubSubMessenger_Send(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name               string
		pubSubServerOption pstest.ServerReactorOption
		message            []byte
		wantErrSubstr      string
	}{
		{
			name:    "success",
			message: []byte("test"),
		},
		{
			name:               "error_send_message",
			pubSubServerOption: pstest.WithErrorInjection("Publish", codes.NotFound, injectedPublishError),
			message:            []byte("test"),
			wantErrSubstr:      injectedPublishError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := newTestPubSubGrpcConn(ctx, t, tc.pubSubServerOption)
			msger, err := NewPubSubMessenger(ctx, serverProjectID, serverTopicID, option.WithGRPCConn(conn))
			if err != nil {
				t.Fatalf("failed to create new PubSubMessenger: %v", err)
			}

			// Create the test topic.
			if _, err := msger.client.CreateTopic(ctx, serverTopicID); err != nil {
				t.Fatalf("failed to create test PubSub topic: %v", err)
			}

			gotErr := msger.Send(ctx, tc.message)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if err := msger.Cleanup(); err != nil {
				t.Errorf("Process(%+v) failed to cleanup: %v", tc.name, err)
			}
		})
	}
}

// Creates a grpc connection with PubSub test server.
func newTestPubSubGrpcConn(ctx context.Context, t *testing.T, opts ...pstest.ServerReactorOption) *grpc.ClientConn {
	t.Helper()

	// Create PubSub test server.
	svr := pstest.NewServer(opts...)
	t.Cleanup(func() {
		if err := svr.Close(); err != nil {
			t.Fatalf("failed to cleanup test PubSub server: %v", err)
		}
	})

	// Connect to the server without using TLS.
	conn, err := grpc.DialContext(ctx, svr.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("fail to connect to test PubSub server: %v", err)
	}

	return conn
}
