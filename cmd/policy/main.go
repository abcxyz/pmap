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

// Package main is the main entrypoint to the application.
package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/serving"
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/server"
	"google.golang.org/protobuf/types/known/structpb"
)

// main is the application entry point. It primarily wraps the realMain function with
// a context that properly handles signals from the OS.
func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	if err := realMain(ctx); err != nil {
		done()
		logger.Fatal(err)
	}
}

// realMain creates an HTTP server to receive GCS notifications
// via PubSub push messages.
// This server supports graceful stopping and cancellation by:
//   - using a cancellable context
//   - listening to incoming requests in a goroutine
func realMain(ctx context.Context) (runErr error) {
	logger := logging.FromContext(ctx)
	logger.Debugw("server starting",
		"commit", version.Commit,
		"version", version.Version)

	cfg, err := server.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("server.NewConfig: %w", err)
	}

	successMessenger, err := server.NewPubSubMessenger(ctx, cfg.ProjectID, cfg.SuccessTopicID)
	if err != nil {
		return fmt.Errorf("failed to create success event messenger: %w", err)
	}
	handler, err := server.NewHandler(ctx, []server.Processor[*structpb.Struct]{}, successMessenger)
	if err != nil {
		return fmt.Errorf("server.NewHandler: %w", err)
	}

	defer func() {
		if err := handler.Cleanup(); err != nil {
			runErr = fmt.Errorf("failed to clean up handler %w", err)
		}
	}()

	srv, err := serving.New(cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to create serving infrastructure: %w", err)
	}

	return srv.StartHTTPHandler(ctx, handler.HTTPHandler())
}
