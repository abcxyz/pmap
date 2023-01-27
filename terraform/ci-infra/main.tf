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
  event_type               = toset(["mapping", "retention"])
}

data "google_project" "project" {
  project_id = var.project_id
}

// Create CI infrastructure resources including artifact repository, workload identity pool and provider,
// and CI service account.
module "github_ci_infra" {
  source                 = "git@github.com:abcxyz/terraform-modules.git//modules/github_ci_infra?ref=main"
  project_id             = var.project_id
  name                   = "pmap"
  github_repository_name = "pmap"
}

resource "google_project_service" "serviceusage" {
  project            = var.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "pubsub.googleapis.com",
    "iam.googleapis.com",
    "bigquery.googleapis.com",
    "storage.googleapis.com"
  ])
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

resource "google_bigquery_table" "pmap" {
  for_each            = local.event_type
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.pmap.dataset_id
  table_id            = each.key
  deletion_protection = true

  time_partitioning {
    type  = "DAY"
    field = "publish_time"
  }

  schema = <<EOF
[
  {
    "name": "data",
    "type": "JSON",
    "mode": "REQUIRED"
  },
  {
    "name": "subscription_name",
    "type": "STRING",
    "mode": "REQUIRED"
  },
  {
    "name": "message_id",
    "type": "STRING",
    "mode": "REQUIRED"
  },
  {
    "name": "publish_time",
    "type": "TIMESTAMP",
    "mode": "REQUIRED"
  },
  {
    "name": "attributes",
    "type": "JSON",
    "mode": "REQUIRED"
  }
]
EOF
}

resource "google_pubsub_topic" "bigquery" {
  for_each = local.event_type
  project  = var.project_id
  name     = each.key

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_subscription" "bigquery" {
  for_each = local.event_type
  project  = var.project_id
  name     = "${each.key}-bigquery"
  topic    = google_pubsub_topic.bigquery[each.key].name

  expiration_policy {
    ttl = "" // Subscription never expires.
  }
  bigquery_config {
    table          = "${var.project_id}.${google_bigquery_dataset.pmap.dataset_id}.${google_bigquery_table.pmap[each.key].table_id}"
    write_metadata = true
  }

  depends_on = [
    google_bigquery_dataset_iam_member.viewer,
    google_bigquery_dataset_iam_binding.editors
  ]
}

// Grant metadataViewer role required for writting to BigQuery.
// See link: https://cloud.google.com/pubsub/docs/create-subscription#assign_bigquery_service_account.
resource "google_bigquery_dataset_iam_member" "viewer" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.pmap.dataset_id
  role       = "roles/bigquery.metadataViewer"
  member     = "serviceAccount:${local.pubsub_svc_account_email}"
}

// Authoritatively grant the dataEditor role required for writting to BigQuery.
// See link: https://cloud.google.com/pubsub/docs/create-subscription#assign_bigquery_service_account.
resource "google_bigquery_dataset_iam_binding" "editors" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.pmap.dataset_id
  role       = "roles/bigquery.dataEditor"
  members    = ["serviceAccount:${local.pubsub_svc_account_email}"]
}

resource "google_storage_bucket" "pmap" {
  name                        = "pmap"
  project                     = var.project_id
  location                    = "US"
  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 2 // Delete in 30 days.
    }
  }
}

// Grant object creator role to GitHub access service account.
resource "google_storage_bucket_iam_member" "object_creator" {
  bucket = google_storage_bucket.pmap.name
  role   = "roles/storage.objectCreator"
  member = module.github_ci_infra.service_account_member
}

resource "google_storage_notification" "pmap" {
  for_each           = local.event_type
  bucket             = google_storage_bucket.pmap.name
  payload_format     = "NONE"
  topic              = google_pubsub_topic.pmap_gcs_notification[each.key].id
  event_types        = ["OBJECT_FINALIZE"]
  object_name_prefix = each.key
  depends_on         = [google_pubsub_topic_iam_binding.publishers]
}

// Enable notifications by giving the correct IAM permission to the unique service account.
data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

resource "google_pubsub_topic_iam_binding" "publishers" {
  for_each = local.event_type
  topic    = google_pubsub_topic.pmap_gcs_notification[each.key].id
  role     = "roles/pubsub.publisher"
  members  = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_pubsub_topic" "pmap_gcs_notification" {
  for_each = local.event_type
  project  = var.project_id
  name     = "${each.key}-gcs-notification"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Grant WIF service account suscriber permission to GCS notification topic.
resource "google_pubsub_topic_iam_member" "gcs_notification_subscriber" {
  for_each = local.event_type
  topic    = google_pubsub_topic.pmap_gcs_notification[each.key].id
  project  = var.project_id
  role     = "roles/pubsub.subscriber"
  member   = module.github_ci_infra.service_account_member
}

// Create a dedicated service account for pmap services to run as.
resource "google_service_account" "ci_run_service_account" {
  project      = var.project_id
  account_id   = "run-pmap-sa"
  display_name = "Cloud Run Service Account for pmap"
}
