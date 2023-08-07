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

resource "google_project_service" "services" {
  for_each = toset([
    "monitoring.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false
}

# The email channel where alert will be sent to.
resource "google_monitoring_notification_channel" "email_notification_channel" {
  project = var.project_id

  display_name = "pmap alerts email channel"

  type = "email"

  labels = {
    email_address = var.notification_channel_email
  }

  force_delete = false

  depends_on = [
    google_project_service.services["monitoring.googleapis.com"],
  ]
}

# This alert will trigger if: in a user defined rolling window size, the number
# of succeeded pmap-prober cloud job runs below the user defined threshold.
resource "google_monitoring_alert_policy" "prober_service_success_number_below_threshold" {
  project = var.project_id

  display_name = "PMAP-Prober Service Alert: Number of successed PMAP probes below threshold"

  combiner = "OR"

  # Conditions are:
  # 1. The metric is completed_execution_count
  # 2. The metrics is applied to pmap-prober
  # 3. Only count on success jobs.
  # 4. When the number of succeeded jobs below
  #    the threshold, alert will be triggered.
  conditions {
    display_name = "Too many failed PMAP probes"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/job/completed_execution_count\" resource.type=\"cloud_run_job\" resource.label.\"job_name\"=\"${google_cloud_run_v2_job.pmap_prober.name}\" AND metric.label.\"result\"=\"succeeded\""
      duration        = "0s"
      comparison      = "COMPARISON_LT"
      threshold_value = var.prober_alert_threshold
      aggregations {
        alignment_period   = var.prober_alert_align_window_size_in_seconds
        per_series_aligner = "ALIGN_COUNT"
      }
    }
  }
  notification_channels = [
    google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.alert_enabled
}

resource "google_monitoring_alert_policy" "pmap_messages_too_old" {
  for_each = toset(var.pmap_subscription_ids)

  project = var.project_id

  display_name = "Oldest unacked messages age for pmap subscription id {${each.key}} is too old[MEAN]"

  combiner = "OR"
  // It causes an alert when oldest unacked message age for pmap measured over 120-minutes intervals,
  // exceeds a threshold of xxx seconds.
  conditions {
    display_name = "Oldest unacked messages age is too old[MEAN]"
    condition_threshold {
      threshold_value = var.oldest_unacked_messages_age_threshold
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      trigger {
        count = 1
      }
      aggregations {
        per_series_aligner   = "ALIGN_MEAN"
        alignment_period     = "7200s"
        cross_series_reducer = "REDUCE_MEAN"
      }
      filter = "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\" resource.type=\"pubsub_subscription\" resource.label.\"subscription_id\"=\"${each.key}\" resource.label.\"project_id\"=\"${var.project_id}\""
    }
  }
  notification_channels = [
    google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.alert_enabled
}

resource "google_monitoring_alert_policy" "pmap_num_of_undeliverable_messages_oversized" {
  for_each = toset(var.pmap_subscription_ids)

  project = var.project_id

  display_name = "Nnumber of undeliverable messages for pmap subscription id {${each.key}} oversized[MEAN]"

  combiner = "OR"
  // It causes an alert when number of undelivered messages for pmap measured over 24h intervals,
  // exceeds a threshold of 10.
  conditions {
    display_name = "Number of undeliverable messages oversized[MEAN]"
    condition_threshold {
      threshold_value = var.num_of_undeliverable_messages_threshold
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      trigger {
        count = 1
      }
      aggregations {
        per_series_aligner   = "ALIGN_MEAN"
        alignment_period     = "86400s"
        cross_series_reducer = "REDUCE_MEAN"
      }
      filter = "metric.type=\"pubsub.googleapis.com/subscription/dead_letter_message_count\" resource.type=\"pubsub_subscription\" resource.label.\"subscription_id\"=\"${each.key}\" resource.label.\"project_id\"=\"${var.project_id}\""
    }
  }
  notification_channels = [
    google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.alert_enabled
}
