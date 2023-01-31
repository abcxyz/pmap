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

output "policy_service_url" {
  description = "Policy cloud run service URL."
  value       = module.policy_service.pmap_service_url
}

output "mapping_service_url" {
  description = "Mapping cloud run service URL."
  value       = module.mapping_service.pmap_service_url
}

output "policy_gcs_subscription_id" {
  description = "The Pub/Sub subscription ID for policy."
  value       = module.policy_service.gcs_pubsub_subscription_id
}

output "mapping_gcs_subscription_id" {
  description = "The Pub/Sub subscription ID for mapping."
  value       = module.mapping_service.gcs_pubsub_subscription_id
}