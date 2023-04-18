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
  description = "The GCP project that hosts the pmap infra resources for CI."
  type        = string
}

variable "gcs_bucket_name" {
  description = "Globally unique GCS bucket name."
  type        = string
}

variable "event_types" {
  description = "Pmap event types."
  type        = list(string)
}

variable "static_gcs_bucket_name" {
  description = "Name for static GCS bucket."
  type        = string
}
