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

output "cloudrun_service_url" {
  value       = { for service in var.services : service.name => module.cloud-run[service.name].service_url }
  description = "Map of cloud run services urls."
}

output "pubsub_subscription_ids" {
  value       = [for subscription in google_pubsub_subscription.push_subscriptions : subscription.id]
  description = "The Pub/Sub subscription ids."
}

output "dead_letter_topic_id" {
  value       = google_pubsub_topic.dead_letter_topic.id
  description = "The ID of the Pub/Sub dead letter topic."
}

output "cloudrun_service_identitys" {
  value       = { for service in var.services : service.name => module.cloud-run[service.name].cloud_run_service_account }
  description = "Map of dedicated service accounts for cloud run services."
}
