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


# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "\nError: $1 is not installed. Please install it before proceeding.\n"
        exit 1
    fi
}

echo -e "\nChecking system requirements...\n"

# Check for required tools
check_command "gcloud"
check_command "terraform"
check_command "git"
check_command "helm"
check_command "kubectl"

echo -e "\nAll required tools are installed. Proceeding...\n"

# if [ "$STATE_CHOICE" == "2" ]; then
#     read -p "Enter GCS bucket name: " GCS_BUCKET
#     cat <<EOF > backend.tf
# terraform {
#   backend "gcs" {
#     bucket  = "$GCS_BUCKET"
#     prefix  = "terraform/state"
#   }
# }
# EOF
#     echo -e "\nConfigured remote backend in GCS bucket: $GCS_BUCKET\n"
# else
#     cat <<EOF > backend.tf
# terraform {
#   backend "local" {}
# }
# EOF
#     echo -e "\nUsing local backend for Terraform state storage.\n"
# fi

# Always initialize Terraform with retry logic
MAX_RETRIES=1
RETRY_DELAY=10
ATTEMPT=1

while [ $ATTEMPT -le $MAX_RETRIES ]; do
    echo -e "\nInitializing Terraform (Attempt $ATTEMPT)...\n"
    terraform init && break
    echo -e "\nTerraform init failed. Retrying in $RETRY_DELAY seconds...\n"
    sleep $RETRY_DELAY
    ((ATTEMPT++))
done

if [ $ATTEMPT -gt $MAX_RETRIES ]; then
    echo -e "\nError: Terraform init failed after $MAX_RETRIES attempts. Exiting.\n"
    exit 1
fi

# Apply Terraform with retry logic
ATTEMPT=1
while [ $ATTEMPT -le $MAX_RETRIES ]; do
    echo -e "\nApplying Terraform configuration (Attempt $ATTEMPT)...\n"
    terraform apply --var-file=generated-terraform.tfvars -auto-approve&& break
    # terraform plan --var-file=generated-terraform.tfvars  && break
    echo -e "\nTerraform apply failed. Retrying in $RETRY_DELAY seconds...\n"
    sleep $RETRY_DELAY
    ((ATTEMPT++))
done

if [ $ATTEMPT -gt $MAX_RETRIES ]; then
    echo -e "\nError: Terraform apply failed after $MAX_RETRIES attempts. Exiting.\n"
    exit 1
fi

echo -e "\nInfrastructure deployment successful!\n"

# Save Terraform outputs to outputs.json for deploy-app.sh to use
echo -e "\nSaving Terraform outputs to outputs.json...\n"
terraform output -json > outputs.json

if [ $? -ne 0 ]; then
    echo -e "\nError: Failed to save Terraform outputs. Exiting.\n"
    exit 1
fi
echo -e "\nTerraform outputs saved to outputs.json.\n"

# Call the schema_loader script
echo -e "\nExecuting schema loader script...\n"
if ./../installer_scripts/schema_loader/schema_loader.sh; then
    echo -e "\nSchema loading completed successfully!\n"
else
    echo -e "\nError: schema_loader/schema_loader.sh failed. Please check its output for details.\n"
    exit 1
fi
