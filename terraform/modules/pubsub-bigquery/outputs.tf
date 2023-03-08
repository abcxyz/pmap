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

output "topics_and_tables" {
  description = "Pub/Sub topic and BigQuery table for success events."
  value = {
    downstream_pubsub_topic         = google_pubsub_topic.bigquery[local.success_table].name
    bigquery_table                  = google_bigquery_table.pmap[local.success_table].id
    downstream_failure_pubsub_topic = google_pubsub_topic.bigquery[local.failure_table].name
    failure_bigquery_table          = google_bigquery_table.pmap[local.failure_table].id
  }
}
