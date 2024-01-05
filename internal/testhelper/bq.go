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

// Package testhelper provides utilities that are intended to enable easier and more concise writing of test and probder code.
package testhelper

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/sethvargo/go-retry"
)

// BQEntry defines the fields we need from bigquery entry.
type BQEntry struct {
	Data       string
	Attributes string
}

// SingleBQEntry returns a single [BQEntry] from the given query.
func SingleBQEntry(ctx context.Context, bqQuery *bigquery.Query, backoff retry.Backoff) (*BQEntry, error) {
	var entry *BQEntry
	if err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		result, err := func() (*BQEntry, error) {
			job, err := bqQuery.Run(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to run query: %w", err)
			}

			if status, err := job.Wait(ctx); err != nil {
				return nil, fmt.Errorf("failed to wait for query: %w", err)
			} else if status.Err() != nil {
				return nil, fmt.Errorf("query failed: %w", status.Err())
			}

			it, err := job.Read(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to read query result: %w", err)
			}

			var r BQEntry
			if err := it.Next(&r); err != nil {
				return nil, fmt.Errorf("failed to read first entry: %w", err)
			}
			return &r, nil
		}()
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to get entry: %w", err))
		}

		entry = result
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get matched bq entry: %w", err)
	}
	return entry, nil
}
