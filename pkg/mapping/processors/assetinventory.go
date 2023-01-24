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

// Package processors provides essential processors for pmap.
package processors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/structpb"

	asset "cloud.google.com/go/asset/apiv1"
	v1 "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // "cloud.google.com/go/asset/apiv1" still uses v1.Policy(deprecated).
)

const (
	gcpProvider = "gcp"
	pageSize    = 3
)

// AssetInventoryProcessor is the Cloud Asset Inventory validation and enrichment processor.
type AssetInventoryProcessor struct {
	// defaultResourceScope is used when there is no project found in the ResourceMapping.Resource.Name
	// A resourceScope can be a project, a folder, or an organization. The processor logic is limited to the resources within the scope.
	// See format and example here: https://cloud.google.com/asset-inventory/docs/reference/rest/v1/TopLevel/searchAllResources#path-parameters
	defaultResourceScope string
	client               *asset.Client
}

// Option is the option to set up a AssetInventoryProcessor.
type Option func(p *AssetInventoryProcessor) (*AssetInventoryProcessor, error)

// WithClient provides a asset client to the processor.
func WithClient(client *asset.Client) Option {
	return func(p *AssetInventoryProcessor) (*AssetInventoryProcessor, error) {
		p.client = client
		return p, nil
	}
}

// NewProcessor creates a new AssetInventoryProcessor with the given options.
// Need defaultResourceScope because resources such as GCS bucket won't include Project info in its resource name.
// See details: https://cloud.google.com/asset-inventory/docs/resource-name-format.
func NewProcessor(ctx context.Context, defaultResourceScope string, opts ...Option) (*AssetInventoryProcessor, error) {
	p := &AssetInventoryProcessor{defaultResourceScope: defaultResourceScope}
	for _, opt := range opts {
		var err error
		p, err = opt(p)
		if err != nil {
			return nil, fmt.Errorf("failed to apply client options: %w", err)
		}
	}

	if p.client == nil {
		client, err := asset.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create the Asset Inventory client: %w", err)
		}
		p.client = client
	}
	return p, nil
}

// Process validates the existence of resource associated with ResourceMapping,
// and enriches ResourceMapping with additional annotations such location, ancestors, etc.
// based on info fetched from Asset Inventory.
func (p *AssetInventoryProcessor) Process(ctx context.Context, resourceMapping *v1alpha1.ResourceMapping) error {
	logger := logging.FromContext(ctx)

	if resourceMapping.GetResource().GetProvider() != gcpProvider {
		// Skip non-GCP ResourceMapping
		logger.Debug("%T: skipping unsupported resource provider %q, want %q", p,
			resourceMapping.GetResource().GetProvider(), gcpProvider)
		return nil
	}

	resourceName := resourceMapping.GetResource().GetName()

	resourceScope, err := parseProject(resourceName)
	if err != nil {
		return err
	}
	// Need defaultResourceScope because resources such as GCS bucket won't include Project info in its resource name.
	// See details: https://cloud.google.com/asset-inventory/docs/resource-name-format.
	if resourceScope == "" {
		resourceScope = p.defaultResourceScope
	}

	additionalAnnos, err := p.validateAndEnrich(ctx, resourceScope, resourceName)
	if err != nil {
		return fmt.Errorf("failed to validate and enrich with resource %q in resourceScope %q: %w", resourceName, resourceScope, err)
	}

	mergedAnnos, err := mergeAnnotations(resourceMapping.GetAnnotations(), additionalAnnos)
	if err != nil {
		return err
	}

	resourceMapping.Annotations = mergedAnnos
	return nil
}

