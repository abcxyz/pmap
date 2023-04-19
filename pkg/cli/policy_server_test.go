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
	"testing"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/sethvargo/go-envconfig"
)

func TestPolicyServerCommand(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name   string
		args   []string
		env    map[string]string
		expErr string
	}{
		{
			name:   "too_many_args",
			args:   []string{"foo"},
			expErr: `unexpected arguments: ["foo"]`,
		},
		{
			name: "invalid_config_missing_project_id",
			env: map[string]string{
				"SUCCESS_TOPIC_ID": "test_success_topic",
			},
			expErr: `invalid configuration: PROJECT_ID is empty and requires a value`,
		},
		{
			name: "invalid_config_missing_success_topic_id",
			env: map[string]string{
				"PROJECT_ID": "test_project",
			},
			expErr: `invalid configuration: SUCCESS_TOPIC_ID is empty and requires a value`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, done := context.WithCancel(ctx)
			defer done()

			var cmd MappingServerCommand
			cmd.testFlagSetOpts = []cli.Option{cli.WithLookupEnv(envconfig.MultiLookuper(
				envconfig.MapLookuper(tc.env),
				envconfig.MapLookuper(map[string]string{
					// Make the test choose a random port.
					"PORT": "0",
				}),
			).Lookup)}

			_, _, _ = cmd.Pipe()

			_, _, closer, err := cmd.RunUnstarted(ctx, tc.args)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			defer closer()
			if err != nil {
				return
			}
		})
	}
}