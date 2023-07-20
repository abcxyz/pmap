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
  success_table            = var.event
  failure_table            = "${var.event}-failure"
  tables                   = toset([local.success_table, local.failure_table])
  pubsub_svc_account_email = "service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

data "google_project" "project" {
  project_id = var.project_id
}

resource "google_project_service" "serviceusage" {
  project = var.project_id

  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "pubsub.googleapis.com",
    "iam.googleapis.com",
    "bigquery.googleapis.com"
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

resource "google_bigquery_table" "pmap" {
  for_each = local.tables

  project = var.project_id

  dataset_id          = var.dataset_id
  table_id            = each.key
  deletion_protection = var.bigquery_table_delete_protection

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
  for_each = local.tables

  project = var.project_id

  name = "${each.key}-bigquery"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_subscription" "bigquery" {
  for_each = local.tables

  project = var.project_id

  name  = "${each.key}-bigquery"
  topic = google_pubsub_topic.bigquery[each.key].name

  expiration_policy {
    ttl = "" // Subscription never expires.
  }
  bigquery_config {
    table          = "${var.project_id}.${var.dataset_id}.${google_bigquery_table.pmap[each.key].table_id}"
    write_metadata = true
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.bigquery_dead_letter[each.key].id
    max_delivery_attempts = 7
  }

  retry_policy {
    minimum_backoff = "2s"
    maximum_backoff = "300s"
  }
}

resource "google_pubsub_topic" "bigquery_dead_letter" {
  for_each = local.tables

  project = var.project_id

  name = "${each.key}-bigquery-dead-letter"

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

# A dummy subscription is required as the dead letter topic should have at
# least one subscription so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "bigquery_dead_letter" {
  for_each = local.tables

  project = var.project_id

  name  = "${each.key}-bigquery-dead-letter"
  topic = google_pubsub_topic.bigquery_dead_letter[each.key].id

  expiration_policy {
    ttl = "" # Never expire
  }
}

# Grant Pub/Sub publisher role of Pub/Sub topics to the pmap service account.
resource "google_pubsub_topic_iam_member" "publisher" {
  for_each = local.tables

  topic  = google_pubsub_topic.bigquery[each.key].id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${var.run_service_account}"
}

# The Cloud Pub/Sub service account for this project needs the publisher role to
# publish dead-lettered messages to the dead letter topic.
resource "google_pubsub_topic_iam_member" "dead_letter_publisher" {
  for_each = local.tables

  topic  = google_pubsub_topic.bigquery_dead_letter[each.key].id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${local.pubsub_svc_account_email}"
}

# The Cloud Pub/Sub service account for this project needs the subscriber role
# to forward messages from this subscription to the dead letter topic.
resource "google_pubsub_subscription_iam_member" "dead_letter_subscriber" {
  for_each = local.tables

  project = var.project_id

  subscription = google_pubsub_subscription.bigquery[each.key].id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}