// validateAndEnrich validates the existence of resource associated with ResourceMapping,
// and return additional annotations such location, ancestors, etc.
// based on info fetched from Asset Inventory.
func (p *AssetInventoryProcessor) validateAndEnrich(ctx context.Context, resourceScope, resourceName string) (*structpb.Struct, error) {
	resourceSearchQuery := fmt.Sprintf("name=%s", resourceName)
	resourceSearchReq := &assetpb.SearchAllResourcesRequest{
		Scope:    resourceScope,
		Query:    resourceSearchQuery,
		PageSize: pageSize,
	}
	resource, err := p.getSingleResource(ctx, resourceSearchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get single matched resource: %w", err)
	}

	var ancestors []string

	if v := resource.GetOrganization(); v != "" {
		ancestors = append(ancestors, v)
	}
	if v := resource.GetFolders(); len(v) > 0 {
		ancestors = append(ancestors, v...)
	}
	if v := resource.GetProject(); v != "" {
		ancestors = append(ancestors, v)
	}

	iamSearchQuery := fmt.Sprintf("resource=%s", resourceName)
	iamSearchReq := &assetpb.SearchAllIamPoliciesRequest{
		Scope: resourceScope,
		Query: iamSearchQuery,
	}

	iamPolicies, err := p.getIAMPolicies(ctx, iamSearchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM policies with query %q resourceScope %q: %w", iamSearchQuery, resourceScope, err)
	}

	assetInventoryAnnos := map[string]any{}

	if len(ancestors) != 0 {
		assetInventoryAnnos["ancestors"] = ancestors
	}
	if resource.GetLocation() != "" {
		assetInventoryAnnos["location"] = resource.GetLocation()
	}
	if len(resource.GetLabels()) > 0 {
		assetInventoryAnnos["labels"] = resource.GetLabels()
	}
	if resource.GetCreateTime() != nil {
		assetInventoryAnnos["createTime"] = resource.GetCreateTime()
	}
	if len(resource.GetTagKeys()) > 0 {
		assetInventoryAnnos["tagKeys"] = resource.GetTagKeys()
	}
	if len(resource.GetTagValues()) > 0 {
		assetInventoryAnnos["tagValues"] = resource.GetTagValues()
	}
	if len(iamPolicies) > 0 {
		assetInventoryAnnos["iamPolicies"] = iamPolicies
	}

	assetInventorySpb, err := toProtoStruct(assetInventoryAnnos)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Asset Inventory annotations to structpb.Struct: %w", err)
	}

	return assetInventorySpb, nil
}

// getIAMPolicies get all IAM policies.
//
//nolint:staticcheck // see import.
func (p *AssetInventoryProcessor) getIAMPolicies(ctx context.Context, req *assetpb.SearchAllIamPoliciesRequest) ([]*v1.Policy, error) {
	iamPolicySearchResultIt := p.client.SearchAllIamPolicies(ctx, req)
	//nolint:staticcheck // see import.
	var iamPolicies []*v1.Policy
	for {
		iamPolicySearchResult, err := iamPolicySearchResultIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to search IAM policies: %w", err)
		}

		iamPolicies = append(iamPolicies, iamPolicySearchResult.GetPolicy())
	}
	return iamPolicies, nil
}

// getSingleResource get the single matched resource in Cloud Asset Inventory,
// returns error if 0 matched resource or multiple matched resources are found.
func (p *AssetInventoryProcessor) getSingleResource(ctx context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.ResourceSearchResult, error) {
	resourceSearchResultIt := p.client.SearchAllResources(ctx, req)
	var resources []*assetpb.ResourceSearchResult
	for {
		result, err := resourceSearchResultIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to search resources: %w", err)
		}
		resources = append(resources, result)
	}
	if got, want := len(resources), 1; got != want {
		return nil, fmt.Errorf("%d matched resources found, expected %d matched resource", got, want)
	}
	return resources[0], nil
}

// Stop stops the processor.
func (p *AssetInventoryProcessor) Stop() error {
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("failed to stop Asset Inventory Processor: %w", err)
	}
	return nil
}

// toProtoStruct converts v, which must marshal into a JSON object,
// into a proto struct.
func toProtoStruct(v any) (*structpb.Struct, error) {
	jb, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}
	x := &structpb.Struct{}
	if err := x.UnmarshalJSON(jb); err != nil {
		return nil, fmt.Errorf("structpb.Struct.UnmarshalJSON: %w", err)
	}
	return x, nil
}

// mergeAnnotations merges two annotations represented by structpb.Struct,
// if there is any field conflict, the field value in annos2 will override the field value in annos1.
func mergeAnnotations(annos1, annos2 *structpb.Struct) (*structpb.Struct, error) {
	fields1 := annos1.AsMap()
	fields2 := annos2.AsMap()
	mergedFields := map[string]any{}
	for k1, v1 := range fields1 {
		mergedFields[k1] = v1
	}
	for k2, v2 := range fields2 {
		mergedFields[k2] = v2
	}

	s, err := structpb.NewStruct(mergedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to construct a Struct from a merged Go map: %w", err)
	}
	return s, nil
}

// parseProject gets "Project" from "ResourceName" follows format here:https://cloud.google.com/asset-inventory/docs/resource-name-format.
// Return empty string for resources such as GCS bucket won't include "Project" info in its "ResourceName".
func parseProject(resourceName string) (string, error) {
	s := strings.Split(resourceName, "/")
	project := ""
	for i, e := range s {
		if e == "projects" {
			if i+1 >= len(s) || s[i+1] == "" {
				// This is obviously an invalid input.
				return "", fmt.Errorf("invalid resource name: %s", resourceName)
			}
			project = s[i+1]
		}
	}
	if project == "" {
		return "", nil
	}
	return fmt.Sprintf("projects/%s", project), nil
}
