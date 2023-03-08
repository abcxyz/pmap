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

output "bigquery_topic" {
  description = <<EOF
      BigQuery tables and corresponding Pub/Sub topics for publishing events
      to the tables. One pair for successful events and the other for failures.
    EOF
  value = {
    event_topic   = google_pubsub_topic.bigquery[local.success_table].name
    event_table   = google_bigquery_table.pmap[local.success_table].id
    failure_topic = google_pubsub_topic.bigquery[local.failure_table].name
    failure_table = google_bigquery_table.pmap[local.failure_table].id
  }
}
