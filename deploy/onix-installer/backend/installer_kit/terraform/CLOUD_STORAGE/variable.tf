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

variable "bucket_name" {
    type = string
    description = "Name of your bucket"
}

variable "bucket_location" {
    type = string
    description = "Location for your bucket"
}

variable "bucket_storage_class" {
    type = string
    description = "Your bucket storgae class"
    default = "STANDARD"
}

variable "destroy_value" {
    type = bool
    description = "Whether to give terraform privilege to delete the bucket or not"
    default = true
}

variable "versioning_value" {
    type = bool
    description = "Whether to set object versioning or not"
    default = false
}

variable "bucket_labels" {
    type = map(string)
    description = "Labels for your bucket"
    default = {
      "name" = "beckn"
    }
}

variable "access_level" {
    type = bool
    description = "Whether to set uniform bucket level access or not"
    default = true
}