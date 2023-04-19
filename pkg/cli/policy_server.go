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
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/server"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ cli.Command = (*PolicyServerCommand)(nil)

type PolicyServerCommand struct {
	cli.BaseCommand

	cfg *server.HandlerConfig

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option
}

func (c *PolicyServerCommand) Desc() string {
	return `Start an Policy server`
}

func (c *PolicyServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Start a Policy server.
`
}

func (c *PolicyServerCommand) Flags() *cli.FlagSet {
	c.cfg = &server.HandlerConfig{}
	set := cli.NewFlagSet(c.testFlagSetOpts...)
	return c.cfg.ToFlags(set)
}

func (c *PolicyServerCommand) Run(ctx context.Context, args []string) error {
	srv, handler, closer, err := c.RunUnstarted(ctx, args)
	if err != nil {
		return err
	}
	defer closer()

	return srv.StartHTTPHandler(ctx, handler)
}

func (c *PolicyServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, http.Handler, func(), error) {
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

	successMessenger, err := server.NewPubSubMessenger(ctx, c.cfg.ProjectID, c.cfg.SuccessTopicID)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create success event messenger: %w", err)
	}
	handler, err := server.NewHandler(ctx, []server.Processor[*structpb.Struct]{}, successMessenger)
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