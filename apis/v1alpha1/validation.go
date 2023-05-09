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
)

// ValidateResourceMapping checks if the ResourceMapping is valid.
func ValidateResourceMapping(m *ResourceMapping) (vErr error) {
	for _, e := range m.Contacts.Email {
		if _, err := mail.ParseAddress(e); err != nil {
			vErr = errors.Join(vErr, fmt.Errorf("invalid owner: %w", err))
		}
	}
	if m.Resource.Provider == "" {
		vErr = errors.Join(vErr, fmt.Errorf("empty resource provider"))
	}
	if _, ok := m.Annotations.AsMap()["assetInfo"]; ok {
		vErr = errors.Join(vErr, fmt.Errorf("reserved key is included: assertInfo"))
	}
	return
}
