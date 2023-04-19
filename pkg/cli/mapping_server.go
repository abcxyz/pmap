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

// Package cli implements the commands for the PMAP CLI.

package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/serving"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/mapping/processors"
	"github.com/abcxyz/pmap/pkg/server"
)

var _ cli.Command = (*MappingServerCommand)(nil)

type MappingServerCommand struct {
	cli.BaseCommand

	cfg *server.HandlerConfig

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option
}

func (c *MappingServerCommand) Desc() string {
	return `Start an Mapping server`
}

func (c *MappingServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Start a Mapping server.
`
}

func (c *MappingServerCommand) Flags() *cli.FlagSet {
	c.cfg = &server.HandlerConfig{}
	set := cli.NewFlagSet(c.testFlagSetOpts...)
	return c.cfg.ToFlags(set)
}

func (c *MappingServerCommand) Run(ctx context.Context, args []string) error {
	srv, handler, closer, err := c.RunUnstarted(ctx, args)
	if err != nil {
		return err
	}
	defer closer()

	return srv.StartHTTPHandler(ctx, handler)
}

func (c *MappingServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, http.Handler, func(), error) {
	closer := func() {}

	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return nil, nil, closer, fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return nil, nil, closer, fmt.Errorf("unexpected arguments: %q", args)
	}

	logger := logging.FromContext(ctx)
	logger.Debugw("server starting",
		"commit", version.Commit,
		"version", version.Version)

	if err := c.cfg.Validate(); err != nil {
		return nil, nil, closer, fmt.Errorf("invalid configuration: %w", err)
	}
	logger.Debugw("loaded configuration", "config", c.cfg)

	if c.cfg.FailureTopicID == "" {
		return nil, nil, closer, fmt.Errorf("missing FAILURE_TOPIC_ID in config")
	}

	successMessenger, err := server.NewPubSubMessenger(ctx, c.cfg.ProjectID, c.cfg.SuccessTopicID)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create success event messenger: %w", err)
	}

	failureMessenger, err := server.NewPubSubMessenger(ctx, c.cfg.ProjectID, c.cfg.FailureTopicID)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create failure event messenger: %w", err)
	}
	processor, err := processors.NewAssetInventoryProcessor(ctx, fmt.Sprintf("projects/%s", c.cfg.ProjectID))
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create asset inventory processor: %w", err)
	}
	handler, err := server.NewHandler(ctx,
		[]server.Processor[*v1alpha1.ResourceMapping]{processor},
		successMessenger,
		server.WithFailureMessenger(failureMessenger))
	if err != nil {
		return nil, nil, closer, fmt.Errorf("server.NewHandler: %w", err)
	}
	closer = func() {
		if err := handler.Cleanup(); err != nil {
			logger.Errorw("failed to close clean up handler", "error", err)
		}
	}

	srv, err := serving.New(c.cfg.Port)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create serving infrastructure: %w", err)
	}
	return srv, handler.HTTPHandler(), closer, nil
}