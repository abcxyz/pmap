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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestNewValidateCmd(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))
	td := t.TempDir()

	cases := []struct {
		name      string
		args      []string
		dir       string
		fileDatas map[string][]byte
		expOut    string
		expErr    string
	}{
		{
			name:   "unexpected_args",
			args:   []string{"foo"},
			expErr: `unexpected arguments: [foo]`,
		},
		{
			name:   "missing_path",
			args:   []string{},
			expErr: `path is required`,
		},
		{
			name: "invalid_yaml",
			dir:  "dir_invalid_yaml",
			fileDatas: map[string][]byte{
				"file1.yaml": []byte(`
		foo
		`),
			},
			args:   []string{"-path", filepath.Join(td, "dir_invalid_yaml")},
			expErr: "file \"file1.yaml\": failed to unmarshal yaml to ResourceMapping",
		},
		{
			name: "valid_contents",
			fileDatas: map[string][]byte{
				"file1.yaml": []byte(`
resource:
    provider: gcp
    name: //pubsub.googleapis.com/projects/test-project/topics/test-topic
contacts:
    email:
        - pmap@gmail.com
annotations:
    fields:
        location:
            kind:
                stringvalue: global
`),
				"file2.yml": []byte(`
resource:
    provider: gcp
    name: //pubsub.googleapis.com/projects/test-project/subscriptions/test-subsriptions
contacts:
    email:
        - pmap@gmail.com
annotations:
    fields:
        location:
            kind:
                stringvalue: global
`),
				"file3.txt": []byte(`
resource:
    provider: gcp
    name: //pubsub.googleapis.com/projects/test-project/subscriptions/test-subsriptions
contacts:
    email:
        - pmap@gmail.com
annotations:
    fields:
        location:
            kind:
                stringvalue: global
`),
			},
			dir:    "dir_valid_contents",
			args:   []string{"-path", filepath.Join(td, "dir_valid_contents")},
			expOut: "processing file \"file1.yaml\"\nprocessing file \"file2.yml\"\nValidation passed",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.dir != "" && tc.fileDatas != nil {
				if err := os.MkdirAll(filepath.Join(td, tc.dir), 0o755); err != nil {
					t.Fatal(err)
				}
				for name, data := range tc.fileDatas {
					if err := os.WriteFile(filepath.Join(td, tc.dir, name), data, 0o600); err != nil {
						t.Fatalf("failed to write data to file %s: %v", name, err)
					}
				}
			}

			var cmd MappingValidateCommand
			_, stdout, _ := cmd.Pipe()

			args := append([]string{}, tc.args...)

			if err := cmd.Run(ctx, args); err != nil {
				if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
					t.Fatal(diff)
				}
				if err != nil {
					return
				}
			}
			if diff := cmp.Diff(strings.TrimSpace(tc.expOut), strings.TrimSpace(stdout.String())); diff != "" {
				t.Errorf("output: diff (-want, +got):\n%s", diff)
			}
			if want, got := strings.TrimSpace(tc.expOut), strings.TrimSpace(stdout.String()); got != want {
				t.Errorf("got\n\n%s\n\nwant\n\n%s\n\n", got, want)
			}
		})
	}
}
