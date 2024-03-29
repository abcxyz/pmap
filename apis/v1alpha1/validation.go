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

// Package v1alpha1 contains versioned pmap contracts, e.g. resource mapping
// definition.
package v1alpha1

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"sort"
	"strings"
)

const (
	// Reserved key where annotation from CAIS will be stored.
	AnnotationKeyAssetInfo = "assetInfo"
)

// ValidateResourceMapping checks if the ResourceMapping is valid.
func ValidateResourceMapping(m *ResourceMapping) (vErr error) {
	for _, e := range m.GetContacts().GetEmail() {
		if _, err := mail.ParseAddress(e); err != nil {
			vErr = errors.Join(vErr, fmt.Errorf("invalid owner: %w", err))
		}
	}

	if _, ok := m.GetAnnotations().AsMap()[AnnotationKeyAssetInfo]; ok {
		vErr = errors.Join(vErr, fmt.Errorf("reserved key is included: %s", AnnotationKeyAssetInfo))
	}

	if err := validateResource(m.GetResource()); err != nil {
		vErr = errors.Join(vErr, err)
	}

	return
}

func validateResource(r *Resource) (vErr error) {
	if r.GetName() == "" {
		vErr = errors.Join(vErr, fmt.Errorf("empty resource name"))
	}

	if r.GetProvider() == "" {
		vErr = errors.Join(vErr, fmt.Errorf("empty resource provider"))
	}

	if err := validateSubscope(r); err != nil {
		vErr = errors.Join(vErr, err)
	}

	return
}

func validateSubscope(r *Resource) error {
	if r.GetSubscope() == "" {
		return nil
	}

	// If r.Subscope = "parent/foo/child/bar?key1=value1&key2=value2" after
	// url.Parse(r.Subscope), we will have: u.RawQuery: key1=value1&key2=value2
	u, err := url.Parse(r.GetSubscope())
	if err != nil {
		return fmt.Errorf("subscope validation failed: failed to parse subscope string %s: %w", r.GetSubscope(), err)
	}

	// [url.Parse] silently discards malformed value pairs. So we need to use
	// [url.ParseQuery] to check if there are any errors.
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return fmt.Errorf("subscope validation failed: failed to parse qualifier string %s: %w", u.RawQuery, err)
	}

	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var kvPairs []string
	for _, k := range keys {
		sort.Strings(q[k])
		for _, v := range q[k] {
			kvPairs = append(kvPairs, fmt.Sprintf("%s=%s", k, v))
		}
	}

	wantQueryString := strings.Join(kvPairs, "&")
	if wantQueryString != u.RawQuery {
		return fmt.Errorf("subscope validation failed: qualifiers must be in alphabetical order, want: %s, got: %s", wantQueryString, u.RawQuery)
	}

	return nil
}
