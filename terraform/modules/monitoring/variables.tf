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
  description = "The GCP project that host the pmap service."
  type        = string
}

variable "prober_bucket_id" {
  description = "The bucket id where prober will upload files to."
  type        = string
}

variable "prober_bigquery_dataset_id" {
  description = "The ID of the bigquery dataset where prober run queries from."
  type        = string
}

variable "prober_mapping_table_id" {
  description = "The ID of the bigquery table which stores the resource mapping result."
  type        = string
}

variable "prober_policy_table_id" {
  description = "The ID of the bigquery table which stores the policy result."
  type        = string
}

variable "prober_query_retry_wait_duartion" {
  description = "The wait duation between the next bigquery query retry attempt."
  type        = string
  default     = "5s"
}

variable "prober_query_retry_count" {
  description = "The max number for bigquery query retry attempt."
  type        = string
  default     = "5"
}

variable "prober_mapping_gcs_bucket_prefix" {
  description = "The file name prefix for mapping."
  type        = string
  default     = "mapping/prober"
}

variable "prober_policy_gcs_bucket_prefix" {
  description = "The file name prefix for policy."
  type        = string
  default     = "policy/prober"
}

variable "prober_scheduler" {
  type        = string
  description = "How often the prober service should be triggered, default is every 30 minutes. Learn more at: https://cloud.google.com/scheduler/docs/configuring/cron-job-schedules?&_ga=2.26495481.-578386315.1680561063#defining_the_job_schedule."
  default     = "*/30 * * * *"
}

variable "pmap_prober_image" {
  type        = string
  description = "Docker image for pmap prober."
}

variable "prober_alert_threshold" {
  type        = number
  description = "Send alert for Prober-Service when the number of succeeded prober runs below the threshold."
  default     = 2
}

variable "prober_alert_align_window_size_in_seconds" {
  type        = string
  description = "The sliding window size for counting failed prober job runs. Format example: 600s."
  default     = "3600s"
}

variable "alert_enabled" {
  type        = bool
  description = "True if alerts are enabled, otherwise false."
  default     = false
}

variable "notification_channel_email" {
  type        = string
  description = "The Email address where alert notifications send to."
}

variable "log_level" {
  type        = string
  description = "Log level for writting logs"
  default     = "DEBUG"
}
