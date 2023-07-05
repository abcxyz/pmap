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
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/multicloser"
	"github.com/abcxyz/pkg/serving"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/internal/version"
	"github.com/abcxyz/pmap/pkg/mapping/processors"
	"github.com/abcxyz/pmap/pkg/server"

	asset "cloud.google.com/go/asset/apiv1"
)

var _ cli.Command = (*MappingServerCommand)(nil)

type MappingServerCommand struct {
	cli.BaseCommand

	cfg *server.HandlerConfig

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option
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
	c.cfg = &server.HandlerConfig{}
	set := cli.NewFlagSet(c.testFlagSetOpts...)
	return c.cfg.ToFlags(set)
}

func (c *MappingServerCommand) Run(ctx context.Context, args []string) error {
	srv, handler, closer, err := c.RunUnstarted(ctx, args)
	defer func() {
		if err := closer.Close(); err != nil {
			logging.FromContext(ctx).Errorw("failed to close", "error", err)
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
	logger.Debugw("server starting",
		"commit", version.Commit,
		"version", version.Version)

	if err := c.cfg.Validate(); err != nil {
		return nil, nil, closer, fmt.Errorf("invalid configuration: %w", err)
	}
	logger.Debugw("loaded configuration", "config", c.cfg)

	// For mapping server, we also require a failure topic ID.
	if c.cfg.FailureTopicID == "" {
		return nil, nil, closer, fmt.Errorf("missing PMAP_FAILURE_TOPIC_ID in config")
	}

	if c.cfg.DefaultResourceScope == "" {
		return nil, nil, closer, fmt.Errorf(`PMAP_RESOURCE_SCOPE is empty and require a value from one of the following format:\n
			projects/{PROJECT_ID}\n
			projects/{PROJECT_NUMBER}\n
			folders/{FOLDER_NUMBER}\n
			organizations/{ORGANIZATION_NUMBER}\n`)
	}

	scope := strings.Split(c.cfg.DefaultResourceScope, "/")[0]
	if _, ok := server.SupportedResourceScope[scope]; !ok {
		return nil, nil, closer, fmt.Errorf(`PMAP_RESOURCE_SCOPE: %s doesn't have a valid value, the ResourceScope should be empty(default to project scopre) or one of the following formats:\n
		projects/{PROJECT_ID}\n
		projects/{PROJECT_NUMBER}\n
		folders/{FOLDER_NUMBER}\n
		organizations/{ORGANIZATION_NUMBER}\n`, c.cfg.DefaultResourceScope)
	}

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
