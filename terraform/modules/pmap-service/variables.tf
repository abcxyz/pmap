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
  description = "The pmap ervice name."
  type        = string
}

variable "image" {
  type        = string
  description = "The pmap service immage."
}

variable "publish_to_topic_id" {
  description = "The Pub/Sub topics to which the handlers pass the messages."
  type        = string
}

variable "subscribe_to_topic_id" {
  description = "The Pub/Sub topic for GCS bucket notifications."
  type        = string
}

variable "gcs_bucket_name" {
  description = "The GCS bucket of all pmap events."
  type        = string
}

variable "subscription_filter" {
  description = "The subscription only delivers the messages that match the filter."
  type        = string
  default     = ""
}
