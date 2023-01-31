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

module "mapping_service" {
  source                  = "../modules/pmap-service"
  service_name            = "mapping"
  project_id              = var.project_id
  image                   = var.mapping_service_image
  downstream_pubsub_topic = "projects/${var.project_id}/topics/mapping-bigquery"
  upstream_pubsub_topic   = "projects/${var.project_id}/topics/mapping-gcs"
  gcs_bucket_name         = "pmap"
  pmap_service_account    = "run-pmap-sa@${var.project_id}.iam.gserviceaccount.com"
  ci_service_account      = var.ci_service_account
}

module "policy_service" {
  source                  = "../modules/pmap-service"
  service_name            = "policy"
  project_id              = var.project_id
  image                   = var.policy_service_image
  downstream_pubsub_topic = "projects/${var.project_id}/topics/policy-bigquery"
  upstream_pubsub_topic   = "projects/${var.project_id}/topics/policy-gcs"
  gcs_bucket_name         = "pmap"
  pmap_service_account    = "run-pmap-sa@${var.project_id}.iam.gserviceaccount.com"
  ci_service_account      = var.ci_service_account
}
