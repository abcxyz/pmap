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
  github_slug = "abcxyz/pmap"
}

provider "google" {
  project = var.project_id
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
    "storage.googleapis.com"
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

// Create a single GCS bucket.
resource "google_storage_bucket" "bucket" {
  name                        = var.gcs_bucket_name
  project                     = var.project_id
  location                    = var.gcs_bucket_location
  uniform_bucket_level_access = true
  labels                      = var.gcs_bucket_labels
  retention_policy {
    is_locked        = var.gcs_bucket_retention_policy.is_locked
    retention_period = var.gcs_bucket_retention_policy.retention_period
  }
  versioning {
    enabled = false
  }

  dynamic "lifecycle_rule" {
    for_each = var.gcs_bucket_lifecycle_rules
    content {
      action {
        type          = lifecycle_rule.value.action.type
        storage_class = lookup(lifecycle_rule.value.action, "storage_class", null)
      }
      condition {
        age                        = lookup(lifecycle_rule.value.condition, "age", null)
        created_before             = lookup(lifecycle_rule.value.condition, "created_before", null)
        with_state                 = lookup(lifecycle_rule.value.condition, "with_state", lookup(lifecycle_rule.value.condition, "is_live", false) ? "LIVE" : null)
        matches_storage_class      = contains(keys(lifecycle_rule.value.condition), "matches_storage_class") ? split(",", lifecycle_rule.value.condition["matches_storage_class"]) : null
        matches_prefix             = contains(keys(lifecycle_rule.value.condition), "matches_prefix") ? split(",", lifecycle_rule.value.condition["matches_prefix"]) : null
        matches_suffix             = contains(keys(lifecycle_rule.value.condition), "matches_suffix") ? split(",", lifecycle_rule.value.condition["matches_suffix"]) : null
        num_newer_versions         = lookup(lifecycle_rule.value.condition, "num_newer_versions", null)
        custom_time_before         = lookup(lifecycle_rule.value.condition, "custom_time_before", null)
        days_since_custom_time     = lookup(lifecycle_rule.value.condition, "days_since_custom_time", null)
        days_since_noncurrent_time = lookup(lifecycle_rule.value.condition, "days_since_noncurrent_time", null)
        noncurrent_time_before     = lookup(lifecycle_rule.value.condition, "noncurrent_time_before", null)
      }
    }
  }
}

resource "google_storage_bucket_iam_member" "additional_members" {
  for_each = {
    for m in var.gcs_bucket_iam_role_to_member : "${m.role} ${m.member}" => m
  }
  bucket = google_storage_bucket.bucket.name
  role   = each.value.role
  member = each.value.member
}

// Grant object creator role to GitHub access service account.
resource "google_storage_bucket_iam_member" "members" {
  bucket = google_storage_bucket.bucket.name
  role   = "roles/storage.objectCreator"
  member = google_service_account.gh-access-acc.member
}

// Grant external identities to impersonate the GitHub Access Account via workload identity federation.
resource "google_service_account_iam_member" "external_provider_roles" {
  service_account_id = google_service_account.gh-access-acc.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${module.workload-identity-federation.pool_name}/attribute.repository/${local.github_slug}"
}

resource "google_service_account" "gh-access-acc" {
  project      = var.project_id
  account_id   = "gh-access-sa"
  display_name = "GitHub Access Account"
}

// Cannot destroy and recreate as the pool name cannot be reused even it is deleted.
module "workload-identity-federation" {
  source      = "github.com/abcxyz/pkg//terraform/modules/workload-identity-federation"
  project_id  = var.project_id
  github_slug = local.github_slug
}

resource "google_storage_notification" "notification" {
  for_each           = { for path in var.pubsub_for_gcs_notification : path.topic => path }
  bucket             = google_storage_bucket.bucket.name
  payload_format     = each.value.notification_payload_format
  topic              = google_pubsub_topic.topics[each.key].name
  object_name_prefix = each.value.object_name_prefix
  event_types        = each.value.notification_event_types
  depends_on         = [google_pubsub_topic_iam_binding.binding]
}

// Enable notifications by giving the correct IAM permission to the unique service account.
data "google_storage_project_service_account" "gcs_account" {
  project = var.project_id
}

resource "google_pubsub_topic_iam_binding" "binding" {
  for_each = { for path in var.pubsub_for_gcs_notification : path.topic => path }
  topic    = google_pubsub_topic.topics[each.key].id
  role     = "roles/pubsub.publisher"
  members  = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}

resource "google_pubsub_topic" "topics" {
  for_each                   = { for path in var.pubsub_for_gcs_notification : path.topic => path }
  project                    = var.project_id
  name                       = each.key
  message_retention_duration = each.value.topic_message_retention_duration

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

resource "google_pubsub_topic_iam_member" "publisher" {
  for_each = { for topic in google_pubsub_topic.topics : topic.name => topic }

  project = var.project_id
  topic   = each.value.id
  role    = "roles/pubsub.subscriber"
  member  = "serviceAccount:${var.subscriber_service_account}"
}
