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

variable "mapping_service_image" {
  description = "The service image of mapping service."
  type        = string
}

variable "policy_service_image" {
  description = "The service image of policy service."
  type        = string
}

variable "ci_service_account" {
  description = "CI service account."
  type        = string
}
