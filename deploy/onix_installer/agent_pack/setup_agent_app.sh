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

# Safely extracts a value from the .env file.
# Handles trailing comments and trims leading/trailing whitespace.
extract_env() {
    local key="$1"
    local file="$2"
    # 1. grep the line starting with key=
    # 2. sed to remove the key= prefix
    # 3. sed to remove inline comments (# ...) but only if they are not inside quotes.
    #    This regex removes everything from the last unquoted # to the end of the line.
    # 4. xargs to trim leading/trailing whitespace
    grep "^${key}=" "$file" | sed -e "s/^${key}=//" -e 's/[[:space:]]*#.*$//' | xargs
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
export GOOGLE_CLOUD_PROJECT=$(extract_env "GOOGLE_CLOUD_PROJECT" "$ENV_FILE")
export REGION=$(extract_env "REGION" "$ENV_FILE")
export APP_NAME=$(extract_env "APP_NAME" "$ENV_FILE")
export STAGING_BUCKET=$(extract_env "STAGING_BUCKET" "$ENV_FILE")

if [ -z "$GOOGLE_CLOUD_PROJECT" ] || [ -z "$REGION" ] || [ -z "$APP_NAME" ]; then
    echo "❌ Error: Missing required variables in agent_config.env."
    echo "Please ensure GOOGLE_CLOUD_PROJECT, REGION, and APP_NAME are defined."
    exit 1
fi

# Ensure APP_NAME length is at most 6 characters
if [ "${#APP_NAME}" -gt 6 ]; then
    echo "❌ Error: APP_NAME ($APP_NAME) exceeds the maximum length of 6 characters."
    echo "Please update APP_NAME in agent_config.env to be 6 characters or fewer."
    exit 1
fi

# Derive STAGING_BUCKET if not provided
if [ -z "$STAGING_BUCKET" ]; then
    STAGING_BUCKET="gs://${GOOGLE_CLOUD_PROJECT}-dpi-${APP_NAME}-bucket"
    echo ">> STAGING_BUCKET not found in config, auto-deriving: $STAGING_BUCKET"
fi
echo "✅ Configuration loaded."

# 2.5 Verify Blueprint Repository
echo ">> Verifying Agent Blueprint Repository..."

if [ ! -d "$SCRIPT_DIR/dpi_agent_blueprint" ]; then
    echo "❌ Error: 'dpi_agent_blueprint' directory not found in agent_pack."
    echo "Please ensure you have cloned the 'dpi_agent_blueprint' repository into:"
    echo "$SCRIPT_DIR/dpi_agent_blueprint"
    exit 1
fi
echo "✅ Agent Blueprint Repository found."

# 3. Authentication
echo ">> Authenticating with Google Cloud..."
gcloud auth login
gcloud config set project "$GOOGLE_CLOUD_PROJECT"
echo "✅ Authenticated."

# 4. Service Account Configuration
echo "========================================================="
echo "        Service Account Configuration                    "
echo "========================================================="
SA_EMAIL=""
read -p "Do you already have a service account with the required permissions? (Y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]?$ ]]; then
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


# 5. Python Environment & Rendering
echo ">> Setting up Python environment..."
if [ ! -d "$SCRIPT_DIR/venv" ]; then
    python3 -m venv "$SCRIPT_DIR/venv"
fi
source "$SCRIPT_DIR/venv/bin/activate"
pip install -r "$SCRIPT_DIR/dpi_agent_blueprint/requirements.txt" jinja2 google-cloud-discoveryengine

if [ ! -f "$RENDER_SCRIPT" ]; then
    echo "❌ Error: Rendering script not found at $RENDER_SCRIPT."
    exit 1
fi

echo ">> Generating Terraform .tfvars..."
python3 -u "$RENDER_SCRIPT" || exit 1

if [ ! -f "$OUT_FILE" ]; then
    echo "❌ Error: Failed to generate $OUT_FILE."
    exit 1
fi
echo "✅ Terraform configuration generated."


# 6. Terraform Execution (with retry logic)
echo ">> Navigating to Terraform directory: $TF_DIR"
cd "$TF_DIR"

# --- Remote Backend Configuration ---
BUCKET_NAME="${GOOGLE_CLOUD_PROJECT}-dpi-${APP_NAME}-bucket"
TARGET_REGION="${REGION:-asia-south1}"

echo ">> Configuring Remote Terraform State in GCS..."
echo "Checking if bucket gs://${BUCKET_NAME} exists..."

if gsutil ls -b "gs://${BUCKET_NAME}" >/dev/null 2>&1; then
    echo "✅ Bucket gs://${BUCKET_NAME} already exists."
else
    echo "Bucket does not exist. Creating gs://${BUCKET_NAME} in ${TARGET_REGION}..."
    if gsutil mb -p "$GOOGLE_CLOUD_PROJECT" -l "$TARGET_REGION" "gs://${BUCKET_NAME}"; then
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

# Read SESSION_DB_TYPE from agent_config.env
SESSION_DB_TYPE=$(extract_env "SESSION_DB_TYPE" "$ENV_FILE")
SESSION_DB_TYPE=${SESSION_DB_TYPE:-agent-engine}

echo ">> Extracting Infrastructure Details (Mode: $SESSION_DB_TYPE)..."

# Define all required outputs
REDIS_HOST=$(extract_output "redis_instance_ip") || exit 1
NETWORK_ATTACHMENT_ID=$(extract_output "agent_network_attachment_id") || exit 1
AGENT_SA_EMAIL=$(extract_output "agent_app_service_account_email") || exit 1
AGENT_PROXY_IP=$(extract_output "agent_proxy_internal_ip") || exit 1
AGENT_INVOKER_SA_EMAIL=$(extract_output "agent_invoker_service_account_email") || exit 1

if [ "$SESSION_DB_TYPE" == "database" ]; then
    DB_HOST=$(extract_output "db_instance_private_ip_address") || exit 1
    DB_USER=$(extract_output "agent_db_user") || exit 1
    DB_PASS=$(extract_output "agent_db_password") || exit 1
    DB_NAME=$(extract_output "agent_db_name") || exit 1
    SESSION_DB_URL="postgresql+asyncpg://${DB_USER}:${DB_PASS}@${DB_HOST}:5432/${DB_NAME}"
else
    echo ">> SESSION_DB_TYPE is '$SESSION_DB_TYPE', skipping database output extraction."
    DB_HOST=""
    DB_USER=""
    DB_PASS=""
    DB_NAME=""
    SESSION_DB_URL=""
fi

echo ">> Extracting Datastore IDs for ingestion..."
AGENT_DATASTORE_IDS=$(extract_output "agent_datastore_ids") || echo "{}"

# We extract the DATASTORE_IMPORTS value directly.
DATASTORE_IMPORTS=$(extract_env "DATASTORE_IMPORTS" "$ENV_FILE")
# Strip single and double quotes at the ends (only the outer wrapper)
DATASTORE_IMPORTS="${DATASTORE_IMPORTS#\'}"
DATASTORE_IMPORTS="${DATASTORE_IMPORTS%\'}"
DATASTORE_IMPORTS="${DATASTORE_IMPORTS#\"}"
DATASTORE_IMPORTS="${DATASTORE_IMPORTS%\"}"

if [ "$AGENT_DATASTORE_IDS" != "{}" ] && [ "$DATASTORE_IMPORTS" != "{}" ]; then
    echo ">> Triggering Document Ingestion..."
    if ! python3 -u "$SCRIPT_DIR/ingest_datastore.py" "$GOOGLE_CLOUD_PROJECT" "$AGENT_DATASTORE_IDS" "$DATASTORE_IMPORTS"; then
        echo "❌ Error: Datastore ingestion failed. Halting installation."
        exit 1
    fi
else
    echo ">> No datastores to ingest."
fi

echo "✅ All required infrastructure outputs retrieved successfully."

export REDIS_HOST SESSION_DB_TYPE SESSION_DB_URL DB_HOST DB_USER DB_PASS DB_NAME NETWORK_ATTACHMENT_ID AGENT_SA_EMAIL AGENT_INVOKER_SA_EMAIL

ABS_BLUEPRINT_ENV="$ENV_FILE"

echo ">> Generating Consolidated env-vars Override..."
CONSOLIDATED_ENV="$SCRIPT_DIR/consolidated.env"
> "$CONSOLIDATED_ENV"

# 1. Parse the blueprint env file
grep -v '^#' "$ABS_BLUEPRINT_ENV" | grep -v '^[[:space:]]*$' >> "$CONSOLIDATED_ENV"
echo "STAGING_BUCKET=$STAGING_BUCKET" >> "$CONSOLIDATED_ENV"
echo "REDIS_HOST=$REDIS_HOST" >> "$CONSOLIDATED_ENV"
if [ "$SESSION_DB_TYPE" == "database" ]; then
    echo "SESSION_DB_URL=$SESSION_DB_URL" >> "$CONSOLIDATED_ENV"
fi
echo "NETWORK_ATTACHMENT_ID=$NETWORK_ATTACHMENT_ID" >> "$CONSOLIDATED_ENV"
echo "AGENT_SA_EMAIL=$AGENT_SA_EMAIL" >> "$CONSOLIDATED_ENV"
echo "AGENT_ENGINE_LOCATION=$REGION" >> "$CONSOLIDATED_ENV"
echo "OTEL_OBSERVABILITY_LOG_NAME=dpi-${APP_NAME}-agent-traces" >> "$CONSOLIDATED_ENV"

if [ -n "$AGENT_PROXY_IP" ]; then
    echo ">> Injecting explicit proxy configuration into env vars..."
    echo "HTTP_PROXY=http://$AGENT_PROXY_IP:8888" >> "$CONSOLIDATED_ENV"
    echo "HTTPS_PROXY=http://$AGENT_PROXY_IP:8888" >> "$CONSOLIDATED_ENV"
    echo "NO_PROXY=169.254.169.254,metadata.google.internal,localhost,127.0.0.1,googleapis.com,*.googleapis.com,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" >> "$CONSOLIDATED_ENV"
fi

echo ">> Injecting Datastore IDs into environment variables..."
if [ "$AGENT_DATASTORE_IDS" != "{}" ]; then
    # Use jq to iterate over the keys and values.
    # We replace dots (agri.biochar_advice) with underscores (agri_biochar_advice) and uppercase them (AGRI_BIOCHAR_ADVICE_DATASTORE_ID)
    echo "$AGENT_DATASTORE_IDS" | jq -r 'to_entries | .[] | "\(.key | gsub("\\."; "_") | ascii_upcase)_DATASTORE_ID=\(.value)"' >> "$CONSOLIDATED_ENV"
    echo "✅ Injected $(echo "$AGENT_DATASTORE_IDS" | jq 'length') Datastore IDs."
fi

echo "✅ Generated $CONSOLIDATED_ENV successfully."
echo ">> Deploying Application via Agent Engine..."

cd "$SCRIPT_DIR"

echo ">> Copying consolidated.env to dpi_agent_blueprint/.env ..."
cp "$CONSOLIDATED_ENV" "$SCRIPT_DIR/dpi_agent_blueprint/.env"


STATE_FILE="$INSTALLER_ROOT/backend/installer_kit/installer_state.json"
EXISTING_AGENT_ID=$(jq -r '.agent_engine_id // empty' "$STATE_FILE" 2>/dev/null || echo "")

if [ -z "$EXISTING_AGENT_ID" ]; then
    echo ">> No existing Agent Engine ID found. Initiating --create flow."
    python3 -u app_sdk.py --create --env-vars-file="$CONSOLIDATED_ENV" || exit 1
else
    echo ">> Found existing Agent Engine ID: $EXISTING_AGENT_ID. Initiating --update flow."
    # Inject it into env so app_sdk picks it up (optional, but good for local debugging)
    echo "AGENT_ENGINE_ID=$EXISTING_AGENT_ID" >> "$CONSOLIDATED_ENV"
    python3 -u app_sdk.py --update --agent-engine="$EXISTING_AGENT_ID" --env-vars-file="$CONSOLIDATED_ENV" || exit 1
fi

echo "========================================================="
echo "✅ Agent App Deployment Complete!"
echo "========================================================="

echo "========================================================="
echo "        Starting Phase 2 Deployment"
echo "========================================================="

POST_DEPLOY_AGENT_ID=$(jq -r '.agent_engine_id // empty' "$STATE_FILE" 2>/dev/null || echo "")

if [ -z "$POST_DEPLOY_AGENT_ID" ]; then
    echo "❌ Error: Could not determine final Agent Engine ID. Skipping Phase 2."
else
    echo ">> Agent Engine ID: $POST_DEPLOY_AGENT_ID"
    # Agent Engine endpoints are namespaced differently than just the resource ID alone.
    AGENT_ENGINE_ENDPOINT="https://${REGION}-aiplatform.googleapis.com/v1/${POST_DEPLOY_AGENT_ID}:query"

    # Identify the correct Adapter Topic
    ADAPTER_TOPIC=$(extract_env "ADAPTER_TOPIC_ID" "$ENV_FILE" || echo "")

    if [ -z "$ADAPTER_TOPIC" ]; then
        echo ">> No ADAPTER_TOPIC_ID provided in env, resolving from Phase 1 Outputs..."
        # Extract output handles missing gracefully if wrapped properly.
        RAW_TOPIC_OUTPUT=$(jq -r '.adapter_topic_full_id.value // empty' "$TF_DIR/outputs.json" 2>/dev/null || echo "")

        if [ -n "$RAW_TOPIC_OUTPUT" ] && [ "$RAW_TOPIC_OUTPUT" != "null" ]; then
             ADAPTER_TOPIC="$RAW_TOPIC_OUTPUT"
        fi
    fi

    cd "$TF_DIR/deployments/agent_phase2"

    ENABLE_ADAPTER_INTEGRATION="false"
    if [ -z "$ADAPTER_TOPIC" ]; then
         echo ">> couldn't resolve ADAPTER_TOPIC_ID. Skipping Adapter Pub/Sub Integration in Phase 2..."
    else
         ENABLE_ADAPTER_INTEGRATION="true"
         echo ">> Deploying Phase 2 Subscription connecting $ADAPTER_TOPIC to $AGENT_ENGINE_ENDPOINT ..."
    fi

    if [ "$ENABLE_ADAPTER_INTEGRATION" = "true" ]; then
         cat <<EOF > agent_p2.tfvars
project_id                 = "${GOOGLE_CLOUD_PROJECT}"
app_name                   = "${APP_NAME}"
agent_invoker_sa_email     = "${AGENT_INVOKER_SA_EMAIL}"
enable_adapter_integration = ${ENABLE_ADAPTER_INTEGRATION}
adapter_topic_id           = "${ADAPTER_TOPIC}"
agent_engine_endpoint_url  = "${AGENT_ENGINE_ENDPOINT}"
EOF

         echo ">> Configuring Remote Terraform State for Phase 2..."
         cat <<EOF > backend.tf
terraform {
  backend "gcs" {
    bucket  = "${BUCKET_NAME}"
    prefix  = "terraform/state/agent_phase2"
  }
}
EOF

         echo ">> Initializing Phase 2 Terraform..."
         terraform init

         echo ">> Applying Phase 2 Terraform..."
         terraform apply -var-file="agent_p2.tfvars" -auto-approve

         terraform output -json > phase2_outputs.json
         DLQ_TOPIC=$(jq -r '.agent_dlq_topic_name.value // empty' phase2_outputs.json 2>/dev/null || echo "")

         echo "✅ Phase 2 Deployment Complete!"
    fi
fi

echo ""
echo "========================================================="
echo "✅ Agent App Deployment Script Finished!"
echo "========================================================="
if [ -n "$POST_DEPLOY_AGENT_ID" ]; then
    echo "Agent Engine ID: $POST_DEPLOY_AGENT_ID"
    echo "Agent Engine URL: $AGENT_ENGINE_ENDPOINT"
    echo "Agent Invoker SA: $AGENT_INVOKER_SA_EMAIL"
    echo "Dead Letter Topic: $DLQ_TOPIC"
echo "========================================================="
fi
