#!/bin/bash
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

# Script to assign required IAM roles to the user account for BECKN Onix Installer

# Ensure gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "Error: gcloud CLI is not installed. Please install it and try again."
    exit 1
fi

# Prompt for required details
read -p "Enter your GCP Project ID: " PROJECT_ID
read -p "Enter your User Email (e.g., your-email@domain.com): " USER_EMAIL

echo "--------------------------------------------------------"
echo "Assigning roles to $USER_EMAIL in project $PROJECT_ID..."
echo "Note: These are required even if you have the 'Owner' role."
echo "--------------------------------------------------------"

# Define the list of roles based on BECKN Onix documentation
# Reference: https://github.com/GoogleCloudPlatform/dpi-accelerator-beckn-onix/tree/main/deploy/onix-installer
ROLES=(
  "roles/iam.serviceAccountTokenCreator"    # Service Account Token Creator
  "roles/iam.serviceAccountUser"            # Service Account User
  "roles/iam.serviceAccountAdmin"           # Service Account Admin
  "roles/resourcemanager.projectIamAdmin"   # Project IAM Admin
  "roles/secretmanager.admin"               # Secret Manager Admin
  "roles/cloudsql.admin"                    # Cloud SQL Admin
  "roles/artifactregistry.admin"            # Artifact Registry Administrator
  "roles/container.clusterAdmin"            # Kubernetes Engine Cluster Admin
  "roles/storage.objectAdmin"               # Storage Object Admin
  "roles/iam.workloadIdentityPoolAdmin"      # Workload Identity Pool Admin
)

# Assign each role to the user account
for ROLE in "${ROLES[@]}"; do
    echo "Assigning $ROLE..."
    gcloud projects add-iam-policy-binding "$PROJECT_ID" \
      --member="user:$USER_EMAIL" \
      --role="$ROLE" >/dev/null \
      --condition=None
done

echo "--------------------------------------------------------"
echo "Success: All required IAM roles have been assigned to $USER_EMAIL."
echo "You can now proceed to Step 2: Create and Configure the Installer Service Account."
