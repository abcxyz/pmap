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
  short_sha                = substr(var.commit_sha, 0, 7)
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
    "compute.googleapis.com",
    "run.googleapis.com",
    "iam.googleapis.com"
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

// Create cloud run services.
module "cloud-run" {
  source     = "../modules/cloud-run"
  for_each   = { for service in var.services : service.name => service }
  project_id = var.project_id
  service = {
    name             = "${each.key}-${local.short_sha}"
    region           = "us-central1"
    image            = each.value.image
    env_vars         = each.value.env_vars
    publish_to_topic = each.value.publish_to_topic
  }
}

// Create push subscriptions with cloud run service push endpoint.
resource "google_pubsub_subscription" "push_subscriptions" {
  for_each = { for service in var.services : service.name => service }
  project  = var.project_id
  name     = "${each.key}-${local.short_sha}-push"
  topic    = each.value.subscribe_to_topic
  filter   = each.value.subscription_filter
  dead_letter_policy {
    dead_letter_topic     = "projects/${var.project_id}/topics/dead-letter-${local.short_sha}"
    max_delivery_attempts = 5
  }

  push_config {
    push_endpoint = module.cloud-run[each.key].service_url
    oidc_token {
      service_account_email = google_service_account.oidc_token_creator.email
    }
  }

  depends_on = [
    google_pubsub_topic.dead_letter_topic
  ]
}

// Delegate a service account for generating OIDC token, this service account must have invoker permission to destination cloud run service.
resource "google_service_account" "oidc_token_creator" {
  project      = var.project_id
  account_id   = "oidc-${local.short_sha}"
  display_name = "Service Account for generating OIDC Token for the topic."
}

resource "google_pubsub_topic" "dead_letter_topic" {
  project = var.project_id
  name    = "dead-letter-${local.short_sha}"
  labels = {
    type = "dead-letter-topic"
  }

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Create a dummy subscription as the dead letter topic should have at least one subscription so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "dead_letter" {
  project = var.project_id
  name    = "dead-letter-sub-${local.short_sha}"
  topic   = google_pubsub_topic.dead_letter_topic.id

  depends_on = [
    google_pubsub_topic.dead_letter_topic
  ]
}

// Grant permissions required for dead letter topic.
resource "google_pubsub_topic_iam_member" "publisher" {
  for_each = { for service in var.services : service.name => service }

  project = var.project_id
  topic   = each.value.subscribe_to_topic
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_pubsub_subscription_iam_member" "subscriber" {
  for_each = { for service in var.services : service.name => service }

  project      = var.project_id
  subscription = google_pubsub_subscription.push_subscriptions[each.key].id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}

// Grant invoker roles to cloud run invokers.
resource "google_cloud_run_service_iam_binding" "bindings" {
  for_each = { for service in var.services : service.name => service }

  project  = var.project_id
  service  = "${each.key}-${local.short_sha}"
  location = "us-central1"
  role     = "roles/run.invoker"
  members  = [google_service_account.oidc_token_creator.member]

  depends_on = [
    module.cloud-run
  ]
}

// Grant Pub/Sub publisher role of desired Pub/Sub topic to cloud run service account.
resource "google_pubsub_topic_iam_member" "member" {
  for_each = { for service in var.services : service.name => service }

  project = var.project_id
  topic   = each.value.publish_to_topic
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${module.cloud-run[each.key].cloud_run_service_account}"
}

// Grant GCS object viewer permission to cloud run services.
resource "google_storage_bucket_iam_member" "member" {
  for_each = { for service in var.services : service.name => service }

  bucket = each.value.gcs_bucket_name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${module.cloud-run[each.key].cloud_run_service_account}"
}
