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
	"fmt"
	"testing"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
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
		event              *v1alpha1.PmapEvent
		wantErrSubstr      string
	}{
		{
			name:  "success",
			event: &v1alpha1.PmapEvent{},
		},
		{
			name:               "error_send_event",
			pubSubServerOption: pstest.WithErrorInjection("Publish", codes.NotFound, injectedPublishError),
			event:              &v1alpha1.PmapEvent{},
			wantErrSubstr:      injectedPublishError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := newTestPubSubGrpcConn(ctx, t, tc.pubSubServerOption)
			testTopic, err := createTestMessangerTopic(ctx, serverProjectID, serverTopicID, option.WithGRPCConn(conn))
			if err != nil {
				t.Fatalf("failed to create new pubsub client and topic: %v", err)
			}
			msger := NewPubSubMessenger(testTopic)
			if err != nil {
				t.Fatalf("failed to create new PubSubMessenger: %v", err)
			}

			eventByte, err := protojson.Marshal(tc.event)
			if err != nil {
				t.Errorf("failed to marshal event to byte: %v", err)
			}
			if err != nil {
				t.Errorf("%v", err)
			}
			gotErr := msger.Send(ctx, eventByte, map[string]string{})
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}

			testTopic.Stop()
		})
	}
}

// Creates a GRPC connection with PubSub test server. Note that the GRPC connection is not closed at the end because
// it is duplicative if the PubSub client is also closing. Please remember to close the connection if the PubSub client
// will not close.
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

func createTestMessangerTopic(ctx context.Context, projectID, topicID string, opts ...option.ClientOption) (*pubsub.Topic, error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pubsub client: %w", err)
	}
	if _, err := client.CreateTopic(ctx, serverTopicID); err != nil {
		return nil, fmt.Errorf("failed to create test PubSub topic: %w", err)
	}
	topic := client.Topic(topicID)
	return topic, nil
}
