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
  value       = google_service_account.oidc_service_account.email
}

output "run_service_account" {
  description = "Service Account Cloud Run services to run as."
  value       = google_service_account.run_service_account.email
}

output "gcs_pubsub_topic" {
  description = "A map of event to PubSub topics."
  value       = google_pubsub_topic.pmap_gcs_notification
}

output "bigquery_dataset" {
  description = "BigQuery dataset ID."
  value       = google_bigquery_dataset.pmap.dataset_id
}

output "downstream_resouces" {
  description = "A map of event to downstream PubSub topics and BigQuery tables."
  value       = { for event in var.event_types : event => module.pubsub_bigquery[event].topics_and_tables }
}
