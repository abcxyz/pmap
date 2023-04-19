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

	"github.com/abcxyz/pkg/cfgloader"
	"github.com/abcxyz/pkg/cli"
	"github.com/sethvargo/go-envconfig"
)

// HandlerConfig defines the set over environment variables required
// for running this application.
type HandlerConfig struct {
	Port           string `env:"PORT,default=8080"`
	ProjectID      string `env:"PROJECT_ID,required"`
	SuccessTopicID string `env:"SUCCESS_TOPIC_ID,required"`
	FailureTopicID string `env:"FAILURE_TOPIC_ID"`
}

// Validate validates the handler config after load.
func (cfg *HandlerConfig) Validate() error {
	if cfg.ProjectID == "" {
		return fmt.Errorf("PROJECT_ID is empty and requires a value")
	}

	if cfg.SuccessTopicID == "" {
		return fmt.Errorf("SUCCESS_TOPIC_ID is empty and requires a value")
	}

	return nil
}

// NewConfig creates a new HandlerConfig from environment variables.
func NewConfig(ctx context.Context) (*HandlerConfig, error) {
	var cfg HandlerConfig
	err := cfgloader.Load(ctx, &cfg, cfgloader.WithLookuper(envconfig.OsLookuper()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}
	return &cfg, nil
}

// ToFlags binds the config to the give [cli.FlagSet] and returns it.
func (cfg *HandlerConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
	// Command options
	f := set.NewSection("COMMON SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "port",
		Target:  &cfg.Port,
		EnvVar:  "PORT",
		Default: "8080",
		Usage:   `The port the server listens to.`,
	})

	f.StringVar(&cli.StringVar{
		Name:   "project-id",
		Target: &cfg.ProjectID,
		EnvVar: "PROJECT_ID",
		Usage:  `Google Cloud project ID.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "success-topic-id",
		Target:  &cfg.SuccessTopicID,
		EnvVar:  "SUCCESS_TOPIC_ID",
		Example: "test-success-topic",
		Usage:   "The topic id which handles the resources that are processed successfully",
	})

	f.StringVar(&cli.StringVar{
		Name:    "failure-topic-id",
		Target:  &cfg.FailureTopicID,
		EnvVar:  "FAILURE_TOPIC_ID",
		Example: "test-failure-topic",
		Usage:   "The topic id which handles the resources that failed to process",
	})

	return set
}
