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

// Below variables will not be needed once the schemas are determined.
variable "mapping_table_schema" {
  description = "The JSON representation of the mapping table schema."
  type        = string
}

variable "retention_table_schema" {
  description = "The JSON representation of the retention table schema."
  type        = string
}

variable "mapping_table_clustering" {
  description = "List of fields used for data clustering in mapping table."
  type        = list(string)
  default     = []
}

variable "retention_table_clustering" {
  description = "List of fields used for data clustering in retention table."
  type        = list(string)
  default     = []
}

variable "topic_schema_encoding" {
  description = "The encoding of messages validated against schema. Default value is ENCODING_UNSPECIFIED. Possible values are ENCODING_UNSPECIFIED, JSON, and BINARY."
  type        = string
  default     = "ENCODING_UNSPECIFIED"
}

variable "topic_schema_type" {
  description = "The type of the schema definition Default value is TYPE_UNSPECIFIED. Possible values are TYPE_UNSPECIFIED, PROTOCOL_BUFFER, and AVRO."
  type        = string
  default     = "TYPE_UNSPECIFIED"
}

variable "mapping_topic_schema" {
  description = "The string representation of the mapping topic schema."
  type        = string
}

variable "retention_topic_schema" {
  description = "The string representation of the retention topic schema."
  type        = string
}

variable "table_partition_field" {
  description = "The field used to determine how to create a time-based partition for both mapping and retention."
  type        = string
}
