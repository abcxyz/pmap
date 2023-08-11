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

locals {
  mapping_service_name = "mapping"
  policy_service_name  = "policy"

  alert_subscription_map = {
    mapping_bigquery         = module.common_infra.bigquery_subscriptions[local.mapping_service_name].event_subscription_id,
    mapping_bigquery_failure = module.common_infra.bigquery_subscriptions[local.mapping_service_name].failure_subscription_id,
    policy_bigquery          = module.common_infra.bigquery_subscriptions[local.policy_service_name].event_subscription_id,
    policy_bigquery_failure  = module.common_infra.bigquery_subscriptions[local.policy_service_name].failure_subscription_id,
    mapping_gcs              = module.mapping_service.gcs_notification_subscription_id,
    policy_gcs               = module.policy_service.gcs_notification_subscription_id
  }

}

module "common_infra" {
  source = "../modules/common"

  project_id = var.project_id

  gcs_bucket_name                  = var.gcs_bucket_name
  event_types                      = [local.mapping_service_name, local.policy_service_name]
  bigquery_table_delete_protection = var.bigquery_table_delete_protection
}

module "mapping_service" {
  source = "../modules/pmap-service"

  project_id = var.project_id

  service_name             = local.mapping_service_name
  pmap_container_image     = var.pmap_container_image
  pmap_args                = ["mapping", "server"]
  upstream_topic           = module.common_infra.gcs_notification_topics[local.mapping_service_name].name
  downstream_topic         = module.common_infra.bigquery_topics[local.mapping_service_name].event_topic
  downstream_failure_topic = module.common_infra.bigquery_topics[local.mapping_service_name].failure_topic
  pmap_service_account     = module.common_infra.run_service_account
  oidc_service_account     = module.common_infra.oidc_service_account
  gcs_events_filter        = var.mapping_gcs_events_filter
  pmap_specific_envvars    = var.pmap_specific_envvars
}

module "policy_service" {
  source = "../modules/pmap-service"

  project_id = var.project_id

  service_name         = local.policy_service_name
  pmap_container_image = var.pmap_container_image
  pmap_args            = ["policy", "server"]
  upstream_topic       = module.common_infra.gcs_notification_topics[local.policy_service_name].name
  downstream_topic     = module.common_infra.bigquery_topics[local.policy_service_name].event_topic
  pmap_service_account = module.common_infra.run_service_account
  oidc_service_account = module.common_infra.oidc_service_account
  gcs_events_filter    = var.policy_gcs_events_filter
}

module "monitoring" {
  source = "../modules/monitoring"

  project_id = var.project_id

  prober_bucket_id           = var.gcs_bucket_name
  prober_bigquery_dataset_id = module.common_infra.bigquery_dataset
  prober_mapping_table_id    = local.mapping_service_name
  prober_policy_table_id     = local.policy_service_name
  pmap_prober_image          = var.pmap_prober_image
  notification_channel_email = var.notification_channel_email
  pmap_subscription_ids      = local.alert_subscription_map
}
