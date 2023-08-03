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
	"errors"
	"fmt"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

// allowedScopes showes the scopes that are supported
// when doing resources searching.
// https://cloud.google.com/asset-inventory/docs/reference/rest/v1/TopLevel/searchAllResources#path-parameters
var allowedScopes = []string{
	"projects/{PROJECT_ID}",
	"projects/{PROJECT_NUMBER}",
	"folders/{FOLDER_NUMBER}",
	"organizations/{ORGNANIZATION_NUMBER}",
}

// HandlerConfig defines the set over environment variables required
// for running this application.
type HandlerConfig struct {
	Port           string `env:"PORT,default=8080"`
	ProjectID      string `env:"PROJECT_ID,required"`
	SuccessTopicID string `env:"PMAP_SUCCESS_TOPIC_ID,required"`
	// FailureTopicID is optional for policy service
	FailureTopicID string `env:"PMAP_FAILURE_TOPIC_ID"`
}

// MappingConfig defines the environment variables required
// for running mapping service.
type MappingHandlerConfig struct {
	// DefaultResourceScope is the default resource scope to search resources.
	// This is only used for global resources such as GCS bucket.
	DefaultResourceScope string `env:"PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE,required"`
	HandlerConfig
}

// Validate validates the handler config after load.
func (cfg *HandlerConfig) Validate() error {
	if cfg.ProjectID == "" {
		return fmt.Errorf("PROJECT_ID is empty and requires a value")
	}

	if cfg.SuccessTopicID == "" {
		return fmt.Errorf("PMAP_SUCCESS_TOPIC_ID is empty and requires a value")
	}

	return nil
}

// ValidateMappingConfig validates the handler config for mapping service after load.
func (cfg *MappingHandlerConfig) Validate() (retErr error) {
	if err := cfg.HandlerConfig.Validate(); err != nil {
		retErr = errors.Join(retErr, fmt.Errorf("invalid configuration: %w", err))
	}

	// For mapping server, we also require a failure topic ID.
	if cfg.HandlerConfig.FailureTopicID == "" {
		retErr = errors.Join(retErr, fmt.Errorf("PMAP_FAILURE_TOPIC_ID is empty and require a value for mapping service"))
	}

	if cfg.DefaultResourceScope == "" {
		retErr = errors.Join(retErr, fmt.Errorf(`PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE is empty, allowed values are: %v`, allowedScopes))
	}

	scope := strings.Split(cfg.DefaultResourceScope, "/")[0]
	switch scope {
	case "projects", "folders", "organizations":
		break
	default:
		retErr = errors.Join(retErr, fmt.Errorf(`PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE: %s is required in one of the formats: %v`, cfg.DefaultResourceScope, allowedScopes))
	}

	return retErr
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
		EnvVar:  "PMAP_SUCCESS_TOPIC_ID",
		Example: "test-success-topic",
		Usage:   "The topic id which handles the resources that are processed successfully.",
	})

	f.StringVar(&cli.StringVar{
		Name:    "failure-topic-id",
		Target:  &cfg.FailureTopicID,
		EnvVar:  "PMAP_FAILURE_TOPIC_ID",
		Example: "test-failure-topic",
		Usage:   "The topic id which handles the resources that failed to process.",
	})

	return set
}

func (cfg *MappingHandlerConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
	cfg.HandlerConfig.ToFlags(set)

	f := set.NewSection("MAPPING SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "default-resource-scope",
		Target:  &cfg.DefaultResourceScope,
		EnvVar:  "PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE",
		Example: "projects/test-project-id",
		Usage:   fmt.Sprintf(`The default scope to search for resources. Format: %v`, allowedScopes),
	})
	return set
}
