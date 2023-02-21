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
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/pkg/testutil"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"

	pkgtestutil "github.com/abcxyz/pkg/testutil"
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

			conn := testutil.TestPubSubGrpcConn(ctx, t, tc.pubSubServerOption)
			msger, err := NewPubSubMessenger(ctx, serverProjectID, serverTopicID, option.WithGRPCConn(conn))
			if err != nil {
				t.Fatalf("failed to create new PubSubMessenger: %v", err)
			}

			// Create the test topic.
			if _, err := msger.client.CreateTopic(ctx, serverTopicID); err != nil {
				t.Fatalf("failed to create test PubSub topic: %v", err)
			}

			gotErr := msger.Send(ctx, tc.event)
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if err := msger.Cleanup(); err != nil {
				t.Errorf("Process(%+v) failed to cleanup: %v", tc.name, err)
			}
		})
	}
}
