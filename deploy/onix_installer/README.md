# BECKN Onix GCP Installer

## Introduction

The **BECKN Onix GCP Installer** automates the provisioning of infrastructure and application deployment for your BECKN ecosystem. It simplifies setup by managing cloud resources and deploying necessary components.

For a detailed developer guide, please look at [Onix Installer Developer Guide](https://docs.google.com/document/d/1U5gAA4AOUQV5RHYMeM41YW4JYaDEkW87wq_c3DQ-cIU/edit?resourcekey=0-vp7RQSqCutxy8ORAaVjGzA&tab=t.0#heading=h.ggruvyc1rfuz)

---

## Prerequisites

**Important:** This installer is designed to run on **macOS or Linux** environments and requires **Bash version 4.0 or higher**.

Follow these steps to prepare your environment before running the installer.
### Google Cloud Project
You must have google cloud project with 

 - Billing enabled.
 - The following APIs enabled. You can run the `enable_apis.sh` script from the `deploy/onix_installer` directory to enable them automatically:

    ```bash
    chmod +x scripts/enable_apis.sh
    ./scripts/enable_apis.sh
    ```

    The script enables:
    -  Compute Engine API
    -  Kubernetes Engine API
    -  Cloud Resource Manager API
    -  Service Networking API
    -  Secret Manager API
    -  Google Cloud Memorystore for Redis API
    -  Cloud SQL Admin API
    -  Artifact Registry API

### Registered Domain Names
You will need at least one registered domain name (e.g., your-company.com) for which you have administrative access.

Why? These domains are required to expose the Onix services (like the gateway, registry, adapter or subscriber) to the public internet securely.


### Fine-Grained IAM (Identity and Access Management) Permissions (Least Privilege Refinement)

The BECKN Onix Installer implements a least-privilege security model for deployed resources:

-   **GKE (Google Kubernetes Engine) Nodes Service Account**: Uses granular roles like `logging.logWriter`, `monitoring.metricWriter`, `stackdriver.resourceMetadata.writer`, and `artifactregistry.reader`.
    -   **Gateway / Adapter**: `roles/secretmanager.secretAccessor` (read-only access to secrets).
    -   **Subscriber / Registry Admin**: `roles/secretmanager.admin` (required for key generation and management).
    -   **Registry / Registry Admin**: `roles/cloudsql.client` and `roles/cloudsql.instanceUser` for PostgreSQL connections.

---
### Required Tools
   
-   **Google Cloud SDK (`gcloud`)**: For authenticating with Google Cloud and managing resources.
    -   Follow the [official installation guide](https://cloud.google.com/sdk/docs/install).
    -   After installation, run `gcloud init` and `gcloud auth login`.
-   **Terraform**: For provisioning infrastructure as code.
    -   [Installation Guide](https://learn.hashicorp.com/tutorials/terraform/install-cli)
-   **Helm**: For deploying applications onto Kubernetes.
    -   [Installation Guide](https://helm.sh/docs/intro/install/)
-   **kubectl**: A command-line tool for interacting with Kubernetes.
    -   [Installation Guide](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
-   **Docker**: For building and running containers.
    -   [Installation Guide](https://docs.docker.com/engine/install/)
-   **Additional Tools**:
    -   **gsutil**: [Installation Guide](https://cloud.google.com/storage/docs/gsutil_install)
    -   **jq**: [Installation Guide](https://jqlang.github.io/jq/download/)
    -   **gke-gcloud-auth-plugin**: [Installation Guide](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl#install_plugin)
    -   **psql**: [Installation Guide](https://www.postgresql.org/download/)
-   **Python 3.12**:
    For Linux:
```bash
    sudo add-apt-repository ppa:deadsnakes/ppa
    sudo apt update
    sudo apt install python3.12 python3.12-venv
    ```
    For Mac:
     ```bash
    brew install pyenv
    pyenv install 3.12.0
    pyenv global 3.12.0
```

-   **Go**:
    [Installation Guide] https://go.dev/doc/install

-   **Node.js & Angular**:
    ```bash
    # Install Node.js (LTS) via nvm
    curl -o nvm_install.sh https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh
    chmod 700 nvm_install.sh
    ./nvm_install.sh
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    nvm install --lts

    # Install Angular CLI
    npm install -g @angular/cli
    ```

---

## Installation and Setup

### 1. Set Up IAM Permissions for Your User

Ensure the user account running the installer has specific IAM roles on the GCP project. **Note**: These are required even if the user has the "Owner" role.

Run the `set_user_iam.sh` script from the `deploy/onix_installer/scripts` directory to assign the required roles to your user account:

```bash
chmod +x deploy/onix_installer/scripts/set_user_iam.sh
./deploy/onix_installer/scripts/set_user_iam.sh
```

This script will prompt for your GCP Project ID and User Email, then assign the necessary roles:

-   Service Usage Admin
-   Service Account Token Creator
-   Service Account User
-   Service Account Admin
-   Project IAM Admin
-   Secret Manager Admin
-   Cloud SQL Admin
-   Artifact Registry Administrator
-   Kubernetes Engine Admin
-   Storage Admin
-   Workload Identity Pool Admin
-   Vertex AI Administrator


### 2. Create and Configure the Installer Service Account

The installer requires a dedicated service account to manage resources.

**Option A: Automated Setup (Recommended)**

When you run the installer, it will ask for a service account. If it is not created yet, the installer will automatically create it for you and assign all required roles. You do not need to create it separately, you can proceed directly to [Section 3: Configuration](#3-configuration).

**Option B: Manual Setup**

If you prefer creating a service account manually, create a service account in the GCP Console and manually assign it the following roles:

-   Compute Network Admin
-   Compute Load Balancer Admin
-   Compute Instance Admin (v1)
-   Kubernetes Engine Admin
-   Service Account Admin
-   Project IAM Admin
-   Service Account User
-   Storage Admin
-   Cloud SQL Admin
-   Redis Admin
-   Pub/Sub Admin
-   Secret Manager Admin
-   DNS Administrator
-   Compute Security Admin
-   Workload Identity Pool Admin
-   Service Usage Admin
-   Discovery Engine Admin


### 3. Configuration

Set your active GCP project in the `gcloud` CLI:

```bash
gcloud config set project your-project-id
```


### 4. Verify Tool Installation
gcloud --version
terraform --version
helm version
kubectl version --client
docker --version
gsutil version
jq --version
gke-gcloud-auth-plugin --version
python3 --version
go version

### 5. Create Build Artifacts

Before running the installer, you need to build and push the required Docker images to your Artifact Registry.
**Note** Make sure Docker daemon is running in your system

**A. Set Up Google Artifact Registry**

1.  Choose a location (e.g., `asia-south1`) and a name for your repository (e.g., `onix-artifacts`).

2.  Create the Docker repository:
    ```bash
    gcloud artifacts repositories create onix-artifacts \
        --repository-format=docker \
        --location=asia-south1 \
        --description="BECKN Onix Docker repository"
    ```

3.  Configure Docker to authenticate with Artifact Registry:
    ```bash
    gcloud auth configure-docker asia-south1-docker.pkg.dev
    ```

**B. Build and Push the Adapter Image**

1.  Set environment variables for your GCP project, location, and repository name:
    ```bash
    export GCP_PROJECT_ID=$(gcloud config get-value project)
    export AR_LOCATION=asia-south1
    export AR_REPO_NAME=onix-artifacts
    export REGISTRY_URL=${AR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPO_NAME}
    ```

2.  Update the `source.yaml` file with your registry URL. The file is located at `deploy/onix_installer/artifacts/source.yaml`. Uncomment the `registry` field and replace the placeholder `<REGISTRY_URL>` with the value of `$REGISTRY_URL`.

3.  Build and run `onixctl` to build the adapter image and plugins:
    ```bash
    go build ./cmd/onixctl
    ./onixctl --config deploy/onix_installer/artifacts/source.yaml
    ```

**C. Build and Push Core Service Images**

Build and push the Docker images for the gateway, registry, registry-admin, and subscriber services.

> [!IMPORTANT]
> Take note of the image URLs and tags created in this step (e.g., `${REGISTRY_URL}/gateway:${TAG}`). You will need these URLs when configuring the installer UI for app deployment.

```bash
export TAG=latest # Or any tag you prefer

# Gateway
docker buildx build --platform linux/amd64 -t $REGISTRY_URL/gateway:$TAG -f Dockerfile.gateway .
docker push $REGISTRY_URL/gateway:$TAG

# Registry
docker buildx build --platform linux/amd64 -t $REGISTRY_URL/registry:$TAG -f Dockerfile.registry .
docker push $REGISTRY_URL/registry:$TAG

# Registry Admin
docker buildx build --platform linux/amd64 -t $REGISTRY_URL/registry-admin:$TAG -f Dockerfile.registry-admin .
docker push $REGISTRY_URL/registry-admin:$TAG

# Subscriber
docker buildx build --platform linux/amd64 -t $REGISTRY_URL/subscriber:$TAG -f Dockerfile.subscriber .
docker push $REGISTRY_URL/subscriber:$TAG
```

### 5. Run the installer
Once all prerequisites and configurations are complete, change into the installer directory and execute the installer:

```bash
 cd deploy/onix_installer
 make run-installer
```
Note that it will run both backend and frontend portions of the installer application and you can find logs in the logs folder.

### 6. Access installer UI
Go to any browser, and open localhost:4200

### 7. Prepare Adapter Deployment Artifacts

Before proceeding with the application deployment step in the installer, if either a BAP (Buyer Application Provider) or BPP (Seller Application Provider) is being deployed, ensure that the `artifacts` folder contains the necessary artifacts.

-   **`schemas` folder**: This folder is needed if you want to enable schema validation. It should contain all required schema files.
-   **`routing_configs` folder**: This folder must contain the routing configuration files specific to your deployment. For detailed information on configuring routing rules, please refer to the [routing configuration documentation](artifacts/routing_configs/README.md).
    -   `bapTxnReceiver-routing.yaml`: Required if a BAP is being deployed.
    -   `bapTxnCaller-routing.yaml`: Required if a BAP is being deployed.
    -   `bppTxnReceiver-routing.yaml`: Required if a BPP is being deployed.
    -   `bppTxnCaller-routing.yaml`: Required if a BPP is being deployed.
-   **`plugins` folder**: This folder should contain the plugin bundle, which is a `.zip` file.

***Important Notes***

- Please add your routing config .yaml files in artifacts/routing_configs folder before infra deployment step.
- If you are using pubsub please add Adapter topic name from infra deployment output and add in .yaml files in artifacts/routing_configs folder.

### 8. Invoker Service Accounts and Impersonation

The installer creates dedicated service accounts for invoker authentication. These accounts are permitted to call specific Onix services when Inbound Authentication is enabled.

#### Created Invoker Accounts

The following invoker service accounts are created per application deployment:

-   `subscriber-invoker-sa-${appName}@${project_id}.iam.gserviceaccount.com` to call Subscriber service APIs
-   `admin-invoker-sa-${appName}@${project_id}.iam.gserviceaccount.com` to call Admin service APIs
-   `adapter-invoker-sa-${appName}@${project_id}.iam.gserviceaccount.com` to call Adapter service's Caller APIs

*Note: Replace `${appName}` with the application name used during
deployment and `${project_id}` with your GCP Project ID.*

#### Using Service Account Impersonation

To call the Adapter or Subscriber services using these invoker accounts without needing their private keys, you can use **Service Account
Impersonation**.

##### Prerequisites

Ensure your user account (or calling service) has the **Service Account Token Creator** role (`roles/iam.serviceAccountTokenCreator`) on the target invoker service account.

#### Target Audience Values

When generating an ID token (via `gcloud` or programmatic libraries), you must use the correct audience for the target service:

| Service | Target Audience | Used For |
| :--- | :--- | :--- |
| **Registry Admin Service** | `https://<registry-admin-domain>/api` | Client calls to approve/reject subscriptions |
| **Subscriber Service** | `https://<subscriber-domain>/api` | Client calls to manage subscriptions |
| **Adapter Service** | `https://<adapter-domain>/caller/api` | BAP/BPP client calls to invoke Adapter |

*Note: Replace `<...-domain>` with the respective FQDN configured during installation.*

##### Generating a Token via `gcloud`

Run the following command to generate a short-lived OIDC ID token for the invoker service account:

```bash
gcloud auth print-identity-token \
    --impersonate-service-account=adapter-invoker-sa-${appName}@${project_id}.iam.gserviceaccount.com \
    --audiences="https://your-adapter-url.com"
```

Use the output of this command in the `Authorization` header of your HTTP requests:

`Authorization: Bearer <OUTPUT_TOKEN>`

> [!IMPORTANT]
> When using service account impersonation to generate an ID token for ONIX, both the **target audience** and **including the service account email (e.g., `include_email=True` in Python or equivalent options in Go/Java)** are **always mandatory** regardless of the language used to ensure the token contains the necessary identity claims for authorization.

For programmatic usage (Go, Python, Java), please refer to the **[Main README Security Section](../../README.md#security-and-authentication)**.


---

# App Deployment Inputs Config Parameters

This document describes the configuration parameters required for
 deploying the ONIX application components via the installer.

## General Configuration

-   **App Name**: A unique name for the application deployment.
-   **Components**: A dictionary indicating which components to deploy (e.g., `gateway`, `registry`, `bap`, `bpp`).

## Domain Names Configuration

Provide the FQDN (Fully Qualified Domain Name) for each service:

-   `registry`: URL for the registry service.
-   `registry_admin`: URL for the registry admin portal.
-   `subscriber`: URL for the subscriber service.
-   `gateway`: URL for the gateway service.
-   `adapter`: URL for the adapter service.

## Image URLs

Provide the Docker image URLs for each service (from Artifact Registry):

-   `registry`
-   `registry_admin`
-   `subscriber`
-   `gateway`
-   `adapter`

## Registry Configuration

-   **Subscriber ID**: Unique ID for the Registry within the new or existing Open Network.
-   **Key ID**: Unique ID for the Signature validation and decryption key for the Registry within the new or existing Open Network.
-   **Enable Auto Approver**: Checkbox to enable automatic approval of incoming subscription requests.

## Gateway Configuration

-   **Subscriber ID**: Unique ID for the Gateway within the new or existing Open Network.

## Adapter Configuration

-   **Enable Schema Validation**: Checkbox to enable validation of incoming request schemas.

## Security Configuration

### Inbound Authentication

-   **Enable Inbound Auth**: Checkbox to enable inbound authentication.
-   **Issuer URL**: URL of the OIDC provider (e.g., `https://accounts.google.com`).
-   **ID Claim**: The claim to check (e.g., `email`).
-   **Allowed Values**: List of allowed values for the ID claim (e.g., emails of allowed callers).
-   **JWKS(JSON Web Key Set) Content**: (Optional) Static JWKS content if dynamic fetching is not used.

### Outbound Authentication

-   **Enable Outbound Auth**: Checkbox to enable outbound authentication.
-   **Audience Overrides**: (Optional) Override audience for specific targets.

## Cleanup and Destruction
To uninstall Onix and delete all associated infrastructure, run the following command from the deploy/onix_installer directory.

 ⚠️ Warning: This command is irreversible. It will delete all infrastructure created by the installer (such as GKE clusters, databases, and Redis instances) and will also delete all Onix services that you created through the installer

```bash 
 make destroy-infra
```
   