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
  description = "The GCP project that owns the topics and subscriptions."
}

variable "dataset_id" {
  type        = string
  description = "A unique ID for the dataset."
}

variable "dataset_location" {
  type        = string
  description = "The geographic location where the dataset should reside."
}

variable "dataset_labels" {
  type        = map(string)
  default     = {}
  description = "The labels associated with this dataset."
}

variable "default_partition_expiration_ms" {
  type        = number
  description = "The default partition expiration for all partitioned tables in the dataset, in milliseconds."
}

variable "dataset_access" {
  type = any

  # At least one owner access is required.
  default = [{
    role          = "roles/bigquery.dataOwner"
    special_group = "projectOwners"
  }]
  description = "An array of objects that define dataset access for one or more entities."
}

variable "tables" {
  type = list(object({
    table_id      = string,
    friendly_name = string
    schema        = string,
    clustering    = list(string),
    time_partitioning = object({
      field = string,
      type  = string,
    }),
    deletion_protection = bool,
    labels              = map(string),
  }))
  default     = []
  description = "A list of objects, each with a list of table properties."
}

variable "pubsub_for_bigquery" {
  description = "List of message paths information from Pub/Sub topic, subscription, to BigQuery table, and to dead letter topic."

  type = list(object({
    topic                                   = string
    pubsub_schema_definition                = string // String representaion of proto definition.
    topic_message_retention_duration        = string
    bigquery_table_id                       = string
    ack_deadline_seconds                    = number
    subscription_message_retention_duration = string
    retain_acked_messages                   = bool
    dead_letter_topic                       = string
    max_delivery_attempts                   = number
    retry_maximum_backoff                   = string
    retry_minimum_backoff                   = string
    write_metadata                          = bool
  }))
}
