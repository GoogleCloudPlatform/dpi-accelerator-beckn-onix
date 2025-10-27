# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#-------------------------------------SQL Instance------------------------------------------------#

variable "db_instance_region" {
    type = string
    description = "Where the instance should reside"
}

variable "db_instance_version" {
    type = string
    description = "Type of DB you need"
}

variable "db_instance_name" {
    type = string
    description = "Instance Name"
}



variable "db_instance_tier" {
    type = string
    description = "Define the Instance tire. Ex: db-custom-1-4096"
}

variable "db_instance_labels" {
  type = map(string)
  description = "Labels for the DB"
}

variable "db_instance_edition" {
    type = string
    description = "ENTERPRISE or ENTERPRISE PLUS"
}

variable "db_aval_type" {
    type = string
    description = "Set REGIONAL/ZONAL availability type"
}

variable "db_instance_disk_size" {
    type = string
    description = "Size of the database"
}

variable "db_instance_disk_type" {
    type = string
    description = "Type of disk"
}

variable "db_ipv4" {
    type = bool
    description = "Whether public IPv4 should be enabled or not"
}

variable "instance_network" {
    type = string
    description = "Private Network for the instance"
}

variable "db_instance_max_connections" {
    type = number
    description = "Max Connections db instance can have"
}

variable "db_instance_cache" {
    type = bool
    description = "Whether data cache is enabled for the instance"
}