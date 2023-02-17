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

// Package testutil provides utilities that are intended to enable easier
// and more concise writing of unit test code.
package testutil

import (
	"context"
	"testing"

	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Creates a GRPC connection with PubSub test server. Note that the GRPC connection is not closed at the end because
// it is duplicative if the PubSub client is also closing. Please remember to close the connection if the PubSub client
// will not close.
func TestPubSubGrpcConn(ctx context.Context, t *testing.T, opts ...pstest.ServerReactorOption) *grpc.ClientConn {
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
