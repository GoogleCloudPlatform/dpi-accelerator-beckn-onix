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


OUTPUTS_FILE="outputs.json"


# PROJECT_ID="$1"
# INSTANCE_NAME="$2"
# DB_NAME="$3"
# IAM_DB_USER="$4"

PROJECT_ID=$(jq -r '.project_id.value' "$OUTPUTS_FILE")
INSTANCE_NAME=$(jq -r '.registry_db_instance_name.value' "$OUTPUTS_FILE")
if [[ "$INSTANCE_NAME" == "null" || -z "$INSTANCE_NAME" ]]; then
  exit 0
fi
DB_NAME=$(jq -r '.registry_database_name.value' "$OUTPUTS_FILE")
IAM_DB_USER=$(jq -r '.database_user_sa_email.value' "$OUTPUTS_FILE")
REGISTRY_ADMIN_DATABASE_USER=$(jq -r '.registry_admin_database_user_sa_email.value' "$OUTPUTS_FILE")
REGION=$(jq -r '.cluster_region.value' "$OUTPUTS_FILE")
# --------------------------------------
# CONFIGURATION FOR SCHEMA LOAD
# --------------------------------------

# REGION="asia-south1"
TMP_DB_USER="schema_loader_user"
TMP_DB_PASS=$(openssl rand -base64 16)
INIT_SQL="./../installer_scripts/schema_loader/init.sql"
PROXY_PORT=5431
INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${INSTANCE_NAME}"

echo $INSTANCE_CONNECTION_NAME

# --------------------------------------
# TOOL CHECKS
# --------------------------------------
command -v gcloud >/dev/null || { echo "❌ gcloud CLI not found."; exit 1; }
command -v psql >/dev/null || { echo "❌ psql not found."; exit 1; }

# --------------------------------------
# CLEANUP FUNCTION
# --------------------------------------
cleanup() {
  echo "🧹 Cleaning up..."
  if [[ -n "$PROXY_PID" ]]; then
    echo "⛔ Stopping Cloud SQL Proxy..."
    kill "$PROXY_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# --------------------------------------
# AUTHENTICATION
# --------------------------------------
echo "🔐 Authenticating..."
gcloud config set project "$PROJECT_ID"

# --------------------------------------
# CREATE TEMP USER
# --------------------------------------
echo "🛠️ Creating temp user $TMP_DB_USER..."
gcloud sql users create "$TMP_DB_USER" \
  --instance="$INSTANCE_NAME" \
  --password="$TMP_DB_PASS" \
  --host="%" \
  --project="$PROJECT_ID"
sleep 2

# --------------------------------------
# DOWNLOAD CLOUD SQL PROXY
# --------------------------------------
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH_MAP="amd64" ;;
  arm64|aarch64) ARCH_MAP="arm64" ;;
  *) echo "❌ Unsupported arch: $ARCH"; exit 1 ;;
esac

PROXY_URL="https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.8.0/cloud-sql-proxy.${OS}.${ARCH_MAP}"

echo "⬇️ Downloading Cloud SQL Proxy for $OS-$ARCH_MAP..."
curl -sSL -o cloud-sql-proxy "$PROXY_URL"
chmod +x cloud-sql-proxy

echo "🚀 Starting proxy..."
./cloud-sql-proxy "$INSTANCE_CONNECTION_NAME" --address 127.0.0.1 --port "$PROXY_PORT" &
PROXY_PID=$!
sleep 60

# --------------------------------------
# APPLY SCHEMA
# --------------------------------------
echo "📄 Applying schema from $INIT_SQL using $TMP_DB_USER..."
PGPASSWORD=$TMP_DB_PASS psql \
  "host=127.0.0.1 port=$PROXY_PORT dbname=$DB_NAME user=$TMP_DB_USER sslmode=disable" \
  -f "$INIT_SQL"

echo "✅ Schema loaded successfully. Temp user retained for audit or future use."

# --------------------------------------
# GRANT DB PRIVILEGES TO IAM-MAPPED SA
# --------------------------------------

# This removes the literal string ".gserviceaccount.com" from the end.
# WARNING: This is likely incorrect for Cloud SQL IAM authentication.
IAM_DB_ROLE=${IAM_DB_USER//.gserviceaccount.com/}
REGISTRY_ADMIN_DB_ROLE=${REGISTRY_ADMIN_DATABASE_USER//.gserviceaccount.com/}

echo "🛡️ Granting DB privileges to IAM role: \"$IAM_DB_ROLE\""
PGPASSWORD=$TMP_DB_PASS psql \
  "host=127.0.0.1 port=$PROXY_PORT dbname=$DB_NAME user=$TMP_DB_USER sslmode=disable" <<EOF
GRANT USAGE ON SCHEMA public TO "$IAM_DB_ROLE";
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "$IAM_DB_ROLE";
GRANT USAGE ON SCHEMA public TO "$REGISTRY_ADMIN_DB_ROLE";
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "$REGISTRY_ADMIN_DB_ROLE";
EOF