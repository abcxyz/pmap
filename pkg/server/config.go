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
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/api/option"
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

// CreateSuccessMessenger creates a success messenger with given context, config, and Google API client options.
func CreateSuccessMessenger(ctx context.Context, cfg *HandlerConfig, opts ...option.ClientOption) (Messenger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return NewPubSubMessenger(ctx, cfg.ProjectID, cfg.SuccessTopicID, opts...)
}

// CreateFailureMessenger creates a failure messenger with given context, config, and Google API client options.
func CreateFailureMessenger(ctx context.Context, cfg *HandlerConfig, opts ...option.ClientOption) (Messenger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if cfg.FailureTopicID == "" {
		return nil, fmt.Errorf("FAILURE_TOPIC_ID is empty and requires a value")
	}
	return NewPubSubMessenger(ctx, cfg.ProjectID, cfg.FailureTopicID, opts...)
}
