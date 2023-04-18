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
  static_bucket_name = "pmap-static-ci-bucket-${random_id.default.hex}"
}

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
