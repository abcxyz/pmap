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
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/cmd/util"
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/mapping/processors"
	"github.com/abcxyz/pmap/pkg/server"
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
func realMain(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Debugw("server starting",
		"commit", version.Commit,
		"version", version.Version)

	cfg, err := server.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("server.NewConfig: %w", err)
	}

	// Create GCS notification handler.
	opt := server.FromConfig(cfg)
	successMessenger, err := server.CreateSuccessMessenger(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create success event messenger: %w", err)
	}
	processor, err := processors.NewProcessor(ctx, fmt.Sprintf("projects/%s", cfg.ProjectID))
	if err != nil {
		return fmt.Errorf("failed to create asset inventory processor: %w", err)
	}
	handler, err := server.NewHandler(ctx,
		[]server.Processor[*v1alpha1.ResourceMapping]{processor},
		successMessenger,
		opt)
	if err != nil {
		return fmt.Errorf("server.NewHandler: %w", err)
	}

	// Run the http server with the handler.
	if err := util.Run(ctx, cfg.Port, handler); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}
	return nil
}
