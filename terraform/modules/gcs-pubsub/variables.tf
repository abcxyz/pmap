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

variable "gcs_bucket_name" {
  type        = string
  description = "The name of the GCS bucket."
}

variable "project_id" {
  type        = string
  description = "The GCP project that owns the GCS bucket."
}

variable "gcs_bucket_location" {
  type        = string
  description = "The location of the GCS bucket."
}

variable "gcs_bucket_labels" {
  type        = map(string)
  default     = null
  description = "A set of key/value label pairs to assign to the GCS bucket."
}

variable "gcs_bucket_retention_policy" {
  type = object({
    is_locked        = bool
    retention_period = number
  })
  default     = null
  description = "Configuration of the bucket's data retention policy for how long objects in the bucket should be retained."
}

variable "gcs_bucket_iam_role_to_member" {
  type = list(object({
    role   = string
    member = string
  }))
  default     = []
  description = "The list of IAM members to grant permissions on the GCS bucket."
}

variable "gcs_bucket_lifecycle_rules" {
  description = "The GCS bucket's Lifecycle Rules configuration."
  type = list(object({
    # Object with keys:
    # - type - The type of the action of this Lifecycle Rule. Supported values: Delete and SetStorageClass.
    # - storage_class - (Required if action type is SetStorageClass) The target Storage Class of objects affected by this Lifecycle Rule.
    action = any

    # Object with keys:
    # - age - (Optional) Minimum age of an object in days to satisfy this condition.
    # - created_before - (Optional) Creation date of an object in RFC 3339 (e.g. 2017-06-13) to satisfy this condition.
    # - with_state - (Optional) Match to live and/or archived objects. Supported values include: "LIVE", "ARCHIVED", "ANY".
    # - matches_storage_class - (Optional) Storage Class of objects to satisfy this condition. Supported values include: MULTI_REGIONAL, REGIONAL, NEARLINE, COLDLINE, STANDARD, DURABLE_REDUCED_AVAILABILITY.
    # - matches_prefix - (Optional) One or more matching name prefixes to satisfy this condition.
    # - matches_suffix - (Optional) One or more matching name suffixes to satisfy this condition
    # - num_newer_versions - (Optional) Relevant only for versioned objects. The number of newer versions of an object to satisfy this condition.
    condition = any
  }))
  default = []
}

variable "pubsub_for_gcs_notification" {
  description = "An object in this list contains information required to create a GCS notification and a Pub/Sub topic and subscription associated with it."

  type = list(object({
    topic                            = string
    notification_payload_format      = string
    object_name_prefix               = string
    notification_event_types         = list(string)
    topic_message_retention_duration = string
  }))
}

variable "subscriber_service_account" {
  type        = string
  description = "Service account that should need to attach subscriptions to topics."
}
