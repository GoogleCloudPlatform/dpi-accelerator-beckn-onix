# Copyright 2026 Google LLC
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

resource "google_sql_user" "user" {
    name     = var.user_name
    instance = var.instance_name
    type = var.user_type
    password = var.password

    lifecycle {
        precondition {
        # The condition must evaluate to true for Terraform to proceed
        condition     = var.user_type != "BUILT_IN" || var.password != null
        error_message = "You must provide a 'password' when 'user_type' is set to 'BUILT_IN'."
        }
    }

    deletion_policy = "ABANDON"
}