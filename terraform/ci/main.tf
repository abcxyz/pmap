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

module "common_infra" {
  source = "../modules/common"

  project_id = var.project_id

  gcs_bucket_name = var.gcs_bucket_name
  event_types     = var.event_types

  // Terraform destroy or terraform apply that would delete the table instance will fail.
  bigquery_table_delete_protection = true
}

resource "random_id" "default" {
  byte_length = 2
}


locals {
  static_bucket_name        = "pmap-static-ci-bucket-${random_id.default.hex}"
  static_artifact_repo_name = "pmap-static-ci-artifact-registry-repo-${random_id.default.hex}"
}

# Add a static GCS bucket for integration test, as GCS bucket is
# unscoped resource
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

# Add a static artifact registry repo for integration test, as artifact repo is
# scoped resource.
resource "google_artifact_registry_repository" "integ_test_deicated_artifact_repo" {
  project = var.project_id

  location      = "us-central1"
  repository_id = local.static_artifact_repo_name
  description   = "static artifact registry repo for pmap ci tests"
  format        = "DOCKER"

  labels = {
    env = "pmap-ci-test"
  }
}

# Add service account to static artifact registry repo
# IAM for testing purpose, therefore the IAM policy
# list won't be empty
resource "google_artifact_registry_repository_iam_member" "static_artifact_repo_object_viewer" {
  project = var.project_id

  location   = google_artifact_registry_repository.integ_test_deicated_artifact_repo.location
  repository = google_artifact_registry_repository.integ_test_deicated_artifact_repo.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${module.common_infra.run_service_account}"
}
