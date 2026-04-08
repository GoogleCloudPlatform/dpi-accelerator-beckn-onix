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

# Script to enable and verify required Google Cloud APIs for BECKN Onix Installer

# Ensure gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: gcloud CLI is not installed. Please install it and try again." >&2
    exit 1
fi

# Prompt for required details
read -p "Enter your GCP Project ID: " PROJECT_ID

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: Project ID cannot be empty." >&2
    exit 1
fi

APIS=(
  "compute.googleapis.com"
  "container.googleapis.com"
  "cloudresourcemanager.googleapis.com"
  "servicenetworking.googleapis.com"
  "secretmanager.googleapis.com"
  "redis.googleapis.com"
  "sqladmin.googleapis.com"
  "artifactregistry.googleapis.com"
)

echo "--------------------------------------------------------"
echo "ℹ️ Enabling required APIs in project $PROJECT_ID..." >&2
echo "--------------------------------------------------------"

# Enable all APIs in one go (Optimization)
gcloud services enable "${APIS[@]}" --project="$PROJECT_ID"

if [ $? -ne 0 ]; then
    echo "❌ Error: Failed to enable APIs." >&2
    exit 1
fi

echo "--------------------------------------------------------"
echo "ℹ️ Verifying enabled APIs individually..." >&2
echo "--------------------------------------------------------"

ALL_SUCCESS=true

for API in "${APIS[@]}"; do
    # Verify each API individually for accuracy by calling gcloud
    if gcloud services list --enabled --project="$PROJECT_ID" --filter="name:$API" --format="value(name)" | grep -q "$API"; then
        echo "✅ API $API is enabled." >&2
    else
        echo "❌ API $API is NOT enabled." >&2
        ALL_SUCCESS=false
    fi
done

echo "--------------------------------------------------------"
if [ "$ALL_SUCCESS" = true ]; then
    echo "✅ Success: All required APIs are enabled." >&2
else
    echo "⚠️ Warning: Some APIs could not be verified as enabled." >&2
fi
echo "--------------------------------------------------------"
