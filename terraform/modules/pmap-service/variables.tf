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

variable "project_id" {
  description = "The GCP project to host the pmap service and its PubSub subscription."
  type        = string
}

variable "service_name" {
  description = "The pmap service name."
  type        = string
}

variable "image" {
  description = "The pmap service image."
  type        = string
}

variable "upstream_pubsub_topic" {
  description = "The Pub/Sub topic for GCS bucket notifications."
  type        = string
}

variable "downstream_pubsub_topic" {
  description = "The downstream Pub/Sub topic."
  type        = string
  default     = null
}

variable "downstream_failure_pubsub_topic" {
  description = "The downstream Pub/Sub topic for failure events."
  type        = string
  default     = null
}

variable "pmap_service_account" {
  description = "Service account for the pmap Cloud Run service to run as."
  type        = string
}

variable "oidc_service_account" {
  description = <<EOT
        Service Account used for generating the OIDC tokens. Required to enable request
        authentication when messages from Pub/Sub are delivered to push endpoints. If the
        endpoint is a Cloud Run service, this service account needs to be the run invoker.
    EOT
  type        = string
}

variable "gcs_events_filter" {
  description = "Optional GCS events subscription filter, for example `attributes.objectID=\"<object_id>\"`. Can be used to select a subset of GCS events."
  type        = string
  default     = null
}
