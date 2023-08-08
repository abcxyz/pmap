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

// Package v1alpha1 contains versioned pmap contracts, e.g. resource mapping definition.
package v1alpha1

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/google/go-cmp/cmp"
)

const (
	// Reserved key where annotation from CAIS will be stored.
	AnnotationKeyAssetInfo = "assetInfo"
)

// ValidateResourceMapping checks if the ResourceMapping is valid.
func ValidateResourceMapping(m *ResourceMapping) (vErr error) {
	for _, e := range m.Contacts.Email {
		if _, err := mail.ParseAddress(e); err != nil {
			vErr = errors.Join(vErr, fmt.Errorf("invalid owner: %w", err))
		}
	}

	if _, ok := m.Annotations.AsMap()[AnnotationKeyAssetInfo]; ok {
		vErr = errors.Join(vErr, fmt.Errorf("reserved key is included: %s", AnnotationKeyAssetInfo))
	}

	if err := validateResource(m.Resource); err != nil {
		vErr = errors.Join(vErr, fmt.Errorf("invalid Resourc: %w", err))
	}

	return
}

func validateResource(r *Resource) (vErr error) {
	if r.Name == "" {
		vErr = errors.Join(vErr, fmt.Errorf("empty resource name"))
	}

	if r.Provider == "" {
		vErr = errors.Join(vErr, fmt.Errorf("empty resource provider"))
	}

	if err := validateAndNormalizeSubscope(r); err != nil {
		vErr = errors.Join(vErr, fmt.Errorf("invalid subscope: %w", err))
	}

	return
}

func validateAndNormalizeSubscope(r *Resource) error {
	if r.Subscope == "" {
		return nil
	}

	// normalize subscope string to only have lower cases
	if diff := cmp.Diff(strings.ToLower(r.Subscope), r.Subscope); diff != "" {
		return fmt.Errorf("subscope should only contain lower case char")
	}

	u, err := url.Parse(r.Subscope)
	if err != nil {
		return fmt.Errorf("failed to parse subscope string: %w", err)
	}

	_, err = url.ParseQuery(u.RawQuery)
	if err != nil {
		return fmt.Errorf("failed to parse qualifier string: %w", err)
	}

	// check if key value pairs are in alphabetical order
	s := strings.Split(u.RawQuery, "&")
	if len(s) <= 1 {
		return nil
	}
	for i := 1; i < len(s); i++ {
		if strings.Split(s[i], "=")[0] < strings.Split(s[i-1], "=")[0] {
			return fmt.Errorf("keys should be in alphabetical order, got %s&%s, want %s&%s", s[i-1], s[i], s[i], s[i-1])
		}
	}

	return nil
}
