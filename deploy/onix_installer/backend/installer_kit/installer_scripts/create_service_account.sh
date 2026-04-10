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


# Detect active GCP project
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: No active GCP project found. Please run 'gcloud config set project [PROJECT_ID]'." >&2
    exit 1
fi

# Prompt for Service Account name
read -p "Enter desired Service Account name (e.g., dpi-installer-sa): " SA_NAME

if [ -z "$SA_NAME" ]; then
    echo "❌ Error: Service Account name cannot be empty." >&2
    exit 1
fi

SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Check if the service account already exists (Idempotency)
if gcloud iam service-accounts describe "$SA_EMAIL" --project "$PROJECT_ID" &>/dev/null; then
    echo "✅ Service account $SA_EMAIL already exists. Proceeding to role check..." >&2
else
    # Create the service account
    echo ">> Creating service account $SA_NAME in project $PROJECT_ID..." >&2
    if ! gcloud iam service-accounts create "$SA_NAME" \
        --project "$PROJECT_ID" \
        --display-name "DPI Installer Service Account"; then
        echo "❌ Error: Failed to create service account." >&2
        exit 1
    fi
    echo "✅ Service account created: $SA_EMAIL" >&2
fi

# Define the list of roles to assign
ROLES=(
  "roles/compute.networkAdmin"
  "roles/compute.loadBalancerAdmin"
  "roles/compute.instanceAdmin.v1"
  "roles/container.admin"
  "roles/iam.serviceAccountAdmin"
  "roles/iam.serviceAccountTokenCreator"
  "roles/resourcemanager.projectIamAdmin"
  "roles/iam.serviceAccountUser"
  "roles/storage.admin"
  "roles/cloudsql.admin"
  "roles/redis.admin"
  "roles/pubsub.admin"
  "roles/secretmanager.admin"
  "roles/dns.admin"
  "roles/compute.securityAdmin"
  "roles/aiplatform.admin"
  "roles/run.admin"
  "roles/iam.workloadIdentityPoolAdmin"
  "roles/serviceusage.serviceUsageAdmin"
  "roles/discoveryengine.admin"
)

# Function to check if a role is assigned to the service account
function is_role_assigned() {
  local role=$1
  gcloud projects get-iam-policy "$PROJECT_ID" \
    --flatten="bindings[].members" \
    --format='value(bindings.role)' \
    --filter="bindings.members:serviceAccount:$SA_EMAIL AND bindings.role='$role'" 2>/dev/null | grep -Fxq "$role"
}

# Assign each role to the service account with retry logic
MAX_RETRIES=3

echo "Assigning roles to $SA_EMAIL..." >&2
for ROLE in "${ROLES[@]}"; do
    # Pre-assignment check
    if is_role_assigned "$ROLE"; then
        echo "✅ Role $ROLE is already assigned. Skipping." >&2
        continue
    fi

    RETRY_COUNT=0
    SUCCESS=false

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        echo ">> Assigning $ROLE (Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)..." >&2
        gcloud projects add-iam-policy-binding "$PROJECT_ID" \
          --member="serviceAccount:$SA_EMAIL" \
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

echo "Service Account $SA_EMAIL has been created and all roles have been assigned." >&2

# Output the service account email to stdout so it can be captured by the calling script.
echo "$SA_EMAIL"