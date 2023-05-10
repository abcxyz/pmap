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

	"github.com/abcxyz/pkg/testutil"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateResouceMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		expErr string
		data   *ResourceMapping
	}{
		{
			name:   "invalid_email",
			expErr: "invalid owner",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &Contacts{
					Email: []string{"invalid.example.com"},
				},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"location": structpb.NewStringValue("global"),
					},
				},
			},
		},
		{
			name:   "empty_resource_provider",
			expErr: "empty resource provider",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &Contacts{
					Email: []string{"pmap@example.com"},
				},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"location": structpb.NewStringValue("global"),
					},
				},
			},
		},
		{
			name:   "assetInfo_included_as_custom_key",
			expErr: "reserved key is included: assetInfo",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &Contacts{
					Email: []string{"pmap@example.com"},
				},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						AnnotationKeyAssetInfo: structpb.NewStringValue("global"),
					},
				},
			},
		},
		{
			name: "success",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &Contacts{
					Email: []string{"pmap@example.com"},
				},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"location": structpb.NewStringValue("global"),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateResourceMapping(tc.data)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Errorf("ValidateResourceMapping got unexpected error: %s", diff)
			}
		})
	}
}
