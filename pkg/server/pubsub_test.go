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

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := testNewPubSubGrpcConn(t, tc.pubSubServerOption)
			testTopic := testCreatePubsubTopic(ctx, t, serverProjectID, serverTopicID, option.WithGRPCConn(conn))

			msger := NewPubSubMessenger(testTopic)

			eventBytes, err := protojson.Marshal(tc.event)
			if err != nil {
				t.Fatalf("failed to marshal event to byte: %v", err)
			}

			gotErr := msger.Send(ctx, eventBytes, map[string]string{})
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
		})
	}
}

// Creates a GRPC connection with PubSub test server. Note that the GRPC connection is not closed at the end because
// it is duplicative if the PubSub client is also closing. Please remember to close the connection if the PubSub client
// will not close.
func testNewPubSubGrpcConn(t *testing.T, opts ...pstest.ServerReactorOption) *grpc.ClientConn {
	t.Helper()

	// Create PubSub test server.
	svr := pstest.NewServer(opts...)
	t.Cleanup(func() {
		if err := svr.Close(); err != nil {
			t.Fatalf("failed to cleanup test PubSub server: %v", err)
		}
	})

	// Connect to the server without using TLS.
	conn, err := grpc.NewClient(svr.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("fail to connect to test PubSub server: %v", err)
	}

	return conn
}

func testCreatePubsubTopic(ctx context.Context, t *testing.T, projectID, topicID string, opts ...option.ClientOption) *pubsub.Topic {
	t.Helper()

	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		t.Fatalf("failed to create pubsub client: %v", err)
	}
	if _, err := client.CreateTopic(ctx, serverTopicID); err != nil {
		t.Fatalf("failed to create test PubSub topic: %v", err)
	}
	topic := client.Topic(topicID)

	t.Cleanup(func() {
		topic.Stop()
		if err := client.Close(); err != nil {
			t.Logf("failed to close pubsub client: %v", err)
		}
	})

	return topic
}
