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

resource "google_service_account" "service_account" {
    account_id = var.account_id
    # The account id that is used to generate the service account email address and a stable unique id.

    display_name = var.display_name
    # A user-specified display name of the service account.

    description = var.description
    # A user-specified description of the service account.

    create_ignore_already_exists = var.create_ignore_already_exists
    # If set to true, the service account will be created if it does not exist. If set to false, the resource will not be created if it does not exist.
    # Default is false.
}

