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
  prober_env_vars = {
    "PROBER_PROJECT_ID" : var.project_id,
    "PROBER_BUCKET_ID" : var.prober_bucket_id,
    "PROBER_BIGQUERY_DATASET_ID" : var.prober_bigquery_dataset_id,
    "PROBER_MAPPING_TABLE_ID" : var.prober_mapping_table_id,
    "PROBER_POLICY_TABLE_ID" : var.prober_policy_table_id,
    "PROBER_MAPPING_GCS_BUCKET_PREFIX" : var.prober_mapping_gcs_bucket_prefix,
    "PROBER_POLICY_GCS_BUCKET_PREFIX" : var.prober_policy_gcs_bucket_prefix,
    "PROBER_QUERY_RETRY_WAIT_DURATION" : var.prober_query_retry_wait_duartion,
    "PROBER_QUERY_RETRY_COUNT" : var.prober_query_retry_count,
    "LOG_LEVEL" : var.log_level
  }
}

resource "google_project_service" "serviceusage" {
  project = var.project_id

  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudscheduler.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

resource "google_cloud_run_v2_job" "pmap_prober" {

  project = var.project_id

  name = "pmap-prober"

  location = "us-central1"

  depends_on = [
    google_project_service.services["cloudscheduler.googleapis.com"],
  ]

  template {

    template {

      max_retries = 3

      containers {
        image = var.pmap_prober_image

        dynamic "env" {
          for_each = local.prober_env_vars
          content {
            name  = env.key
            value = env.value
          }
        }
      }
      service_account = google_service_account.prober_service_account.email
    }
  }

  lifecycle {
    ignore_changes = [
      launch_stage,
      template[0].template[0].containers[0].image,
    ]
  }
}

# This is prober dedicated service accout.
# This service account will be used to trigger cloud run job by cloud scheduler
# and also used by prober for upload GCS object and query bigquery table.
resource "google_service_account" "prober_service_account" {
  project = var.project_id

  account_id   = "pmap-prober"
  display_name = "Prober Service Account"
}

# Grant pmap-prober cloud run invoker and bigquery job user role.
resource "google_project_iam_member" "prober_sa_role" {
  for_each = toset([
    "roles/bigquery.jobUser",
    "roles/run.invoker",
  ])

  project = var.project_id

  role   = each.value
  member = google_service_account.prober_service_account.member
}

# Grant pmap-prober GCS objectCreator role
resource "google_storage_bucket_iam_member" "member" {
  bucket = var.prober_bucket_id
  role   = "roles/storage.objectCreator"
  member = google_service_account.prober_service_account.member
}

# Grant pmap-prober bigqeury dataviewer role
resource "google_bigquery_dataset_iam_member" "viewer" {
  for_each = toset([
    "roles/bigquery.dataViewer",
  ])

  project = var.project_id

  dataset_id = var.prober_bigquery_dataset_id
  role       = each.value
  member     = google_service_account.prober_service_account.member
}

# This is the scheduler for triggering pmap-prober cloud run job
# in a user defined frequency.
resource "google_cloud_scheduler_job" "job" {

  project = var.project_id

  schedule    = var.prober_scheduler
  name        = "pmap-prober-scheduler"
  description = "prober cloud run job scheduler"
  region      = "us-central1"

  retry_config {
    retry_count = 0
  }

  http_target {
    http_method = "POST"
    uri         = "https://${resource.google_cloud_run_v2_job.pmap_prober.location}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${resource.google_cloud_run_v2_job.pmap_prober.name}:run"

    oauth_token {
      service_account_email = google_service_account.prober_service_account.email
    }
  }

  depends_on = [google_cloud_run_v2_job.pmap_prober]
}
