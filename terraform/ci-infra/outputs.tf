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

output "artifact_repository_name" {
  description = "The Artifact Registry name."
  value       = module.github_ci_infra.artifact_repository_name
}

output "wif_pool_name" {
  description = "The Workload Identity Federation pool name."
  value       = module.github_ci_infra.wif_pool_name
}

output "wif_provider_name" {
  description = "The Workload Identity Federation provider name."
  value       = module.github_ci_infra.wif_provider_name
}

output "ci_service_account" {
  description = "CI service account, also the Workload Identify Federation service account for GitHub access."
  value       = module.github_ci_infra.service_account_email
}

output "pmap_dataset_id" {
  description = "Pmap BigQuery dataset ID."
  value       = google_bigquery_dataset.pmap.id
}

output "mapping_table_id" {
  description = "Mapping BigQuery table ID."
  value       = google_bigquery_table.mapping.id
}

output "retention_table_id" {
  description = "Retention BigQuery table ID."
  value       = google_bigquery_table.retention.id
}

output "mapping_pubsub_topic" {
  description = "Topic for writing mapping message to BigQuery."
  value       = google_pubsub_topic.mapping.id
}

output "retention_pubsub_topic" {
  description = "Topic for writing retention message to BigQuery."
  value       = google_pubsub_topic.retention.id
}

output "mapping_bigquery_subscription" {
  description = "BigQuery subscription for writing mapping message to BigQuery."
  value       = google_pubsub_subscription.mapping.id
}

output "retention_bigquery_subscription" {
  description = "BigQuery subscription for writing retention message to BigQuery."
  value       = google_pubsub_subscription.retention.id
}

output "gcs_bucket_url" {
  description = "GCS bucket URL for pmap."
  value       = google_storage_bucket.pmap.url
}

output "gcs_notification_id" {
  description = "Pmap GCS bucket notification ID."
  value       = google_storage_notification.pmap.notification_id
}

output "gcs_notification_topic" {
  description = "Workload Identify Federation service account for GitHub access."
  value       = google_pubsub_topic.pmap_gcs_notification.id
}
