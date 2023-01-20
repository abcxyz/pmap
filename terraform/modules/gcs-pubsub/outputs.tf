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

output "gcs_bucket" {
  value       = google_storage_bucket.bucket
  description = "The created GCS bucket."
}

output "gcs_bucket_name" {
  value       = google_storage_bucket.bucket.name
  description = "The name of the GCS bucket."
}

output "gcs_bucket_url" {
  value       = google_storage_bucket.bucket.url
  description = "The URL of the GCS bucket."
}

output "gcs_notification_id" {
  value       = [for notification in google_storage_notification.notification : notification.notification_id]
  description = "The ID of the created GCS notification."
}

output "gcs_notification_self_link" {
  value       = [for notification in google_storage_notification.notification : notification.self_link]
  description = "The URI of the created GCS notification."
}

output "pubsub_topic_ids" {
  value       = [for topic in google_pubsub_topic.topics : topic.id]
  description = "The IDs of the Pub/Sub topic."
}

output "wif_service_account" {
  value       = google_service_account.gh-access-acc.email
  description = "Workload Idendity Federation service account for GitHub action to access cloud resources."
}
