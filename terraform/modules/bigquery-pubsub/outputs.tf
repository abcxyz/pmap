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

output "pubsub_topic_ids" {
  value       = [for topic in google_pubsub_topic.topics : topic.id]
  description = "The IDs of the Pub/Sub topics."
}

output "dead_letter_topic_ids" {
  value       = [for topic in google_pubsub_topic.dead_letter_topics : topic.id]
  description = "The IDs of the Pub/Sub dead letter topics."
}

output "bigquery_subscriptions_ids" {
  value       = [for subscription in google_pubsub_subscription.bigquery_subscriptions : subscription.id]
  description = "The Pub/Sub subscription names."
}

output "bigquery_tables" {
  value       = google_bigquery_table.tables
  description = "Map of bigquery table resources being provisioned."
}

output "bigquery_dataset" {
  value       = google_bigquery_dataset.dataset
  description = "Bigquery dataset resource."
}

output "bigquery_table_ids" {
  value = [
    for table in google_bigquery_table.tables :
    table.id
  ]
  description = "Unique id for the table being provisioned."
}
