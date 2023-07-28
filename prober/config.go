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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type config struct {
	ProjectID                string        `env:"PROBER_PROJECT_ID,required"`
	GCSBucketID              string        `env:"PROBER_BUCKET_ID,required"`
	BigQueryDataSetID        string        `env:"PROBER_BIGQUERY_DATASET_ID,required"`
	MappingTableID           string        `env:"PROBER_MAPPING_TABLE_ID,required"`
	QueryRetryWaitDuration   time.Duration `env:"PROBER_QUERY_RETRY_WAIT_DURATION,default=5s"`
	QueryRetryLimit          uint64        `env:"PROBER_QUERY_RETRY_COUNT,default=10"`
	ProberMappingServiceName string        `env:"PROBER_MAPPING_SERVICE_NAME,required"`
	ProberMappingFilePrefix  string        `env:"PROBER_MAPPING_FILE_PREFIX,required"`
}

func newTestConfig(ctx context.Context) (*config, error) {
	var c config
	if err := envconfig.ProcessWith(ctx, &c, envconfig.OsLookuper()); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}
	return &c, nil
}
