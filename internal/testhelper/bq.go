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

	"github.com/abcxyz/pkg/bqutil"
)

// BQEntry defines the fields we need from bigquery entry.
type BQEntry struct {
	Data       string
	Attributes string
}

// SingleBQEntry returns a single [BQEntry] from the given query.
func SingleBQEntry(ctx context.Context, bqQuery *bigquery.Query, backoff retry.Backoff) (*BQEntry, error) {
	q := bqutil.NewQuery[BQEntry](bqQuery)
	results, err := bqutil.RetryQueryEntries(ctx, q, 1, backoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query BigQuery: %w", err)
	}
	return &results[0], nil
}
