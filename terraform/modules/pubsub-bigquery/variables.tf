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
  description = "The GCP project that hosts the pmap infra resources for CI."
  type        = string
}

variable "dataset_id" {
  description = "The name of the BigQuery dataset ."
  type        = string
}

variable "event" {
  description = "The pmap event type such as mapping and policy."
  type        = string
}

variable "destination_tables" {
  description = "List of destination BigQuery table IDs to be created."
  type        = list(string)
}

variable "run_service_account" {
  description = "The service account that the Cloud Run service run as."
  type        = string
}
