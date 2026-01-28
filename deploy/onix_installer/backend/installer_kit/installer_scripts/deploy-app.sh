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

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "\nError: $1 is not installed. Please install it before proceeding.\n"
        exit 1
    fi
}

echo -e "\nChecking system requirements...\n"
check_command "gcloud"
check_command "helm"
check_command "kubectl"
check_command "gsutil"
check_command "jq"  # required for parsing JSON outputs to read outputs.json

echo -e "\nAll required tools are installed. Proceeding...\n"

OUTPUTS_FILE="outputs.json" # Relative to CWD
CONFIG_DIR="../../../generated_configs" # Relative to CWD
SCHEMA_DIR="../../../adapter_artifacts/schemas" # Relative to CWD
PLUGIN_DIR="../../../adapter_artifacts/plugins" # Relative to CWD
ROUTING_CONFIG_DIR="../../../adapter_artifacts/routing_configs" # Relative to CWD
# HTTPS_TFVARS="https.tfvars" # Relative to CWD

# Checking if outputs.json exists
if [ ! -f "$OUTPUTS_FILE" ]; then
    echo -e "\nError: outputs.json not found in $(pwd)/$OUTPUTS_FILE. Please run deploy-infra.sh first.\n"
    exit 1
fi

echo -e "\nReading infrastructure outputs from $OUTPUTS_FILE...\n"

NAMESPACE=$(jq -r '.app_namespace_name.value' "$OUTPUTS_FILE")
CLUSTER_NAME=$(jq -r '.cluster_name.value' "$OUTPUTS_FILE")
CLUSTER_REGION=$(jq -r '.cluster_region.value' "$OUTPUTS_FILE")
PROJECT_NAME=$(jq -r '.project_id.value' "$OUTPUTS_FILE")
CONFIG_BUCKET=$(jq -r '.config_bucket_name.value' "$OUTPUTS_FILE")

# Service Account Names
ADAPTER_KSA_NAME=$(jq -r '.adapter_ksa_name.value' "$OUTPUTS_FILE")
GATEWAY_KSA_NAME=$(jq -r '.gateway_ksa_name.value' "$OUTPUTS_FILE")
REGISTRY_KSA_NAME=$(jq -r '.registry_ksa_name.value' "$OUTPUTS_FILE")
REGISTRY_ADMIN_KSA_NAME=$(jq -r '.registry_admin_ksa_name.value' "$OUTPUTS_FILE")
SUBSCRIBER_KSA_NAME=$(jq -r '.subscription_ksa_name.value' "$OUTPUTS_FILE")

# For debugging: print extracted values
echo "NAMESPACE: $NAMESPACE"
echo "CLUSTER_NAME: $CLUSTER_NAME"
echo "CLUSTER_REGION: $CLUSTER_REGION"
echo "PROJECT_NAME: $PROJECT_NAME"
echo "CONFIG_BUCKET: $CONFIG_BUCKET"
echo "ADAPTER_KSA_NAME: $ADAPTER_KSA_NAME"
echo "GATEWAY_KSA_NAME: $GATEWAY_KSA_NAME"
echo "REGISTRY_KSA_NAME: $REGISTRY_KSA_NAME"
echo "REGISTRY_ADMIN_KSA_NAME: $REGISTRY_ADMIN_KSA_NAME"
echo "SUBSCRIPTION_KSA_NAME: $SUBSCRIBER_KSA_NAME"


# Validate essential infrastructure variables
if [ -z "$NAMESPACE" ]; then echo -e "\nError: NAMESPACE missing from outputs.json. Exiting.\n"; exit 1; fi
if [ -z "$CONFIG_BUCKET" ]; then echo -e "\nError: CONFIG_BUCKET missing from outputs.json. Exiting.\n"; exit 1; fi
if [ -z "$CLUSTER_NAME" ]; then echo -e "\nError: CLUSTER_NAME missing from outputs.json. Exiting.\n"; exit 1; fi
if [ -z "$CLUSTER_REGION" ]; then echo -e "\nError: CLUSTER_REGION missing from outputs.json. Exiting.\n"; exit 1; fi
if [ -z "$PROJECT_NAME" ]; then echo -e "\nError: PROJECT_NAME missing from outputs.json. Exiting.\n"; exit 1; fi

# Validate required environment variables from main.py
if [ -z "$DEPLOY_SERVICES" ]; then
    echo -e "\nError: DEPLOY_SERVICES environment variable not set. Exiting.\n"
    exit 1
