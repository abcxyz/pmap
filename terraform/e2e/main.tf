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
}

module "common_infra" {
  source             = "../modules/common"
  project_id         = var.project_id
  gcs_bucket_name    = var.gcs_bucket_name
  ci_service_account = var.ci_service_account
}

module "mapping_service" {
  source                          = "../modules/pmap-service"
  project_id                      = var.project_id
  service_name                    = local.mapping_service_name
  image                           = var.mapping_service_image
  upstream_pubsub_topic           = module.common_infra.gcs_pubsub_topic[local.mapping_service_name].name
  downstream_pubsub_topic         = module.common_infra.mapping_downstream_pubsub_topics[local.mapping_service_name].name
  downstream_failure_pubsub_topic = module.common_infra.mapping_downstream_pubsub_topics["${local.mapping_service_name}-failure"].name
  pmap_service_account            = module.common_infra.run_service_account
  oidc_service_account            = module.common_infra.oidc_service_account
}

module "policy_service" {
  source                  = "../modules/pmap-service"
  project_id              = var.project_id
  service_name            = local.policy_service_name
  image                   = var.policy_service_image
  upstream_pubsub_topic   = module.common_infra.gcs_pubsub_topic[local.policy_service_name].name
  downstream_pubsub_topic = module.common_infra.policy_downstream_pubsub_topics[local.policy_service_name].name
  pmap_service_account    = module.common_infra.run_service_account
  oidc_service_account    = module.common_infra.oidc_service_account
}
