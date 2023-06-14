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

// These errors are used in EventHandler's Handle() function.
// The RetryableError will be returned later by HTTPHandler() to pubsub
// so pubsub will try send messages to handler again.
type RetryableError struct {
	err error
}

// Unwrap implements error wrapping.
func (e *RetryableError) Unwrap() error {
	return e.err
}

// Error returns the error string.
func (e *RetryableError) Error() string {
	if e.err == nil {
		return "retryable: <nil>"
	}
	return "retryable: " + e.err.Error()
}
