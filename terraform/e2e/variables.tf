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
  description = "The GCP project to host the pmap services and other resources created during CI."
  type        = string
}

variable "pmap_container_image" {
  description = "The container image for pmap."
  type        = string
}

variable "gcs_bucket_name" {
  description = "Globally unique GCS bucket name."
  type        = string
}

variable "mapping_gcs_events_filter" {
  description = <<EOF
      "Optional GCS events subscription filter for mapping events,
      for example `attributes.objectId=\"<object_id>\"`. Can be used
      to select a subset of GCS events."
    EOF
  type        = string
  default     = null
}

variable "policy_gcs_events_filter" {
  description = <<EOF
      "Optional GCS events subscription filter for mapping events,
      for example `attributes.objectId=\"<object_id>\"`. Can be used
      to select a subset of GCS events."
    EOF
  type        = string
  default     = null
}

variable "bigquery_table_delete_protection" {
  description = <<EOF
      Whether or not to allow Terraform to destroy the BigQuery table instances.
      By default it is false. If set to true, a terraform destroy or terraform
      apply that would delete the instance will fail.
    EOF
  type        = bool
  default     = false
}

variable "extra_container_env_vars" {
  type        = map(string)
  description = <<EOT
        "The extra envirnoment variables for mapping services as key value pair.
        Including <PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE>: 
        The scope for where the resources resides in.
        Options can be one of the following:
        projects/{PROJECT_ID}
        projects/{PROJECT_NUMBER}
        folders/{FOLDER_NUMBER}
        organizations/{ORGANIZATION_NUMBER}"
        Example:
        {PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE: }
    EOT
  default     = {}
}
