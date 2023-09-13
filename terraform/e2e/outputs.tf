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

output "bigquery_dataset" {
  description = "BigQuery dataset ID."
  value       = module.common_infra.bigquery_dataset
}

output "bigquery_topics" {
  description = "A map of event to Pub/Sub topics and BigQuery tables."
  value       = module.common_infra.bigquery_topics
}

output "bigquery_subscriptions" {
  description = "A map of event to Pub/Sub topics and BigQuery tables."
  value       = module.common_infra.bigquery_subscriptions
}

output "run_service_account_member" {
  description = "Service Account name Cloud Run services to run as in the form serviceAccount:{email}."
  value       = module.common_infra.run_service_account_member
}

output "run_service_account" {
  description = "Service Account Cloud Run services to run as."
  value       = module.common_infra.run_service_account
}

output "run_service_account_name" {
  description = "Service Account name Cloud Run services to run as."
  value       = module.common_infra.run_service_account_name
}

output "prober_service_account" {
  description = "Service Account Prober job to run as."
  value       = module.monitoring.prober_service_account
}

output "prober_service_account_name" {
  description = "Service Account name Prober job to run as."
  value       = module.monitoring.prober_service_account_name
}

output "gcs_bucket_name" {
  description = "GCS bucket name"
  value       = module.common_infra.gcs_bucket_name
}

