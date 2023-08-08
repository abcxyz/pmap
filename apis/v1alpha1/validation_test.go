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
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateResouceMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		expErr       string
		data         *ResourceMapping
		wantSubscope string
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
			name:         "success",
			wantSubscope: "parent/foo/child/bar?key1=value1&key2=value2",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
					Subscope: "parent/foo/child/bar?key1=value1&key2=value2",
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
			name: "empty_subscope_success",
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
		{
			name:   "invalid_subscope_url",
			expErr: "failed to parse subscope string",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
					Subscope: "parent/foo/child/\\\bar?key1=value1&key2=value2",
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
			name:         "invalid_query_string",
			expErr:       "failed to parse qualifier string",
			wantSubscope: "parent/foo/child/bar?key1=value1&;=value2",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
					Subscope: "parent/foo/child/bar?key1=value1&;=value2",
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
			name:         "nomalize_subscope",
			wantSubscope: "normalize?key1=value1",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
					Subscope: "NORMALIZE?KEY1=VALUE1",
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
			name:   "keys_not_sorted",
			expErr: "keys should be in alphabetical order",
			data: &ResourceMapping{
				Resource: &Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
					Subscope: "parent/foo/child/bar?key2=value2&key1=value1",
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
			if tc.wantSubscope != "" {
				if diff := cmp.Diff(tc.data.Resource.Subscope, tc.wantSubscope); diff != "" {
					t.Errorf("subscope normalization failed (-want, +got): %v", diff)
				}
			}
		})
	}
}
