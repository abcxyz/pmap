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

package v1alpha1

import (
	"testing"

	"github.com/abcxyz/pkg/protoutil"
	"github.com/abcxyz/pkg/testutil"
)

func TestValidateResouceMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		expErr string
		data   []byte
	}{
		{
			name:   "invalid_email",
			expErr: "invalid owner",
			data: []byte(`
resource:
    provider: gcp
    name: //pubsub.googleapis.com/projects/test-project/topics/test-topic
contacts:
    email:
        - pmap.gmail.com
annotations:
    fields:
        location:
            kind:
                stringvalue: global
        `),
		},
		{
			name:   "empty_resource_provider",
			expErr: "empty resource provider",
			data: []byte(`
resource:
    provider: ""
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
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var resourceMapping ResourceMapping
			if err := protoutil.FromYAML(tc.data, &resourceMapping); err != nil {
				t.Errorf("failed to unmarshal data to ResourceMapping: %v", err)
			}
			if err := ValidateResourceMapping(&resourceMapping); err != nil {
				if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
