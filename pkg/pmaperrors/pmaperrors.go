// Copyright 2023 PAMP authors (see AUTHORS file)
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

// Package pmaperrors defines the sentinel errors for the project.
package pmaperrors

import (
	"errors"
	"fmt"
)

type processError struct {
	err error
}

func New(format string, args ...any) error {
	return Wrap(fmt.Errorf(format, args...))
}

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &processError{err}
}

func Is(err error) bool {
	var rerr *processError
	return errors.As(err, &rerr)
}

func (e *processError) Unwrap() error {
	return e.err
}

func (e *processError) Error() string {
	if e.err == nil {
		return "pmap process err: <nil>"
	}
	return "pmap process err: " + e.err.Error()
}