fi

# Update https.tfvars with domains provided by main.py
# Collect all domains that are set from environment variables
DOMAINS_ARRAY=()
[ -n "$REGISTRY_DOMAIN" ] && DOMAINS_ARRAY+=("\"$REGISTRY_DOMAIN\"")
[ -n "$REGISTRY_ADMIN_DOMAIN" ] && DOMAINS_ARRAY+=("\"$REGISTRY_ADMIN_DOMAIN\"")
[ -n "$SUBSCRIBER_DOMAIN" ] && DOMAINS_ARRAY+=("\"$SUBSCRIBER_DOMAIN\"")
[ -n "$GATEWAY_DOMAIN" ] && DOMAINS_ARRAY+=("\"$GATEWAY_DOMAIN\"")
[ -n "$ADAPTER_DOMAIN" ] && DOMAINS_ARRAY+=("\"$ADAPTER_DOMAIN\"")

DOMAINS_LIST=$(IFS=,; echo "${DOMAINS_ARRAY[*]}")


#########################################
# Part 1: Upload Configs to GCS Bucket #
#########################################

echo -e "\nUploading configs and schemas from $CONFIG_DIR to bucket: gs://$CONFIG_BUCKET/ ...\n"
gsutil -m cp -r "$CONFIG_DIR"/* "gs://$CONFIG_BUCKET/configs/"

if [ $? -eq 0 ]; then
    echo -e "\nConfig Files upload successful!\n"
else
    echo -e "\nConfig Files upload failed. Please check the logs and retry.\n"
    exit 1
fi

gsutil -m cp -r "$ROUTING_CONFIG_DIR"/* "gs://$CONFIG_BUCKET/configs/"

if [ $? -eq 0 ]; then
    echo -e "\nRouting Config Files upload successful!\n"
else
    echo -e "\nRouting Config Files upload failed. Please check the logs and retry.\n"
    exit 1
fi

gsutil -m cp -r "$PLUGIN_DIR" "gs://$CONFIG_BUCKET/"

if [ $? -eq 0 ]; then
    echo -e "\nPlugin bundle upload successful!\n"
else
    echo -e "\nPlugin bundle upload failed. Please check the logs and retry.\n"
    exit 1
fi

# Add schema validation check
if [ "$ENABLE_SCHEMA_VALIDATION" == "true" ]; then
    if [ ! -d "$SCHEMA_DIR" ]; then
        echo -e "\nError: Schema validation is enabled, but the schemas directory '$SCHEMA_DIR' was not found. Please ensure the directory exists and contains the necessary schema files.\n"
        exit 1
    fi
    echo -e "\nSchema validation is enabled. Uploading schemas from $SCHEMA_DIR to bucket: gs://$CONFIG_BUCKET/configs/schemas/ ...\n"
    gsutil -m cp -r "$SCHEMA_DIR" "gs://$CONFIG_BUCKET/configs/"
    if [ $? -eq 0 ]; then
        echo -e "\nSchemas upload successful!\n"
    else
        echo -e "\nSchemas upload failed. Please check the logs and retry.\n"
        exit 1
    fi
else
    echo -e "\nSchema validation is disabled. Skipping schema upload.\n"
fi

#########################################
# Part 2: Deploy Application via Helm   #
#########################################

echo -e "\nConnecting to the cluster...\n"

gcloud container clusters get-credentials "$CLUSTER_NAME" --region "$CLUSTER_REGION" --project "$PROJECT_NAME"

get_chart_path_for_service() {
    case "$1" in
        registry)
            echo "./helm_charts/onix_registry"
            ;;
        registry-admin)
            echo "./helm_charts/onix_registry_admin"
            ;;
        subscriber)
            echo "./helm_charts/onix_subscriber"
            ;;
        gateway)
            echo "./helm_charts/onix_gateway"
            ;;
        adapter)
            echo "./helm_charts/onix_adapter"
            ;;
        *)
            echo "" # Return empty string for unknown service
            ;;
    esac
}

# Services to iterate based on DEPLOY_SERVICES env var
IFS=',' read -r -a SERVICES_TO_DEPLOY <<< "$DEPLOY_SERVICES"

echo -e "\nDeploying services: ${SERVICES_TO_DEPLOY[*]}\n"
ATTEMPTS=5  # Max attempts (adjust if needed)
SLEEP_TIME=20 # Seconds between checks

# List to collect successfully deployed hosts for final output
SERVICE_HOSTS=()

for SERVICE in "${SERVICES_TO_DEPLOY[@]}"; do
    SERVICE_BASE_NAME=$(echo "$SERVICE" | tr 'a-z' 'A-Z')

    SERVICE_KSA="${SERVICE_BASE_NAME}_KSA_NAME"
    KSA="${!SERVICE_KSA}"

    SERVICE_DOMAIN="${SERVICE_BASE_NAME}_DOMAIN"
    DOMAIN="${!SERVICE_DOMAIN}"

    SERVICE_IMAGE_URL="${SERVICE_BASE_NAME}_IMAGE_URL"
    IMAGE_URL="${!SERVICE_IMAGE_URL}"

    SERVICE=${SERVICE//_/-}

    CHART_DIR=$(get_chart_path_for_service "$SERVICE")
    if [ -z "$CHART_DIR" ]; then
        echo -e "\nError: Unknown service '$SERVICE'. No chart path defined. Exiting.\n"
        exit 1
    fi
    echo deploying service: "$SERVICE"
    

    # Validate that essential env vars are indeed set for current service
    if [ -z "$DOMAIN" ]; then
        echo -e "\nError: Domain for service '$SERVICE' not provided (missing ${SERVICE_BASE_NAME}_DOMAIN env var). Skipping deployment.\n"
        exit 1
    fi
    if [ -z "$IMAGE_URL" ]; then
        echo -e "\nError: Image URL for service '$SERVICE' not provided (missing ${SERVICE_BASE_NAME}_IMAGE_URL env var). Skipping deployment.\n"
        exit 1
    fi

    echo -e "\n=============================="
    echo -e "Deploying $SERVICE service"
    echo -e "Image: $IMAGE_URL"
    echo -e "Domain: $DOMAIN"
    echo -e "Service Account: $KSA"
    echo -e "==============================\n"

    HELM_COMMON_ARGS=(
        "--set" "config.namespace=$NAMESPACE"
        "--set" "config.ksa=$KSA"
        "--set" "config.bucketName=$CONFIG_BUCKET"
        "--set" "image.repository=$IMAGE_URL"
    )

    HELM_COMMAND=("helm" "upgrade" "--install" "onix-$SERVICE" "$CHART_DIR")

    if [[ "$SERVICE" == "adapter" ]]; then
        ADAPTER_HOST="$DOMAIN"
        cat > /tmp/adapter-ingress.yaml <<EOF
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: $ADAPTER_HOST
      paths:
        - path: /bap/receiver/on_subscribe
          pathType: Prefix
          backend:
            service:
              name: onix-subscriber-service
              port: 80
        - path: /bpp/receiver/on_subscribe
          pathType: Prefix
          backend:
            service:
              name: onix-subscriber-service
              port: 80
        - path: /
          pathType: Prefix
          backend:
            service:
              name: onix-adapter-service
              port: 80
EOF
        HELM_COMMAND+=("-f" "/tmp/adapter-ingress.yaml")
        SERVICE_HOSTS+=("$ADAPTER_HOST")

    elif [[ "$SERVICE" == "gateway" ]]; then
        GATEWAY_HOST="$DOMAIN"
        cat > /tmp/gateway-ingress.yaml <<EOF
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: $GATEWAY_HOST
      paths:
        - path: /on_subscribe
          pathType: Prefix
          backend:
            service:
              name: onix-subscriber-service
              port: 80
        - path: /
          pathType: Prefix
          backend:
            service:
              name: onix-gateway-service
              port: 80
EOF
        HELM_COMMAND+=("-f" "/tmp/gateway-ingress.yaml")
        SERVICE_HOSTS+=("$GATEWAY_HOST")
    else
        INGRESS_HOST="$DOMAIN"
        HELM_COMMAND+=("--set" "ingress.host=$INGRESS_HOST")
    fi

    echo -e "\nDeploying $SERVICE using Helm...\n"
    "${HELM_COMMAND[@]}" "${HELM_COMMON_ARGS[@]}"

    if [ $? -ne 0 ]; then
        echo -e "\nHelm deployment for $SERVICE failed. Gathering diagnostics...\n"
        POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=onix-$SERVICE" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        if [[ -n "$POD_NAME" ]]; then
            echo -e "\n--- kubectl describe pod $POD_NAME ---\n"
            kubectl describe pod "$POD_NAME" -n "$NAMESPACE"
            echo -e "\n--- kubectl logs $POD_NAME ---\n"
            kubectl logs "$POD_NAME" -n "$NAMESPACE" --all-containers=true
        else
            echo -e "No pod found for $SERVICE to describe or fetch logs."
        fi
        echo -e "\nRolling back (uninstalling onix-$SERVICE)...\n"
        helm uninstall "onix-$SERVICE" || true
        echo -e "Helm release onix-$SERVICE uninstalled (if it existed)."
        echo -e "Please check logs and retry.\n"
        exit 1
    fi

    kubectl rollout restart deployment -n "$NAMESPACE" -l "app.kubernetes.io/name=onix-$SERVICE"
    sleep 20 # Wait for the restart

    SUCCESS=0
    echo -e "\nWaiting for the $SERVICE deployment to become ready...\n"

    LABEL="app.kubernetes.io/name=onix-$SERVICE"

    for ((j=1; j<=ATTEMPTS; j++)); do
        POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        POD_STATUS=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL" -o jsonpath='{.items[0].status.phase}' 2>/dev/null)

        if [[ -z "$POD_NAME" || -z "$POD_STATUS" ]]; then
            echo -e "Attempt $j/$ATTEMPTS: No pods found for $SERVICE yet. Retrying in $SLEEP_TIME seconds..."
        elif [[ "$POD_STATUS" == "Running" ]]; then
            echo -e "$SERVICE pod is Running. Verifying stability..."
            STABLE=1
            for k in {1..4}; do # Check for 20 seconds (4 * 5s)
                CONTAINER_READY=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].ready}' 2>/dev/null || echo "false")
                RESTARTS=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo 0)
                
                if [[ "$CONTAINER_READY" != "true" || "$RESTARTS" -gt 0 ]]; then
                    echo -e "$SERVICE pod not yet stable (ready: $CONTAINER_READY, restarts: $RESTARTS). Checking again..."
                    STABLE=0
                    sleep 5
                else
                    STABLE=1
                    break # Container is ready and no restarts, break inner loop
                fi
            done

            if [[ "$STABLE" == "1" ]]; then
                SUCCESS=1
                break # Pod is stable, break outer loop
            else
                echo -e "$SERVICE pod failed stability check after $RESTARTS restarts. Exiting."
                kubectl describe pod "$POD_NAME" -n "$NAMESPACE"
                exit 1
            fi
        elif [[ "$POD_STATUS" == "Succeeded" ]]; then
            SUCCESS=1
            break
        elif [[ "$POD_STATUS" == "Failed" || "$POD_STATUS" == "CrashLoopBackOff" || "$POD_STATUS" == "ImagePullBackOff" ]]; then
            echo -e "\nDeployment failed for $SERVICE. Error details:\n"
            kubectl describe pods -n "$NAMESPACE" -l "$LABEL"
            exit 1
        else
            echo -e "Attempt $j/$ATTEMPTS: $SERVICE pod status: $POD_STATUS. Retrying in $SLEEP_TIME seconds..."
        fi
        sleep $SLEEP_TIME
    done

    if [[ "$SUCCESS" == "1" ]]; then
        echo -e "\n$SERVICE deployment successful and stable!\n"
        kubectl get pods -n "$NAMESPACE" -l "$LABEL"
        kubectl get svc -n "$NAMESPACE"
    else
        echo -e "\nDeployment for $SERVICE did not become ready after $ATTEMPTS attempts.\n"
        kubectl get pods -n "$NAMESPACE" -l "$LABEL"
        exit 1
    fi
done

echo -e "\n====================================================="
echo -e "All services deployed! Now enabling HTTPS Load Balancer and SSL certificate with Terraform...\n"

cd phase2
terraform init
terraform apply --var-file p2.tfvars -auto-approve
# terraform plan --var-file p2.tfvars

if [ $? -eq 0 ]; then
    echo -e "\nHTTPS Load Balancer and SSL certificate deployed successfully!\n"
else
    echo -e "\nTerraform apply for HTTPS resources failed. Please check the logs.\n"
    exit 1
fi

echo -e "Your services are available at the following hosts:\n"
for host in "${SERVICE_HOSTS[@]}"; do
    echo -e "  - \033[1;32mhttps://$host\033[0m"
done
echo -e "=====================================================\n"