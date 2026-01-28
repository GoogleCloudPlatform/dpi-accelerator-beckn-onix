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


# Check if gcloud is installed
# if ! command -v gcloud &> /dev/null; then
#     echo "Error: gcloud CLI is not installed. Please install it and try again." >&2
#     exit 1
# fi

# Authenticate with Google Cloud
# echo -e "\nGoogle Cloud Authentication Required\n" >&2
# gcloud auth login
# echo -e "\nAuthentication successful.\n" >&2

# Prompt for required details
read -p "Enter your GCP Project ID: " PROJECT_ID
read -p "Enter desired Service Account name (e.g., beckn-adapter-sa-<user-name>): " SA_NAME

# Create the service account
echo "Creating service account $SA_NAME in project $PROJECT_ID..." >&2
gcloud iam service-accounts create "$SA_NAME" \
    --project "$PROJECT_ID" \
    --display-name "BECKN Adapter Service Account"

# Construct the service account email
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
echo "Service Account created: $SA_EMAIL" >&2

# Define the list of roles to assign
ROLES=(
  "roles/redis.admin"
  "roles/cloudtrace.agent"
  "roles/compute.instanceAdmin.v1"
  "roles/compute.networkAdmin"
  "roles/container.clusterAdmin"
  "roles/resourcemanager.projectIamAdmin"
  "roles/pubsub.admin"
  "roles/pubsub.publisher"
  "roles/secretmanager.admin"
  "roles/secretmanager.secretAccessor"
  "roles/iam.securityAdmin"
  "roles/iam.serviceAccountAdmin"
  "roles/iam.serviceAccountTokenCreator"
  "roles/storage.admin"
  "roles/compute.securityAdmin"
  "roles/compute.loadBalancerAdmin"
  "roles/container.admin"
  "roles/logging.admin"
  "roles/monitoring.admin"
  "roles/iam.serviceAccountUser"
  "roles/cloudsql.admin"
  "roles/storage.objectAdmin"
  "roles/dns.admin"
)

# Assign each role to the service account
echo "Assigning roles to $SA_EMAIL..." >&2
for ROLE in "${ROLES[@]}"; do
    echo "Assigning $ROLE..." >&2
    gcloud projects add-iam-policy-binding "$PROJECT_ID" \
      --member="serviceAccount:$SA_EMAIL" \
      --role="$ROLE" >/dev/null \
      --condition=None
done


echo "Service Account $SA_EMAIL has been created and all roles have been assigned." >&2

# Output the service account email to stdout so it can be captured by the calling script.
echo "$SA_EMAIL"