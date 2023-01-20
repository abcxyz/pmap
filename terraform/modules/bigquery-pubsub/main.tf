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
  pubsub_svc_account_email = "service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"

  iam_to_primitive = {
    "roles/bigquery.dataOwner" : "OWNER"
    "roles/bigquery.dataEditor" : "WRITER"
    "roles/bigquery.dataViewer" : "READER"
  }
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
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "pubsub.googleapis.com",
    "iam.googleapis.com",
    "bigquery.googleapis.com"
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

// Create BigQuery dataset and tables.
resource "google_bigquery_dataset" "dataset" {
  project                         = var.project_id
  dataset_id                      = var.dataset_id
  friendly_name                   = "Privacy data annotations and mappings"
  description                     = "Dataset for data annotations and their mapping to data resources."
  location                        = var.dataset_location
  delete_contents_on_destroy      = false
  default_partition_expiration_ms = var.default_partition_expiration_ms
  labels                          = var.dataset_labels

  dynamic "access" {
    for_each = var.dataset_access
    content {
      # BigQuery API converts IAM to primitive roles in its backend.
      # This causes Terraform to show a diff on every plan that uses IAM equivalent roles.
      # Thus, do the conversion between IAM to primitive role here to prevent the diff.
      role = lookup(local.iam_to_primitive, access.value.role, access.value.role)

      # Additionally, using null as a default value would lead to a permanant diff
      # See https://github.com/hashicorp/terraform-provider-google/issues/4085#issuecomment-516923872
      domain         = lookup(access.value, "domain", "")
      group_by_email = lookup(access.value, "group_by_email", "")
      user_by_email  = lookup(access.value, "user_by_email", "")
      special_group  = lookup(access.value, "special_group", "")
    }
  }
}

resource "google_bigquery_table" "tables" {
  for_each            = { for table in var.tables : table.table_id => table }
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.dataset.dataset_id
  friendly_name       = each.value.friendly_name
  table_id            = each.value.table_id
  labels              = each.value.labels
  schema              = each.value.schema
  clustering          = each.value.clustering
  deletion_protection = each.value.deletion_protection

  time_partitioning {
    type  = each.value.time_partitioning.type
    field = each.value.time_partitioning.field
  }
}

resource "google_pubsub_topic" "topics" {
  for_each                   = { for path in var.pubsub_for_bigquery : path.topic => path }
  project                    = var.project_id
  name                       = each.key
  message_retention_duration = each.value.topic_message_retention_duration
  schema_settings {
    schema   = google_pubsub_schema.schema[each.key].id
    encoding = "BINARY"
  }

  depends_on = [
    google_pubsub_schema.schema,
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_schema" "schema" {
  for_each   = { for path in var.pubsub_for_bigquery : path.topic => path }
  project    = var.project_id
  name       = "${each.key}-schema"
  type       = "PROTOCOL_BUFFER"
  definition = each.value.pubsub_schema_definition
}

resource "google_pubsub_topic" "dead_letter_topics" {
  for_each = toset(distinct([for path in var.pubsub_for_bigquery : path.dead_letter_topic]))
  project  = var.project_id
  name     = each.key
  labels = {
    type = "dead-letter-topic"
  }

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Create a dummy subscription as the dead letter topic should have at least one subscription so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "dummy_subscription" {
  for_each = toset(distinct([for path in var.pubsub_for_bigquery : path.dead_letter_topic]))
  project  = var.project_id
  name     = "${each.key}-subscription"
  topic    = each.key

  depends_on = [
    google_pubsub_topic.dead_letter_topics
  ]
}

resource "google_pubsub_subscription" "bigquery_subscriptions" {
  for_each                   = { for path in var.pubsub_for_bigquery : path.topic => path }
  project                    = var.project_id
  name                       = "${each.key}-bigquery-subscription"
  topic                      = google_pubsub_topic.topics[each.key].name
  ack_deadline_seconds       = each.value.ack_deadline_seconds
  message_retention_duration = each.value.subscription_message_retention_duration
  retain_acked_messages      = each.value.retain_acked_messages
  dead_letter_policy {
    dead_letter_topic     = "projects/${var.project_id}/topics/${each.value.dead_letter_topic}"
    max_delivery_attempts = each.value.max_delivery_attempts
  }
  retry_policy {
    maximum_backoff = each.value.retry_maximum_backoff
    minimum_backoff = each.value.retry_minimum_backoff
  }

  bigquery_config {
    table            = "${var.project_id}.${google_bigquery_dataset.dataset.dataset_id}.${each.value.bigquery_table_id}"
    use_topic_schema = true
    write_metadata   = each.value.write_metadata
  }

  depends_on = [
    google_project_iam_member.viewer,
    google_project_iam_member.editor,
    google_pubsub_topic.dead_letter_topics,
    google_bigquery_table.tables
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
resource "google_pubsub_topic_iam_member" "publisher" {
  for_each = { for topic in google_pubsub_topic.dead_letter_topics : topic.name => topic }

  project = var.project_id
  topic   = each.value.id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_pubsub_subscription_iam_member" "subscriber" {
  for_each = { for path in var.pubsub_for_bigquery : path.topic => path }

  project      = var.project_id
  subscription = google_pubsub_subscription.bigquery_subscriptions[each.key].id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}
