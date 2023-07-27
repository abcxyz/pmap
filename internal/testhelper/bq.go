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
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/abcxyz/pkg/logging"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
)

// BQEntry defines the fields we need from bigquery entry.
type BQEntry struct {
	Data       string
	Attributes string
}

// GetFirstMatchedBQEntryWithRetries queries the DB and get the matched BQEntry.
// If no matched BQEntry was found, the query will be retried with the specified retry inputs.
func GetFirstMatchedBQEntryWithRetries(ctx context.Context, bqQuery *bigquery.Query, retryDuration time.Duration, retryLimit uint64) (*BQEntry, error) {
	logger := logging.FromContext(ctx)

	b := retry.NewConstant(retryDuration)
	var entry *BQEntry
	if err := retry.Do(ctx, retry.WithMaxRetries(retryLimit, b), func(ctx context.Context) error {
		results, err := queryBQEntries(ctx, bqQuery)
		if err != nil {
			logger.Infof("failed query BQEntries: %v", err)
			return err
		}

		// Early exit retry if queried pmap event already found.
		if len(results) > 0 {
			entry = results[0]
			return nil
		}
		logger.Info("Matching retry not found, retrying...")
		return retry.RetryableError(fmt.Errorf("no matching pmap event found in bigquery after timeout"))
	}); err != nil {
		return nil, fmt.Errorf("retry failed: %w", err)
	}
	return entry, nil
}

// queryBQEntries queries the BQ and checks if a bigqury entry matched the query exists or not and return the results.
func queryBQEntries(ctx context.Context, query *bigquery.Query) ([]*BQEntry, error) {
	job, err := query.Run(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to run query: %w", err))
	}

	if status, err := job.Wait(ctx); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to wait for query: %w", err))
	} else if err = status.Err(); err != nil {
		return nil, retry.RetryableError(fmt.Errorf("query failed: %w", err))
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, retry.RetryableError(fmt.Errorf("failed to read job: %w", err))
	}

	var entries []*BQEntry
	for {
		var entry BQEntry
		err := it.Next(&entry)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, retry.RetryableError(fmt.Errorf("failed to get next entry: %w", err))
		}

		entries = append(entries, &entry)
	}
	return entries, nil
}
