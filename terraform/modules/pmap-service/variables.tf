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

variable "pmap_container_image" {
  description = "The container image for pmap."
  type        = string
}

variable "pmap_args" {
  type        = list(string)
  default     = []
  description = "Arguments to the pmap container's entrypoint."
}

variable "upstream_topic" {
  description = "The Pub/Sub topic for GCS bucket notifications."
  type        = string
}

variable "downstream_topic" {
  description = "The downstream Pub/Sub topic."
  type        = string
  default     = null
}

variable "downstream_failure_topic" {
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
  description = <<EOF
      "Optional GCS events subscription filter for mapping events,
      for example `attributes.objectId=\"<object_id>\"`. Can be used
      to select a subset of GCS events."
    EOF
  type        = string
  default     = null
}

variable "pmap_specific_envvars" {
  type        = map(string)
  description = <<EOT
        "The specific container envirnoment variables needed for pmap individual service.
        This can be used by both mapping and policy service.
        For example PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE is only required by
        mapping service.
        Example:
        {PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE: projects/some-project-id}"
    EOT
  default     = {}
}

variable "enable_dead_lettering" {
  description = "Enables the creation of a dead letter topic for the pub/sub push subscription."
  type        = bool
  default     = true
}
