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

locals {
  mapping_service_name                = "mapping"
  policy_service_name                 = "policy"
  pmap_mapping_default_resource_scope = var.pmap_mapping_default_resource_scope == "" ? format("projects/%s", var.project_id) : var.pmap_mapping_default_resource_scope
}

module "common_infra" {
  source = "../modules/common"

  project_id = var.project_id

  gcs_bucket_name                  = var.gcs_bucket_name
  event_types                      = [local.mapping_service_name, local.policy_service_name]
  bigquery_table_delete_protection = var.bigquery_table_delete_protection
}

module "mapping_service" {
  source = "../modules/pmap-service"

  project_id = var.project_id

  service_name                        = local.mapping_service_name
  pmap_container_image                = var.pmap_container_image
  pmap_args                           = ["mapping", "server"]
  upstream_topic                      = module.common_infra.gcs_notification_topics[local.mapping_service_name].name
  downstream_topic                    = module.common_infra.bigquery_topics[local.mapping_service_name].event_topic
  downstream_failure_topic            = module.common_infra.bigquery_topics[local.mapping_service_name].failure_topic
  pmap_service_account                = module.common_infra.run_service_account
  oidc_service_account                = module.common_infra.oidc_service_account
  gcs_events_filter                   = var.mapping_gcs_events_filter
  pmap_mapping_default_resource_scope = local.pmap_mapping_default_resource_scope
}

module "policy_service" {
  source = "../modules/pmap-service"

  project_id = var.project_id

  service_name         = local.policy_service_name
  pmap_container_image = var.pmap_container_image
  pmap_args            = ["policy", "server"]
  upstream_topic       = module.common_infra.gcs_notification_topics[local.policy_service_name].name
  downstream_topic     = module.common_infra.bigquery_topics[local.policy_service_name].event_topic
  pmap_service_account = module.common_infra.run_service_account
  oidc_service_account = module.common_infra.oidc_service_account
  gcs_events_filter    = var.policy_gcs_events_filter
}

resource "random_id" "default" {
  byte_length = 2
}

locals {
  static_bucket_name = "pmap-static-ci-bucket-${random_id.default.hex}"
}

# Create a static GCS bucket in dev project
# so integ test can test the customized scope functionality
# from another project
resource "google_storage_bucket" "integ_test_dedicated_bucket" {
  project = var.project_id

  name                        = local.static_bucket_name
  location                    = "US"
  force_destroy               = false
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  labels = {
    env = "pmap-ci-test"
  }
}

# Add service account to static bucket IAM
# for testing purpose, therefore the IAM policy
# list won't be empty
resource "google_storage_bucket_iam_member" "static_bucket_object_viewer" {
  bucket = google_storage_bucket.integ_test_dedicated_bucket.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${module.common_infra.run_service_account}"
}
