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
  description = "The GCP project to host the pmap services."
  type        = string
}

variable "mapping_service_image" {
  description = "The service immage of mapping service."
  type        = string
}

variable "retention_service_image" {
  description = "The service immage of retention service."
  type        = string
}

variable "infra_project_id" {
  description = "The project id where the ci infra is hosted."
  type        = string
}

variable "commit_sha" {
  description = "Commit sha that triggered this CI deployment"
  type        = string
}
