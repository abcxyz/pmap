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

module "service" {
  source                = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=main"
  project_id            = var.project_id
  name                  = var.service_name
  service_account_email = var.pmap_service_account
  image                 = var.image
  service_iam           = { "roles/run.invoker" = ["serviceAccount:${var.ci_service_account}"] }
}

// Create push subscriptions with the pmap service push endpoint.
resource "google_pubsub_subscription" "pmap" {
  project = var.project_id
  name    = module.service.service_name
  topic   = var.upstream_pubsub_topic

  // Required for Cloud Run, see https://cloud.google.com/run/docs/triggering/pubsub-push#ack-deadline.
  ack_deadline_seconds = 600

  push_config {
    push_endpoint = module.service.url
    oidc_token {
      service_account_email = var.ci_service_account
    }
  }
}

// Grant Pub/Sub publisher role of desired Pub/Sub topic to the pmap service account.
resource "google_pubsub_topic_iam_member" "publisher" {
  topic  = var.downstream_pubsub_topic
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${var.pmap_service_account}"
}

// Grant GCS object viewer permission to the pmap service.
resource "google_storage_bucket_iam_member" "objectViewer" {
  bucket = var.gcs_bucket_name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${var.pmap_service_account}"
}
