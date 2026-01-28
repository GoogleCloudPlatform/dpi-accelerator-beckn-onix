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

variable "account_id" {
    type        = string
    description = "The account id that is used to generate the service account email address and a stable unique id."
}

variable "display_name" {
    type        = string
    description = "A user-specified display name of the service account."
}

variable "description" {
    type        = string
    description = "A user-specified description of the service account."
}

variable "create_ignore_already_exists" {
    type        = bool
    description = "If set to true, the service account will be created if it does not exist. If set to false, the resource will not be created if it does not exist."
    default     = false
}

