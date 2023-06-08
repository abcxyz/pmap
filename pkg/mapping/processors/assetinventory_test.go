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

package processors

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	asset "cloud.google.com/go/asset/apiv1"
	v1 "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // "cloud.google.com/go/asset/apiv1" still uses v1.Policy(deprecated).
)

func TestParseProject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		resourceName  string
		wantProject   string
		wantErrSubstr string
	}{
		{
			name:         "success",
			resourceName: "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
			wantProject:  "projects/test-project",
		},
		{
			name:         "failure_with_no_project_exists_in_resource_name",
			resourceName: "//storage.googleapis.com/test-bucket",
			wantProject:  "",
		},
		{
			name:          "failure_with_invalid_resource_name",
			resourceName:  "//pubsub.googleapis.com/projects//test-project/topics/test-topic",
			wantProject:   "",
			wantErrSubstr: "invalid resource name: //pubsub.googleapis.com/projects//test-project/topics/test-topic",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotProject, gotErr := parseProject(tc.resourceName)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if diff := cmp.Diff(tc.wantProject, gotProject); diff != "" {
				t.Errorf("ParseProject(%+v) got diff (-want, +got): %v", tc.name, diff)
			}
		})
	}
}

type fakeAssetInventoryServer struct {
	assetpb.UnimplementedAssetServiceServer

	searchAllResourcesData   *assetpb.SearchAllResourcesResponse
	searchAllIamPoliciesData *assetpb.SearchAllIamPoliciesResponse
	searchAllResourcesErr    error
	searchAllIamPoliciesErr  error
}

func (s *fakeAssetInventoryServer) SearchAllResources(context.Context, *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
	return s.searchAllResourcesData, s.searchAllResourcesErr
}

func (s *fakeAssetInventoryServer) SearchAllIamPolicies(context.Context, *assetpb.SearchAllIamPoliciesRequest) (*assetpb.SearchAllIamPoliciesResponse, error) {
	return s.searchAllIamPoliciesData, s.searchAllIamPoliciesErr
}

