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

output "oidc_service_account" {
  description = "Service Account used for generating the OIDC tokens."
  value       = google_service_account.oidc_service_account.email
}

output "run_service_account" {
  description = "Service Account Cloud Run services to run as."
  value       = google_service_account.ci_run_service_account.email
}

output "gcs_pubsub_topic" {
  description = "PubSub topic for writing to BigQuery."
  value       = google_pubsub_topic.pmap_gcs_notification
}

output "bigquery_dataset" {
  description = "BigQuery dataset ID."
  value       = google_bigquery_dataset.pmap.dataset_id
}

output "mapping_bigquery_tables" {
  description = "BigQuery tables that store the mapping events."
  value       = module.mapping_bigquery.bigquery_tables
}

output "policy_bigquery_tables" {
  description = "BigQuery tables that store the policy events."
  value       = module.policy_bigquery.bigquery_tables
}

output "mapping_downstream_pubsub_topics" {
  description = "Downstream PubSub topics for mapping events."
  value       = module.mapping_bigquery.pubsub_topics
}

output "policy_downstream_pubsub_topics" {
  description = "Downstream PubSub topics for policy events."
  value       = module.policy_bigquery.pubsub_topics
}
