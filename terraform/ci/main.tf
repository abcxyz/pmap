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
  short_sha = substr(var.commit_sha, 0, 7)
}

module "mapping_service" {
  source                = "../modules/pmap-service"
  service_name          = "mapping-${local.short_sha}"
  project_id            = var.project_id
  image                 = var.mapping_service_image
  publish_to_topic_id   = "projects/${var.infra_project_id}/topics/mapping"
  subscribe_to_topic_id = "projects/${var.infra_project_id}/topics/pmap_gcs_notification"
  gcs_bucket_name       = "pmap-ci"
  // TODO: add mapping subscription filter.
}

module "retention_service" {
  source                = "../modules/pmap-service"
  service_name          = "retention-${local.short_sha}"
  project_id            = var.project_id
  image                 = var.retention_service_image
  publish_to_topic_id   = "projects/${var.infra_project_id}/topics/retention"
  subscribe_to_topic_id = "projects/${var.infra_project_id}/topics/pmap_gcs_notification"
  gcs_bucket_name       = "pmap-ci"
  // TODO: add retention subscription filter.
}
