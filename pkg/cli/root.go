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

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pmap/internal/version"
)

// rootCmd defines the starting command structure.
var rootCmd = func() cli.Command {
	return &cli.RootCommand{
		Name:    "pmap",
		Version: version.HumanVersion,
		Commands: map[string]cli.CommandFactory{
			"mapping": func() cli.Command {
				return &cli.RootCommand{
					Name:        "mapping",
					Description: "Perform operations related to mapping",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &MappingServerCommand{}
						},
					},
				}
			},
			"policy": func() cli.Command {
				return &cli.RootCommand{
					Name:        "policy",
					Description: "Perform operations related to policy",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &PolicyServerCommand{}
						},
					},
				}
			},
			"validate": func() cli.Command {
				return &ValidateCommand{}
			},
		},
	}
}

// Run executes the CLI.
func Run(ctx context.Context, args []string) error {
	return rootCmd().Run(ctx, args) //nolint:wrapcheck // Want passthrough
}
