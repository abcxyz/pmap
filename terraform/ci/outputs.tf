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
  description = <<EOT
        Service Account used for generating the OIDC tokens. Required to enable request
        authentication when messages from Pub/Sub are delivered to push endpoints. If the
        endpoint is a Cloud Run service, this service account needs to be the run invoker.
    EOT
  value       = module.common_infra.oidc_service_account
}

output "oidc_service_account_name" {
  description = <<EOT
        Service Account name used for generating the OIDC tokens. Required to enable request
        authentication when messages from Pub/Sub are delivered to push endpoints. If the
        endpoint is a Cloud Run service, this service account needs to be the run invoker.
    EOT
  value       = module.common_infra.oidc_service_account_name
}

output "run_service_account" {
  description = "Service Account Cloud Run services to run as."
  value       = module.common_infra.run_service_account
}

output "run_service_account_name" {
  description = "Service Account name Cloud Run services to run as."
  value       = module.common_infra.run_service_account_name
}

output "gcs_notification_topics" {
  description = "A map of event to GCS notification Pub/Sub topics."
  value       = module.common_infra.gcs_notification_topics
}

output "bigquery_dataset" {
  description = "BigQuery dataset ID."
  value       = module.common_infra.bigquery_dataset
}

output "bigquery_topics" {
  description = "A map of event to Pub/Sub topics and BigQuery tables."
  value       = module.common_infra.bigquery_topics
}

output "static_gcs_bucket_name" {
  description = "Name for static GCS bucket."
  value       = google_storage_bucket.integ_test_dedicated_bucket.name
}
