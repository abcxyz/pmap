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

// Create ci infrastructure resources including artifact repository, workload identity pool and provider,
// and ci service account.
module "github_ci_infra" {
  source     = "github.com/abcxyz/terraform-modules/modules/github_ci_infra"
  project_id = var.project_id
  name       = "pmap"

  // Terraform apply will fail because of retricted access to this repo via github_repository,
  // could related to the repo privacy setting.
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
  dataset_id                      = "pmap_ci"
  friendly_name                   = "Privacy data annotations and mappings."
  description                     = "Dataset for data annotations and their mapping to data resources."
  location                        = "US"
  delete_contents_on_destroy      = false
  default_partition_expiration_ms = 2592000000 // 30 days.
}

resource "google_bigquery_table" "mapping" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.pmap.dataset_id
  table_id            = "mapping"
  schema              = var.mapping_table_schema
  clustering          = var.mapping_table_clustering
  deletion_protection = false

  time_partitioning {
    type  = "DAY"
    field = var.table_partition_field
  }
}

resource "google_bigquery_table" "retention" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.pmap.dataset_id
  table_id            = "retention"
  schema              = var.retention_table_schema
  clustering          = var.retention_table_clustering
  deletion_protection = false

  time_partitioning {
    type  = "DAY"
    field = var.table_partition_field
  }
}

resource "google_pubsub_topic" "mapping" {
  project = var.project_id
  name    = "mapping"
  schema_settings {
    schema   = google_pubsub_schema.mapping.id
    encoding = var.topic_schema_encoding
  }

  depends_on = [
    google_pubsub_schema.mapping,
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_topic" "retention" {
  project = var.project_id
  name    = "retention"
  schema_settings {
    schema   = google_pubsub_schema.retention.id
    encoding = var.topic_schema_encoding
  }

  depends_on = [
    google_pubsub_schema.retention,
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_schema" "mapping" {
  project    = var.project_id
  name       = "mapping"
  type       = var.topic_schema_type
  definition = var.mapping_topic_schema
}

resource "google_pubsub_schema" "retention" {
  project    = var.project_id
  name       = "retention"
  type       = var.topic_schema_type
  definition = var.retention_topic_schema
}

resource "google_pubsub_topic" "bigquery_dead_letter" {
  project = var.project_id
  name    = "pmap-bigquery-dead-letter"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Create a dummy subscription as the dead letter topic should have at least one subscription so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "bigquery_dead_letter" {
  project = var.project_id
  name    = "bigquery-dead-letter"
  topic   = google_pubsub_topic.bigquery_dead_letter.id

  depends_on = [
    google_pubsub_topic.bigquery_dead_letter
  ]
}

resource "google_pubsub_subscription" "mapping" {
  project = var.project_id
  name    = "mapping-bigquery"
  topic   = google_pubsub_topic.mapping.name
  dead_letter_policy {
    dead_letter_topic     = "projects/${var.project_id}/topics/pmap-bigquery-dead-letter"
    max_delivery_attempts = 7
  }

  bigquery_config {
    table            = "${var.project_id}.${google_bigquery_dataset.pmap.dataset_id}.${google_bigquery_table.mapping.table_id}"
    use_topic_schema = true
    write_metadata   = false
  }

  depends_on = [
    google_project_iam_member.viewer,
    google_project_iam_member.editor,
    google_pubsub_topic.bigquery_dead_letter,
    google_bigquery_table.mapping
  ]
}

resource "google_pubsub_subscription" "retention" {
  project = var.project_id
  name    = "retention-bigquery"
  topic   = google_pubsub_topic.retention.name
  dead_letter_policy {
    dead_letter_topic     = "projects/${var.project_id}/topics/pmap-bigquery-dead-letter"
    max_delivery_attempts = 7
  }

  bigquery_config {
    table            = "${var.project_id}.${google_bigquery_dataset.pmap.dataset_id}.${google_bigquery_table.retention.table_id}"
    use_topic_schema = true
    write_metadata   = false
  }

  depends_on = [
    google_project_iam_member.viewer,
    google_project_iam_member.editor,
    google_pubsub_topic.bigquery_dead_letter,
    google_bigquery_table.retention
  ]
}

// Grant permissions required for writting to BigQuery.
resource "google_project_iam_member" "viewer" {
  project = var.project_id
  role    = "roles/bigquery.metadataViewer"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_project_iam_member" "editor" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

// Grant permissions required for dead letter topic.
resource "google_pubsub_topic_iam_member" "pmap_bigquery_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.bigquery_dead_letter.id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_pubsub_subscription_iam_member" "mapping_subscriber" {
  project      = var.project_id
  subscription = google_pubsub_subscription.mapping.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_pubsub_subscription_iam_member" "retention_subscriber" {
  project      = var.project_id
  subscription = google_pubsub_subscription.retention.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_storage_bucket" "pmap" {
  name                        = "pmap-ci"
  project                     = var.project_id
  location                    = "US"
  uniform_bucket_level_access = true
  versioning {
    enabled = false
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 30 // Delete in 30 days.
    }
  }
}

// Grant object creator role to GitHub access service account.
resource "google_storage_bucket_iam_member" "members" {
  bucket = google_storage_bucket.pmap.name
  role   = "roles/storage.objectCreator"
  member = module.github_ci_infra.service_account_member
}

resource "google_storage_notification" "pmap" {
  bucket         = google_storage_bucket.pmap.name
  payload_format = "NONE" // TODO: to be determined.
  topic          = google_pubsub_topic.pmap_gcs_notification.id
  event_types    = ["OBJECT_FINALIZE"]
  depends_on     = [google_pubsub_topic_iam_binding.binding]
}

// Enable notifications by giving the correct IAM permission to the unique service account.
data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

resource "google_pubsub_topic_iam_binding" "binding" {
  topic   = google_pubsub_topic.pmap_gcs_notification.id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_pubsub_topic" "pmap_gcs_notification" {
  project = var.project_id
  name    = "pmap_gcs_notification"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Grant WIF service account suscriber permission to GCS notification topic.
resource "google_pubsub_topic_iam_member" "gcs_notification_subscriber" {
  project = var.project_id
  topic   = google_pubsub_topic.pmap_gcs_notification.id
  role    = "roles/pubsub.subscriber"
  member  = module.github_ci_infra.service_account_member
}

