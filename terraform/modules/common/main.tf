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
  github_slug              = "abcxyz/pmap"
  pubsub_svc_account_email = "service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

data "google_project" "project" {
  project_id = var.project_id
}

resource "google_project_service" "serviceusage" {
  project            = var.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "pubsub.googleapis.com",
    "iam.googleapis.com",
    "bigquery.googleapis.com",
    "storage.googleapis.com"
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

// Create BigQuery dataset and tables.
resource "google_bigquery_dataset" "pmap" {
  project                         = var.project_id
  dataset_id                      = "pmap"
  friendly_name                   = "Privacy data annotations and mappings."
  description                     = "Dataset for data annotations and their mappings to data resources."
  location                        = "US"
  delete_contents_on_destroy      = false
  default_partition_expiration_ms = 172800000 // 2 days.
}

// Create PubSub topics, BigQuery subcriptions, and BigQuery tables for successfully and unsuccessfully processed events.
module "pubsub_bigquery" {
  for_each = toset(var.event_types)

  source                           = "../pubsub-bigquery"
  project_id                       = var.project_id
  dataset_id                       = google_bigquery_dataset.pmap.dataset_id
  event                            = each.key
  run_service_account              = google_service_account.run_service_account.email
  bigquery_table_delete_protection = var.bigquery_table_delete_protection

  depends_on = [
    google_bigquery_dataset_iam_member.viewer,
    google_bigquery_dataset_iam_member.editors
  ]
}

// Add Pub/Sub service account to metadataViewer role required for writting to BigQuery.
// See link: https://cloud.google.com/pubsub/docs/create-subscription#assign_bigquery_service_account.
resource "google_bigquery_dataset_iam_member" "viewer" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.pmap.dataset_id
  role       = "roles/bigquery.metadataViewer"
  member     = "serviceAccount:${local.pubsub_svc_account_email}"
}

// Grant roles to Pub/Sub service account required for writting to BigQuery.
// See link: https://cloud.google.com/pubsub/docs/create-subscription#assign_bigquery_service_account.
resource "google_bigquery_dataset_iam_member" "editors" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.pmap.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_storage_bucket" "pmap" {
  name                        = var.gcs_bucket_name
  project                     = var.project_id
  location                    = "US"
  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 2 // Delete in 2 days since we are in CI.
    }
  }
}

// Create two notifications, one for mapping and one for policy.
resource "google_storage_notification" "pmap" {
  for_each = toset(var.event_types)

  bucket         = google_storage_bucket.pmap.name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.pmap_gcs_notification[each.key].id
  event_types    = ["OBJECT_FINALIZE"]
  // Separate mapping and policy notifications by object name prefix.
  // Mapping objects start with "mapping", whereas policy start with "policy".
  object_name_prefix = "${each.key}/"
  depends_on         = [google_pubsub_topic_iam_member.publishers]
}

// Enable notifications by giving the correct IAM permission to the unique service account.
data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

resource "google_pubsub_topic_iam_member" "publishers" {
  for_each = toset(var.event_types)

  topic  = google_pubsub_topic.pmap_gcs_notification[each.key].id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"
}

// Create two Pub/Sub topics for gcs notification, one for mapping and one for policy.
resource "google_pubsub_topic" "pmap_gcs_notification" {
  for_each = toset(var.event_types)

  project = var.project_id
  name    = "${each.value}-gcs"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Create a dedicated service account for pmap services to run as.
resource "google_service_account" "run_service_account" {
  project      = var.project_id
  account_id   = "run-pmap-sa"
  display_name = "Cloud Run Service Account for pmap"
}

// Grant GCS object viewer permission to the pmap service account.
resource "google_storage_bucket_iam_member" "object_viewer" {
  bucket = google_storage_bucket.pmap.name
  role   = "roles/storage.objectViewer"
  member = google_service_account.run_service_account.member
}

// Create a dedicated service account for generating the OIDC tokens, required to enable request
// authentication when messages from Pub/Sub are delivered to push endpoints. If the endpoint is
// a Cloud Run service, this service account needs to be the run invoker.
resource "google_service_account" "oidc_service_account" {
  project      = var.project_id
  account_id   = "pmap-oidc"
  display_name = "Service Account used for generating the OIDC tokens"
}
