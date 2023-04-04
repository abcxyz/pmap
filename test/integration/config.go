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

package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type config struct {
	ProjectID              string        `env:"INTEG_TEST_PROJECT_ID,required"`
	GCSBucketID            string        `env:"INTEG_TEST_BUCKET_ID,required"`
	BigQueryDataSetID      string        `env:"INTEG_TEST_BIGQUERY_DATASET_ID,required"`
	MappingTableID         string        `env:"INTEG_TEST_MAPPING_TABLE_ID,required"`
	MappingFailureTableID  string        `env:"INTEG_TEST_MAPPING_FAILURE_TABLE_ID,required"`
	PolicyTableID          string        `env:"INTEG_TEST_POLICY_TABLE_ID,required"`
	QueryRetryWaitDuration time.Duration `env:"INTEG_TEST_QUERY_RETRY_WAIT_DURATION,default=5s"`
	QueryRetryLimit        uint64        `env:"INTEG_TEST_QUERY_RETRY_COUNT,default=5"`
	MappingDownstreamTopic string        `env:"INTEG_TEST_MAPPING_DOWNSTREAM_TOPIC,required"`

	// ObjectPrefix is a unique object prefix for each integration test run.
	// It's required to trigger the correct GCS notification for each CI/CD run
	// which aims to avoid of side effects when multiple CI/CD runs in parallel.
	ObjectPrefix string `env:"INTEG_TEST_OBJECT_PREFIX,required"`
}

func newTestConfig(ctx context.Context) (*config, error) {
	var c config
	if err := envconfig.ProcessWith(ctx, &c, envconfig.OsLookuper()); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}
	return &c, nil
}
