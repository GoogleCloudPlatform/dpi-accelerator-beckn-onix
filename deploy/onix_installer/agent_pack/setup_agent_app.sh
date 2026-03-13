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

set -e

# --- Configuration ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
INSTALLER_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$SCRIPT_DIR/agent_config.env"
RENDER_SCRIPT="$SCRIPT_DIR/render_agent_config.py"
TF_DIR="$INSTALLER_ROOT/backend/installer_kit/terraform"
OUT_FILE="$TF_DIR/generated-terraform.tfvars"


# --- Helper Functions ---

check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "❌ Error: Required command '$1' is not installed."
        return 1
    fi
    return 0
}

validate_prerequisites() {
    echo "--- Checking Prerequisites ---"
    local missing_tools=()
    local tools=("gcloud" "terraform" "python3" "pip" "docker" "jq")

    for tool in "${tools[@]}"; do
        if ! check_command "$tool"; then
            missing_tools+=("$tool")
        fi
    done

    if [ ${#missing_tools[@]} -ne 0 ]; then
        echo "❌ Error: The following tools are missing: ${missing_tools[*]}"
        echo "Please install them and run the script again."
        exit 1
    fi
    echo "✅ All prerequisites are met."
    echo "------------------------------"
    echo
}

cleanup() {
    echo
    echo "--- Cleanup ---"
    if [ -d "$SCRIPT_DIR/venv" ]; then
        echo "Deactivating Python virtual environment..."
        deactivate 2>/dev/null || true
    fi
    echo "Script execution finished."
}

# Extracts a Terraform output value from outputs.json, exiting if not found or empty
extract_output() {
    local key="$1"
    local value
    # Capture output from outputs.json using jq
    value=$(jq -r ".${key}.value // empty" outputs.json 2>/dev/null || echo "")

    if [ -z "$value" ]; then
        echo "❌ Error: Required Terraform output '$key' is missing or empty in outputs.json." >&2
        return 1
    fi
    echo "$value"
}


trap cleanup EXIT

# --- Main Execution ---

echo "========================================================="
echo "        Agent App Installer                              "
echo "========================================================="

# 1. Validation
validate_prerequisites

# 2. Configuration Load
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Error: Configuration file 'agent_config.env' not found in $INSTALLER_ROOT."
    echo "Please create it using 'agent_config.env.example' as a guide."
    exit 1
fi

echo ">> Extracting core variables for gcloud auth and deployment..."
# Extract the required core variables safely
export PROJECT_ID=$(grep -v '^#' "$ENV_FILE" | grep "^PROJECT_ID=" | cut -d '=' -f2 | tr -d '"'\'' ')
export REGION=$(grep -v '^#' "$ENV_FILE" | grep "^REGION=" | cut -d '=' -f2 | tr -d '"'\'' ')
export APP_NAME=$(grep -v '^#' "$ENV_FILE" | grep "^APP_NAME=" | cut -d '=' -f2 | tr -d '"'\'' ')
export AGENT_IMAGE_URL=$(grep -v '^#' "$ENV_FILE" | grep "^AGENT_IMAGE_URL=" | cut -d '=' -f2 | tr -d '"'\'' ')

if [ -z "$PROJECT_ID" ] || [ -z "$REGION" ] || [ -z "$APP_NAME" ] || [ -z "$AGENT_IMAGE_URL" ]; then
    echo "❌ Error: Missing required variables in agent_config.env."
    echo "Please ensure PROJECT_ID, REGION, APP_NAME, and AGENT_IMAGE_URL are defined."
    exit 1
fi
echo "✅ Configuration loaded."

# 3. Python Environment & Rendering
echo ">> Setting up Python environment..."
if [ ! -d "$SCRIPT_DIR/venv" ]; then
    python3 -m venv "$SCRIPT_DIR/venv"
fi
source "$SCRIPT_DIR/venv/bin/activate"
pip install -q jinja2

if [ ! -f "$RENDER_SCRIPT" ]; then
    echo "❌ Error: Rendering script not found at $RENDER_SCRIPT."
    exit 1
fi

echo ">> Generating Terraform .tfvars..."
python3 "$RENDER_SCRIPT"

if [ ! -f "$OUT_FILE" ]; then
    echo "❌ Error: Failed to generate $OUT_FILE."
    exit 1
fi
echo "✅ Terraform configuration generated."

# 4. Authentication
echo ">> Authenticating with Google Cloud..."
gcloud auth login
gcloud config set project "$PROJECT_ID"
echo "✅ Authenticated."

# 5. Service Account Configuration
echo "========================================================="
echo "        Service Account Configuration                    "
echo "========================================================="
SA_EMAIL=""
read -p "Do you already have a service account with the required permissions? (y/N) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    while [ -z "$SA_EMAIL" ]; do
        read -p "Enter the email of the service account to use: " SA_EMAIL
        if [ -z "$SA_EMAIL" ]; then
            echo "Service account email cannot be empty. Please try again."
        fi
    done
else
    echo "A service account is required to provision resources."
    echo "The script to create a new service account will now be executed."

    CREATE_SA_SCRIPT="$INSTALLER_ROOT/backend/installer_kit/installer_scripts/create_service_account.sh"

    if [ ! -f "$CREATE_SA_SCRIPT" ]; then
        echo "❌ Error: '$CREATE_SA_SCRIPT' not found."
        exit 1
    fi

    SA_EMAIL=$(bash "$CREATE_SA_SCRIPT")
    echo
    echo "✅ The service account has been created successfully."
fi

echo
echo "Will proceed using service account: $SA_EMAIL"
echo "---------------------------------------------------------"

echo ">> Setting up Service Account Impersonation..."
gcloud auth application-default login --impersonate-service-account="$SA_EMAIL"
echo "✅ Impersonation configured successfully."


# 6. Terraform Execution (with retry logic)
echo ">> Navigating to Terraform directory: $TF_DIR"
cd "$TF_DIR"

# --- Remote Backend Configuration ---
BUCKET_NAME="dpi-${APP_NAME}-bucket"
TARGET_REGION="${REGION:-asia-south1}"

echo ">> Configuring Remote Terraform State in GCS..."
echo "Checking if bucket gs://${BUCKET_NAME} exists..."

if gsutil ls -b "gs://${BUCKET_NAME}" >/dev/null 2>&1; then
    echo "✅ Bucket gs://${BUCKET_NAME} already exists."
else
    echo "Bucket does not exist. Creating gs://${BUCKET_NAME} in ${TARGET_REGION}..."
    if gsutil mb -p "$PROJECT_ID" -l "$TARGET_REGION" "gs://${BUCKET_NAME}"; then
        echo "✅ Bucket created successfully."
    else
        echo "❌ Error: Failed to create GCS bucket gs://${BUCKET_NAME}."
        exit 1
    fi

    # Enable versioning for state safety
    gsutil versioning set on "gs://${BUCKET_NAME}"
fi

cat <<EOF > backend.tf
terraform {
  backend "gcs" {
    bucket  = "${BUCKET_NAME}"
    prefix  = "terraform/state"
  }
}
EOF
echo "✅ backend.tf generated."

echo ">> Initializing Terraform..."
terraform init

echo ">> Applying Terraform configuration..."
MAX_RETRIES=1
RETRY_DELAY=10
ATTEMPT=1

while [ $ATTEMPT -le $MAX_RETRIES ]; do
    echo "Attempt $ATTEMPT of $MAX_RETRIES..."
    # Step 5: Terraform Apply
    echo ">> Running Terraform Apply..."
    if terraform apply -var-file="$OUT_FILE" -auto-approve; then
        echo "✅ Terraform apply successful."
        break
    else
        echo "⚠️ Terraform apply failed. Retrying in $RETRY_DELAY seconds..."
        sleep $RETRY_DELAY
        ((ATTEMPT++))
    fi
done

if [ $ATTEMPT -gt $MAX_RETRIES ]; then
    echo "❌ Error: Terraform apply failed after $MAX_RETRIES attempts."
    exit 1
fi

# 7. Extract Outputs
echo ">> Generating JSON Output Trace..."
terraform output -json > outputs.json
echo "✅ Outputs saved to outputs.json"

echo ">> Extracting Infrastructure Details..."

# Define all required outputs
REDIS_HOST=$(extract_output "redis_instance_ip") || exit 1
AGENT_ENGINE_ID=$(extract_output "agent_reasoning_engine_id") || exit 1
DB_HOST=$(extract_output "db_instance_private_ip_address") || exit 1
DB_USER=$(extract_output "agent_db_user") || exit 1
DB_PASS=$(extract_output "agent_db_password") || exit 1
DB_NAME=$(extract_output "agent_db_name") || exit 1
AGENT_APP_NAME=$(extract_output "agent_service_name") || exit 1

echo "✅ All infrastructure outputs retrieved successfully."

# Construct the SQLAlchemy connection URL
SESSION_DB_URL="postgresql+asyncpg://${DB_USER}:${DB_PASS}@${DB_HOST}:5432/${DB_NAME}"

export REDIS_HOST SESSION_DB_URL AGENT_ENGINE_ID DB_HOST DB_USER DB_PASS DB_NAME

ABS_BLUEPRINT_ENV="$ENV_FILE"

echo ">> Generating Cloud Run env-vars Override..."
CLOUD_RUN_ENV="cloud_run_env.env"
> "$CLOUD_RUN_ENV"

# 1. Parse the blueprint env file
grep -v '^#' "$ABS_BLUEPRINT_ENV" | grep -v '^[[:space:]]*$' >> "$CLOUD_RUN_ENV"
echo "REDIS_HOST=$REDIS_HOST" >> "$CLOUD_RUN_ENV"
echo "SESSION_DB_TYPE=database" >> "$CLOUD_RUN_ENV"
echo "SESSION_DB_URL=$SESSION_DB_URL" >> "$CLOUD_RUN_ENV"
echo "DB_HOST=$DB_HOST" >> "$CLOUD_RUN_ENV"
echo "DB_USER=$DB_USER" >> "$CLOUD_RUN_ENV"
echo "DB_NAME=$DB_NAME" >> "$CLOUD_RUN_ENV"
echo "AGENT_ENGINE_ID=$AGENT_ENGINE_ID" >> "$CLOUD_RUN_ENV"
echo "OTEL_OBSERVABILITY_LOG_NAME=$APP_NAME" >> "$CLOUD_RUN_ENV"
echo "GOOGLE_CLOUD_PROJECT=$PROJECT_ID" >> "$CLOUD_RUN_ENV"
echo "GOOGLE_CLOUD_MEMORY_SERVICE_LOCATION=$REGION" >> "$CLOUD_RUN_ENV"

echo "✅ Generated $CLOUD_RUN_ENV successfully."
echo ">> Deploying Application via gcloud..."

gcloud run deploy "$AGENT_APP_NAME" \
    --image="$AGENT_IMAGE_URL" \
    --env-vars-file="$CLOUD_RUN_ENV" \
    --region="$REGION" \
    --project="$PROJECT_ID"

echo "========================================================="
echo "✅ Agent App Deployment Complete!"
echo "========================================================="
