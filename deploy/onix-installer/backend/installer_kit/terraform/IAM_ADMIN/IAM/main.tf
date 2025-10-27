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

/**
resource "google_project_iam_binding" "project" {
    project = var.project_id
    role = var.role

    members = var.members
    # List of members that needs to be assigned the role
}
**/



resource "google_project_iam_member" "project" {
    project = var.project_id
    role = var.member_role

    member = var.member
    # Member that needs to be assigned the role
  
}