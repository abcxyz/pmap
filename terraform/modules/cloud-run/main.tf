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

resource "google_cloud_run_service" "services" {
  project  = var.project_id
  name     = var.service.name
  location = var.service.region

  template {
    spec {
      service_account_name = google_service_account.cloudrun_service_identity.email

      containers {
        image = var.service.image

        resources {
          limits = {
            cpu    = "1000m"
            memory = "1G"
          }
        }
        dynamic "env" {
          for_each = var.service.env_vars
          content {
            name  = env.key
            value = env.value
          }
        }
      }
    }

  }

  autogenerate_revision_name = true

  depends_on = [
    google_project_service.services["run.googleapis.com"],
  ]

  lifecycle {
    ignore_changes = [
      metadata[0].annotations["client.knative.dev/user-image"],
      metadata[0].annotations["run.googleapis.com/client-name"],
      metadata[0].annotations["run.googleapis.com/client-version"],
      metadata[0].annotations["run.googleapis.com/ingress-status"],
      metadata[0].annotations["serving.knative.dev/creator"],
      metadata[0].annotations["serving.knative.dev/lastModifier"],
      metadata[0].labels["cloud.googleapis.com/location"],
      template[0].metadata[0].annotations["client.knative.dev/user-image"],
      template[0].metadata[0].annotations["run.googleapis.com/client-name"],
      template[0].metadata[0].annotations["run.googleapis.com/client-version"],
      template[0].metadata[0].annotations["serving.knative.dev/creator"],
      template[0].metadata[0].annotations["serving.knative.dev/lastModifier"],
    ]
  }
}

// Dedicate a service account to cloud run services.
// See https://cloud.google.com/run/docs/securing/service-identity#per-service-identity.
resource "google_service_account" "cloudrun_service_identity" {
  project      = var.project_id
  account_id   = var.service.name
  display_name = "Dedicated Service Account for all cloud run services."
}

// Grant Pub/Sub publisher role of desired Pub/Sub topic to cloud run service account.
resource "google_pubsub_topic_iam_member" "member" {
  project = var.project_id
  topic   = var.service.publish_to_topic
  role    = "roles/pubsub.publisher"
  member  = google_service_account.cloudrun_service_identity.member
}
