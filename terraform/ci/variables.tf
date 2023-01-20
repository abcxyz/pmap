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
  type        = string
  description = "The GCP project to host the cloud run services."
}

variable "services" {
  type = list(object({
    name                = string
    image               = string
    env_vars            = map(string)
    publish_to_topic    = string
    gcs_bucket_name     = string
    subscribe_to_topic  = string
    subscription_filter = string
  }))
  description = "The cloud run services to create."
}

variable "commit_sha" {
  type        = string
  description = "Commit sha that triggered this CI deployment"
}
