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
	"fmt"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

// SupportedResourceScope defines the scopes that are supported
// when doing resources searching.
var SupportedResourceScope = map[string]struct{}{
	"projects":      {},
	"folders":       {},
	"organizations": {},
}

// HandlerConfig defines the set over environment variables required
// for running this application.
type HandlerConfig struct {
	Port           string `env:"PORT,default=8080"`
	ProjectID      string `env:"PROJECT_ID,required"`
	SuccessTopicID string `env:"PMAP_SUCCESS_TOPIC_ID,required"`
	// FailureTopicID is optional for policy service
	FailureTopicID string `env:"PMAP_FAILURE_TOPIC_ID"`
	// MappingConfig is special config for mapping service
	MappingConfig MappingConfig
}

// MappingConfig defines the extra environment variables required
// for running mapping service.
type MappingConfig struct {
	// DefaultResourceScope is required for mapping service
	DefaultResourceScope string `env:"PMAP_RESOURCE_SCOPE"`
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
func (cfg *HandlerConfig) ValidateMappingConfig() error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if cfg.MappingConfig.DefaultResourceScope == "" {
		return fmt.Errorf(`PMAP_RESOURCE_SCOPE is empty and require a value from one of the following format:\n
			projects/{PROJECT_ID}\n
			projects/{PROJECT_NUMBER}\n
			folders/{FOLDER_NUMBER}\n
			organizations/{ORGANIZATION_NUMBER}\n`)
	}

	scope := strings.Split(cfg.MappingConfig.DefaultResourceScope, "/")[0]
	if _, ok := SupportedResourceScope[scope]; !ok {
		return fmt.Errorf(`PMAP_RESOURCE_SCOPE: %s doesn't have a valid value, the ResourceScope should be one of the following formats:\n
		projects/{PROJECT_ID}\n
		projects/{PROJECT_NUMBER}\n
		folders/{FOLDER_NUMBER}\n
		organizations/{ORGANIZATION_NUMBER}\n`, cfg.MappingConfig.DefaultResourceScope)
	}
	return nil
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

	f.StringVar(&cli.StringVar{
		Name:    "default-resource-scope",
		Target:  &cfg.MappingConfig.DefaultResourceScope,
		EnvVar:  "PMAP_RESOURCE_SCOPE",
		Example: "{projects/test-project-id}",
		Usage: `The default scope to search for resources. Format: 
			projects/{PROJECT_ID}\n
			projects/{PROJECT_NUMBER}\n
			folders/{FOLDER_NUMBER}\n
			organizations/{ORGANIZATION_NUMBER}`,
	})

	return set
}
