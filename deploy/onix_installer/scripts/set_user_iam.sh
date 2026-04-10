#!/bin/bash
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

# Script to assign required IAM roles to the user account for BECKN Onix Installer

# Ensure gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: gcloud CLI is not installed. Please install it and try again." >&2
    exit 1
fi

# Prompt for required details
read -p "Enter your GCP Project ID: " PROJECT_ID
read -p "Enter your User Email (e.g., your-email@domain.com): " USER_EMAIL

echo "--------------------------------------------------------"
echo "ℹ️ Assigning roles to $USER_EMAIL in project $PROJECT_ID..." >&2
echo "ℹ️ Note: These are required even if you have the 'Owner' role." >&2
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
  "roles/container.admin"                   # Kubernetes Engine Admin
  "roles/storage.admin"                     # Storage Admin
  "roles/iam.workloadIdentityPoolAdmin"     # Workload Identity Pool Admin
  "roles/serviceusage.serviceUsageAdmin"    # Service Usage Admin
  "roles/aiplatform.admin"                  # Vertex AI Administrator
)

# Function to check if a role is assigned
function is_role_assigned() {
  local role=$1
  gcloud projects get-iam-policy "$PROJECT_ID" \
    --flatten="bindings[].members" \
    --format='value(bindings.role)' \
    --filter="bindings.members:user:$USER_EMAIL AND bindings.role='$role'" 2>/dev/null | grep -Fxq "$role"
}

# Assign each role to the user account with retry logic
MAX_RETRIES=3

for ROLE in "${ROLES[@]}"; do
    if is_role_assigned "$ROLE"; then
        echo "✅ Role $ROLE is already assigned. Skipping." >&2
        continue
    fi

    RETRY_COUNT=0
    SUCCESS=false

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        echo "ℹ️ Assigning $ROLE (Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)..." >&2
        gcloud projects add-iam-policy-binding "$PROJECT_ID" \
          --member="user:$USER_EMAIL" \
          --role="$ROLE" >/dev/null \
          --condition=None

        # Verify if assignment was successful
        if is_role_assigned "$ROLE"; then
            echo "✅ Successfully assigned $ROLE." >&2
            SUCCESS=true
            break
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "⚠️ Warning: Role $ROLE assignment verification failed. Retrying in 2 seconds..." >&2
                sleep 2
            fi
        fi
    done

    if [ "$SUCCESS" = false ]; then
        echo "❌ Error: Failed to assign role $ROLE after $MAX_RETRIES attempts." >&2
        exit 1
    fi
done

echo "--------------------------------------------------------"
echo "✅ Success: All required IAM roles have been assigned to $USER_EMAIL." >&2