func TestProcessor_UpdatedProcess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		server              *fakeAssetInventoryServer
		resourceMapping     *v1alpha1.ResourceMapping
		wantResourceMapping *v1alpha1.ResourceMapping
		wantErrSubstr       string
	}{
		{
			name: "success",
			server: &fakeAssetInventoryServer{
				searchAllResourcesData: &assetpb.SearchAllResourcesResponse{
					Results: []*assetpb.ResourceSearchResult{{
						Name:                   "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
						AssetType:              "pubsub.googleapis.com/Topic",
						Project:                "projects/0",
						Folders:                []string{"folders/0", "folders/1"},
						Organization:           "organizations/0",
						DisplayName:            "projects/test-project/topics/test-topic",
						Labels:                 map[string]string{"env": "dev"},
						Location:               "global",
						ParentAssetType:        "cloudresourcemanager.googleapis.com/Project",
						ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/test-project",
					}},
				},
				searchAllIamPoliciesData: &assetpb.SearchAllIamPoliciesResponse{
					Results: []*assetpb.IamPolicySearchResult{{
						Resource:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
						AssetType:    "pubsub.googleapis.com/Topic",
						Project:      "projects/0",
						Folders:      []string{"folders/0", "folders/1"},
						Organization: "organizations/0",
						//nolint:staticcheck // see import.
						Policy: &v1.Policy{
							//nolint:staticcheck // see import.
							Bindings: []*v1.Binding{{
								Role:    "roles/pubsub.publisher",
								Members: []string{"serviceAccount:test-service@gcp-sa-pubsub.iam.gserviceaccount.com"},
							}},
						},
					}},
				},
			},
			resourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"custom_key": structpb.NewStringValue("test-key"),
					},
				},
			},
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
				Annotations: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"custom_key": structpb.NewStringValue("test-key"),
						v1alpha1.AnnotationKeyAssetInfo: structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"ancestors": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("organizations/0"), structpb.NewStringValue("folders/0"), structpb.NewStringValue("folders/1"), structpb.NewStringValue("projects/0")}}),
								"labels": structpb.NewStructValue(&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"env": structpb.NewStringValue("dev"),
									},
								}),
								"location": structpb.NewStringValue("global"),
								"iamPolicies": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStructValue(&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"bindings": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"members": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("serviceAccount:test-service@gcp-sa-pubsub.iam.gserviceaccount.com")}}),
												"role":    structpb.NewStringValue("roles/pubsub.publisher"),
											},
										})}}),
									},
								})}}),
							},
						}),
					},
				},
			},
		},
		{
			name: "failure_with_resources_search_err",
			server: &fakeAssetInventoryServer{
				searchAllResourcesErr: fmt.Errorf("encountered error during resources search: Internal Server Error"),
				searchAllIamPoliciesData: &assetpb.SearchAllIamPoliciesResponse{
					Results: []*assetpb.IamPolicySearchResult{{
						Resource:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
						AssetType:    "pubsub.googleapis.com/Topic",
						Project:      "projects/0",
						Folders:      []string{"folders/0", "folders/1"},
						Organization: "organizations/0",
						//nolint:staticcheck // see import.
						Policy: &v1.Policy{
							//nolint:staticcheck // see import.
							Bindings: []*v1.Binding{{
								Role:    "roles/pubsub.publisher",
								Members: []string{"serviceAccount:test-service@gcp-sa-pubsub.iam.gserviceaccount.com"},
							}},
						},
					}},
				},
			},
			resourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},
			wantErrSubstr: "encountered error during resources search: Internal Server Error",
		},
		{
			name: "failure_with_zero_matched_resource",
			server: &fakeAssetInventoryServer{
				searchAllResourcesData: &assetpb.SearchAllResourcesResponse{
					Results: []*assetpb.ResourceSearchResult{},
				},
			},
			resourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},

			wantErrSubstr: "0 matched resources found, expected 1 matched resource",
		},
		{
			name: "failure_with_policies_search_err",
			server: &fakeAssetInventoryServer{
				searchAllResourcesData: &assetpb.SearchAllResourcesResponse{
					Results: []*assetpb.ResourceSearchResult{{
						Name:                   "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
						AssetType:              "pubsub.googleapis.com/Topic",
						Project:                "projects/0",
						Folders:                []string{"folders/0", "folders/1"},
						Organization:           "organizations/0",
						DisplayName:            "projects/test-project/topics/test-topic",
						Labels:                 map[string]string{"env": "dev"},
						Location:               "global",
						ParentAssetType:        "cloudresourcemanager.googleapis.com/Project",
						ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/test-project",
					}},
				},
				searchAllIamPoliciesErr: fmt.Errorf("encountered error during iam policies search: Internal Server Error"),
			},
			resourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},
			wantResourceMapping: &v1alpha1.ResourceMapping{
				Resource: &v1alpha1.Resource{
					Provider: "gcp",
					Name:     "//pubsub.googleapis.com/projects/test-project/topics/test-topic",
				},
				Contacts: &v1alpha1.Contacts{Email: []string{"pmap@example.com"}},
			},
			wantErrSubstr: "encountered error during iam policies search: Internal Server Error",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Setup fake Asset Inventory server.
			addr, conn := testutil.FakeGRPCServer(t, func(s *grpc.Server) {
				assetpb.RegisterAssetServiceServer(s, tc.server)
			})

			// Setup fake Asset Inventory client.
			fakeAssetClient, err := asset.NewClient(ctx, option.WithGRPCConn(conn))
			if err != nil {
				t.Fatalf("creating client for fake at %q: %v", addr, err)
			}
			p, err := NewAssetInventoryProcessor(ctx, "projects/fake-project", fakeAssetClient)
			if err != nil {
				t.Fatalf("failed to create AssetInventoryProcessor: %v", err)
			}

			// Run test.
			gotErr := p.Process(ctx, tc.resourceMapping)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			// Verify that the ResourceMapping is modified with additional annotations fetched from Asset Inventory.
			if diff := cmp.Diff(tc.wantResourceMapping, tc.resourceMapping, protocmp.Transform()); diff != "" {
				t.Errorf("Process(%+v) got diff (-want, +got): %v", tc.name, diff)
			}
		})
	}
}
