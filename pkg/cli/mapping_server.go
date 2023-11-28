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

package cli

import (
	"context"
	"fmt"
	"net/http"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/pubsub"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/multicloser"
	"github.com/abcxyz/pkg/serving"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/mapping/processors"
	"github.com/abcxyz/pmap/pkg/server"
)

var _ cli.Command = (*MappingServerCommand)(nil)

type MappingServerCommand struct {
	cli.BaseCommand

	cfg *server.MappingHandlerConfig
}

func (c *MappingServerCommand) Desc() string {
	return `Start an Mapping server. Mapping server provides structured data annotation and metadata discovery.`
}

func (c *MappingServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Start a Mapping server. Mapping server provides structured data annotation and metadata discovery.
`
}

func (c *MappingServerCommand) Flags() *cli.FlagSet {
	c.cfg = &server.MappingHandlerConfig{}
	set := c.NewFlagSet()
	return c.cfg.ToFlags(set)
}

func (c *MappingServerCommand) Run(ctx context.Context, args []string) error {
	logger := logging.FromContext(ctx)

	srv, handler, closer, err := c.RunUnstarted(ctx, args)
	defer func() {
		if err := closer.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close", "error", err)
		}
	}()
	if err != nil {
		return err
	}

	return srv.StartHTTPHandler(ctx, handler)
}

func (c *MappingServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, http.Handler, *multicloser.Closer, error) {
	var closer *multicloser.Closer

	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return nil, nil, closer, fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return nil, nil, closer, fmt.Errorf("unexpected arguments: %q", args)
	}

	logger := logging.FromContext(ctx)
	logger.DebugContext(ctx, "server starting",
		"commit", version.Commit,
		"version", version.Version)

	if err := c.cfg.Validate(); err != nil {
		return nil, nil, closer, fmt.Errorf("invalid mapping configuration: %w", err)
	}
	logger.DebugContext(ctx, "loaded configuration", "config", c.cfg)

	pubsubClient, err := pubsub.NewClient(ctx, c.cfg.ProjectID)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create pubsub client: %w", err)
	}
	closer = multicloser.Append(closer, pubsubClient.Close)

	successTopic := pubsubClient.Topic(c.cfg.SuccessTopicID)
	successMessenger := server.NewPubSubMessenger(successTopic)
	failureTopic := pubsubClient.Topic(c.cfg.FailureTopicID)
	failureMessenger := server.NewPubSubMessenger(failureTopic)
	closer = multicloser.Append(closer, successTopic.Stop, failureTopic.Stop)

	assetClient, err := asset.NewClient(ctx)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create the assetClient: %w", err)
	}
	closer = multicloser.Append(closer, assetClient.Close)

	processor, err := processors.NewAssetInventoryProcessor(ctx, assetClient, c.cfg.DefaultResourceScope)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create assetInventoryProcessor: %w", err)
	}

	handler, err := server.NewHandler(ctx,
		[]server.Processor[*v1alpha1.ResourceMapping]{processor},
		successMessenger,
		server.WithFailureMessenger(failureMessenger))
	if err != nil {
		return nil, nil, closer, fmt.Errorf("server.NewHandler: %w", err)
	}

	srv, err := serving.New(c.cfg.Port)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create serving infrastructure: %w", err)
	}

	return srv, handler.HTTPHandler(), closer, nil
}
