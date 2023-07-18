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

  common_env_vars = {
    "PROJECT_ID" : var.project_id,
    "PMAP_SUCCESS_TOPIC_ID" : var.downstream_topic,
    "PMAP_FAILURE_TOPIC_ID" : var.downstream_failure_topic
  }
}

data "google_project" "project" {
  project_id = var.project_id
}

module "service" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=46d3ffd82d7c3080bc5ec2cc788fe3e21176a8be"

  project_id = var.project_id

  name                  = var.service_name
  service_account_email = var.pmap_service_account
  image                 = var.pmap_container_image
  args                  = var.pmap_args

  service_iam = {
    admins     = []
    developers = []
    invokers   = ["serviceAccount:${var.oidc_service_account}"]
  }

  envvars = merge(local.common_env_vars, var.pmap_specific_envvars)
}

// Create push subscriptions with the pmap service push endpoint.
resource "google_pubsub_subscription" "pmap" {
  project = var.project_id

  name   = module.service.service_name
  topic  = var.upstream_topic
  filter = var.gcs_events_filter

  // Required for Cloud Run, see https://cloud.google.com/run/docs/triggering/pubsub-push#ack-deadline.
  ack_deadline_seconds = 600

  push_config {
    push_endpoint = module.service.url
    oidc_token {
      service_account_email = var.oidc_service_account
    }
  }

  dynamic "dead_letter_policy" {
    for_each = var.enable_dead_lettering ? [1] : []
    content {
      dead_letter_topic     = google_pubsub_topic.gcs_dead_letter[0].id
      max_delivery_attempts = 7
    }
  }
}

resource "google_pubsub_topic" "gcs_dead_letter" {
  count = var.enable_dead_lettering ? 1 : 0
  project = var.project_id
  name  = "${module.service.service_name}-gcs-dead-letter"
}

# A dummy subscription is required as the dead letter topic should have at
# least one subscription so that dead-lettered messages will not be lost.
resource "google_pubsub_subscription" "gcs_dead_letter" {
  count = var.enable_dead_lettering ? 1 : 0
  project = var.project_id

  name  = "${module.service.service_name}-gcs-dead-letter"
  topic = google_pubsub_topic.gcs_dead_letter[0].name

  expiration_policy {
    ttl = "" # Never expire
  }
}

# The Cloud Pub/Sub service account for this project needs the publisher role to
# publish dead-lettered messages to the dead letter topic.
resource "google_pubsub_topic_iam_member" "dead_letter_publisher" {
  count = var.enable_dead_lettering ? 1 : 0
  project = var.project_id

  topic  = google_pubsub_topic.gcs_dead_letter[0].id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${local.pubsub_svc_account_email}"
}

# The Cloud Pub/Sub service account for this project needs the subscriber role
# to forward messages from this subscription to the dead letter topic.
resource "google_pubsub_subscription_iam_member" "dead_letter_subscriber" {
  count = var.enable_dead_lettering ? 1 : 0
  project = var.project_id

  subscription = google_pubsub_subscription.pmap.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${local.pubsub_svc_account_email}"
}
