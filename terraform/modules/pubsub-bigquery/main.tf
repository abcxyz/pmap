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
  success_table = var.event
  failure_table = "${var.event}-failure"
  tables        = toset([local.success_table, local.failure_table])
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
}

// Grant Pub/Sub publisher role of Pub/Sub topics to the pmap service account.
resource "google_pubsub_topic_iam_member" "publisher" {
  for_each = local.tables

  topic  = google_pubsub_topic.bigquery[each.key].id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${var.run_service_account}"
}
