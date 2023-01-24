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
    "iam.googleapis.com",
    "pubsub.googleapis.com"
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

// Dedicate a service account to cloud run service to run as.
// See https://cloud.google.com/run/docs/securing/service-identity#per-service-identity.
resource "google_service_account" "cloudrun_service_identity" {
  project      = var.project_id
  account_id   = var.service_name
  display_name = "Dedicated service account for the service to run as."
}

module "service" {
  source                = "github.com/abcxyz/terraform-modules/modules/cloud_run"
  project_id            = var.project_id
  name                  = var.service_name
  service_account_email = google_service_account.cloudrun_service_identity.email
  image                 = var.image
  service_iam           = { "roles/run.invoker" = [google_service_account.oidc_token_creator.member] }
}

// Create push subscriptions with cloud run service push endpoint.
resource "google_pubsub_subscription" "pmap" {
  project = var.project_id
  name    = var.service_name
  topic   = var.subscribe_to_topic_id
  filter  = var.subscription_filter

  // Required for Cloud Run, see https://cloud.google.com/run/docs/triggering/pubsub-push#ack-deadline.
  ack_deadline_seconds = 600
  dead_letter_policy {
    dead_letter_topic     = "projects/${var.project_id}/topics/dead-letter-${var.service_name}"
    max_delivery_attempts = 7
  }

  push_config {
    push_endpoint = module.service.url
    oidc_token {
      service_account_email = google_service_account.oidc_token_creator.email
    }
  }

  depends_on = [
    google_pubsub_topic.dead_letter_topic
  ]
}

// Delegate a service account for generating OIDC token, this service account must have invoker
// permission to destination cloud run service.
// See https://cloud.google.com/pubsub/docs/push#cloud-run.
resource "google_service_account" "oidc_token_creator" {
  project      = var.project_id
  account_id   = "oidc-${var.service_name}"
  display_name = "Service Account for generating OIDC Token for the topic."
}

resource "google_pubsub_topic" "dead_letter_topic" {
  project = var.project_id
  name    = "dead-letter-${var.service_name}"
  labels = {
    type = "dead-letter-topic"
  }

  depends_on = [
    google_project_service.services["pubsub.googleapis.com"]
  ]
}

// Create a dummy subscription as the dead letter topic should have at least one subscription
// so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "dead_letter" {
  project = var.project_id
  name    = "dead-letter-${var.service_name}"
  topic   = google_pubsub_topic.dead_letter_topic.id

  depends_on = [
    google_pubsub_topic.dead_letter_topic
  ]
}

// Grant permissions required for dead letter topic.
resource "google_pubsub_subscription_iam_member" "dead_letter_subscriber" {
  project      = var.project_id
  subscription = google_pubsub_subscription.pmap.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}

resource "google_pubsub_topic_iam_member" "dead_letter_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.dead_letter_topic.id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${local.pubsub_svc_account_email}"
}

// Grant Pub/Sub publisher role of desired Pub/Sub topic to cloud run service account.
resource "google_pubsub_topic_iam_member" "member" {
  topic  = var.publish_to_topic_id
  role   = "roles/pubsub.publisher"
  member = google_service_account.cloudrun_service_identity.member
}

// Grant GCS object viewer permission to cloud run services.
resource "google_storage_bucket_iam_member" "member" {
  bucket = var.gcs_bucket_name
  role   = "roles/storage.objectViewer"
  member = google_service_account.cloudrun_service_identity.member
}
