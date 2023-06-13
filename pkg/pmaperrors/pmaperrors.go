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

// Error is a concrete error implementation.
type Error string

// Error satisfies the error interface.
func (e Error) Error() string {
	return string(e)
}

// These errors are used in EventHandler's Handle() function.
// The retryable errors will be returned later by HTTPHandler() to pubsub
// so pubsub will try send messages to handler again.
const (
	// ErrNonRetryable is the (base) error to return when a pmap
	// processor considers an error is retryable.
	ErrPubsubRetryable = Error("pubsub retryable error")

	// ErrNonRetryable is the (base) error to return when a pmap
	// processor considers an error is none-retryable.
	ErrPubsubNonRetryable = Error("pubsub none-retryable error")
)
