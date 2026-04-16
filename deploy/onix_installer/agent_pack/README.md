# DPI Blueprint Agent App: Deployment Guide

This document provides a comprehensive, step-by-step walkthrough for setting up,
deploying, and configuring the DPI Reference Agent infrastructure natively on
Google Cloud.

The Agent Pack is an automated installation system that bundles your application
logic, provisions downstream Google Cloud assets (VPC, Network Attachments,
Redis, Discovery Engine Data Stores, IAM Service Accounts), and activates them
on Vertex AI Reasoning Engines.

--------------------------------------------------------------------------------

## Prerequisites and Initial Setup

Before running the deployment, ensure your local environment has the following
CLI tools installed:

-   `gcloud` (Google Cloud CLI)
-   `terraform`
-   `python3` and `pip`
-   `jq` (JSON processor)

All commands assume you are operating from the root of the `dpi_becknonix`
directory (cloned on your cloudtop or local machine).

### Step 1: Assign User IAM Permissions

You must assign yourself the necessary IAM roles to provision infrastructure.
Navigate to the installer directory and run:

```shell
cd deploy/onix_installer
bash scripts/set_user_iam.sh
```

### Step 2: Enable Resource Manager APIs

The Google Cloud Resource Manager API must be enabled to allow Terraform to
configure project-level IAM policies.

```shell
gcloud services enable cloudresourcemanager.googleapis.com --project <PROJECT_ID>
gcloud services enable serviceusage.googleapis.com --project <PROJECT_ID>
```

### Step 3: Clone the Application Blueprint

The installer pulls the application logic from a separate module. You must clone
the `dpi_agent_blueprint` repository **inside** the `agent_pack` directory for
it to be bundled:

```shell
# Navigate to the agent pack directory
cd agent_pack

# Clone the blueprint
git clone https://github.com/GoogleCloudPlatform/dpi_agent_blueprint dpi_agent_blueprint
```

--------------------------------------------------------------------------------

### Configuring Your Environment

The deployment is driven by the environment configuration file located at:
`deploy/onix_installer/agent_pack/agent_config.env`

You must fill in the specific values below to provision standard infrastructure.

### 1. Project Config and Identifiers

| Variable | Explanation | Value Constraints / Examples |
| :--- | :--- | :--- |
| `GOOGLE_CLOUD_PROJECT` | Your target Google Cloud Project ID. | Expects your alphanumeric unique project id (e.g. `my-dpi-project`). |
| `REGION` | The GCP Region to deploy Serverless containers in. | e.g. `asia-south1` or `us-central1`. |
| `APP_NAME` | A unique prefix used to namespace your buckets and SAs. | Use your LDAP or unique slug to prevent resource collision. |
| `STAGING_BUCKET` | Cloud Storage bucket for staging zip assets. | (Recommended) Keep commented to let the script derive it. |
| `DEMOGRAPHIC_REGION` | The region constraints for location context. | Defaults to `INDIA`. |

### 2. Use Cases

| Variable | Explanation | Default / Example |
| :--- | :--- | :--- |
| `SKILLS_OF_ROOT_AGENT` | Comma-separated list of enabled use-cases/skills for standard text mode. | `loan_discovery,pest_detection,mandi_prices,agri_input,biochar_advice,govt_benefits,market_linkages,weather_forecast` |
| `SKILLS_OF_ROOT_AGENT_LIVE` | Comma-separated list of enabled skills for live voice mode. | `loan_discovery,pest_detection,mandi_prices,agri_input,biochar_advice,govt_benefits,market_linkages,weather_forecast` |

### 3. Config Flags

| Variable | Explanation | Default / Permitted |
| :--- | :--- | :--- |
| `SESSION_DB_TYPE` | Where multi-turn session history is stored. | `agent-engine` (internal Vertex service) or `database` (Postgres). |
| `ENABLE_LONG_TERM_MEMORY` | Toggles persistence of user context over time across sessions. | `TRUE` or `FALSE`. |
| `ENABLE_ROOT_AGENT_CACHING` | Toggles context caching for the root agent to reduce latency and cost. | `TRUE` or `FALSE`. |
| `ROOT_AGENT_CACHE_TTL` | Time-to-live for the cache in seconds. | `300` (5 minutes). |

### 4. Agent Application Model

Override base models and variances for routing vs domain agents:

| Variable | Explanation | Default |
| :--- | :--- | :--- |
| `ROOT_AGENT_LLM_MODEL` | The main routing node LLM (Standard Text session). | `gemini-3.1-flash-lite-preview` |
| `ROOT_AGENT_LIVE_LLM_MODEL` | Real-time voice routing LLM. | `gemini-live-2.5-flash` |
| `ROOT_AGENT_LLM_TEMPERATURE` | Generation randomness. | `0.1` (deterministic/factual). |
| `MAX_LLM_CALLS_PER_QUERY` | Protect loops from running forever over multiple tools. | `20` |
| `AGENT_THINKING_LEVEL` | Controls internal reasoning depth for Gemini 3 and newer models (Permitted: `MINIMAL`, `LOW`, `MEDIUM`, `HIGH`). See [documentation](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/start/get-started-with-gemini-3#thinking-level). | `LOW` |
| `AGENT_INCLUDE_THOUGHTS` | Whether to include the thought process in the output response. | `FALSE` |

*Note: Similar override configurations exist for Sub-Agents and Search Grounding
models (e.g., `SUB_AGENTS_LLM_MODEL`).*

### 5. Logging & Observability

| Variable | Explanation | Default |
| :--- | :--- | :--- |
| `LOG_LEVEL` | Verbosity of python logs. | `INFO` |
| `GOOGLE_CLOUD_AGENT_ENGINE_ENABLE_TELEMETRY` | Pushes traces to trace explorer. | `TRUE` |
| `ENABLE_ADK_LOGGING_PLUGIN` | Toggles verbose framework trace logs inside container. | `FALSE` |

### 6. Beckn Open-Network Configurations

These apply specifically to Sub-Agents performing broad network searches over
Beckn network.

| Variable | Explanation | Default |
| :--- | :--- | :--- |
| `MAX_SEARCH_RESULTS` | Number of async on_search callbacks to wait for before parsing returns. | `5` |
| `SEARCH_TIMEOUT_SECONDS` | Max wait time for on_search callbacks in seconds. | `3` |
| `CONFIRM_TIMEOUT_SECONDS` | Max wait time for on_confirm callbacks in seconds. | `3` |
| `STATUS_TIMEOUT_SECONDS` | Max wait time for on_status callbacks in seconds. | `3` |
| `SHOULD_ADD_OAUTH_BEARER_TOKEN` | Enables standard service impersonation and auth adding to hit BAP adapter. | `FALSE` |
| `GOOGLE_ADAPTER_INVOKER_SERVICE_ACCOUNT` | Onix Adapter Invoker service account | Check you Onix Adapter deployment outputs. |
| `GOOGLE_ADAPTER_AUDIENCE` | Audience string used by ONIX Adapter for verifying token. | generally `<adapter-fqdn>/caller/api` |
| `BECKN_ADAPTER_DOMAINS_CONFIG` | (Explained Below) Struct mapping sub-agents to their BAP adapter config | [See Structure](#7-understanding-beckn_adapter_domains_config) |

### 7. Understanding `BECKN_ADAPTER_DOMAINS_CONFIG`

This environment JSON defines per-expertise routing parameters. It maps standard
sub-agents to their live transactional gateways.

```json
{
  "agri.market_linkages": {
    "gateway_url": "https://your-adapter-url.com/bap/caller/",
    "bap_id": "gemini-agent.bap.uag.gov.in",
    "bap_uri": "https://your-adapter-url.com/bap/receiver/",
    "domain": "agri-procurement"
  }
}
```

*   **`gateway_url`**: The BAP Adapter's caller endpoint where the Agent Engine
    sends beckn network request payloads where it is transformed into a
    Beckn-compliant request and forwarded to beckn network.
*   **`bap_id`**: Your unique network subscriber ID.
*   **`bap_uri`**: The BAP Adapter's receiver endpoint where network callbacks
    come back into the system and which routes them back to agent engine.
*   **`domain`**: The standard beckn context taxonomy identifier (e.g.
    `agri-procurement` or `agri-credit`).
*   **`bpp_id`** / **`bpp_uri`** (Optional): Direct B2B routing if you don't
    wish to use standard gateway broadcast.

### 8. External Tool Call Overrides (Mandi Price Tools)

Overriding direct API hooks used by standard tools:

| Variable | Explanation | Default |
| :--- | :--- | :--- |
| `MANDI_API_URL` | Mandi Price fetching REST endpoint. | `https://farmservicesapi.agribazaar.com/search` |
| `MANDI_API_KEY` | API Key verified at Mandi API endpoint. | `your-key` |

### 9. Compute and Scaling (Hard Container Ceilings)

Hard limits for the serverless container deployment. For more details on supported values and recommendations, refer to the [official Vertex AI documentation](https://cloud.google.com/vertex-ai/generative-ai/docs/agent-engine/deploy#customized-resource-controls).

-   **`MIN_INSTANCES`**: Default `1`
-   **`MAX_INSTANCES`**: Default `10`
-   **`CONTAINER_CONCURRENCY`**: Default `4`
-   **`CPU_LIMIT`**: Default `4`
-   **`MEMORY_LIMIT`**: Default `16Gi`

### 10. Knowledge Base Ingestion (DATASTORE_IMPORTS)

Maps sub-agent names to GCS URIs for Document RAG ingestion into Vertex AI Discovery Engine. For details on supported file types and how to prepare your data, see the [official Vertex AI Search documentation](https://docs.cloud.google.com/generative-ai-app-builder/docs/prepare-data).

```bash
DATASTORE_IMPORTS='{"agri.biochar_advice": "gs://your-bucket-name/agri_biochar/*"}'
```

*Note: The installer automatically provisions Discovery Engine Data Stores for
the agents defined here, ingests the documents from the bucket, and binds them
to the agent.*

--------------------------------------------------------------------------------

## Deploying the Agent

Once `agent_config.env` is configured, navigate to the `onix_installer` folder
to initiate the deployment using `make`:

```shell
cd .. # Navigate back to onix_installer
make setup-agent
```

### What to expect

1.  **Validation & Authentication**: The script checks your binaries and prompts
    you to log in to Google Cloud.
2.  **Service Account Initialization**: You will be asked if you want to use a
    pre-provisioned service account or have the script build one for you
    automatically. *(Recommended)*: Let the script create it on the first run,
    note its email, and use that email for all subsequent or update runs. This
    SA is used to run the installer and deploy/provision everything on GCP.
3.  **Infrastructure Provisioning**: Terraform parses your environment variables
    and provisions Service API Enablements, VPC network and subnets, Redis
    Instance, Proxy VM (for public outbound calls from Agent Engine), Cloud SQL
    (if database chosen), Discovery Engine datastores, and VPC network
    attachments (PSC).
4.  **Knowledge Ingestion**: GCS documents are push-imported into newly created
    Discovery Engine datastores for agents like biochar_advice.
5.  **Agent Activation**: The Vertex AI SDK bundles the `dpi_agent_blueprint`
    Python code and deploys it to the Vertex AI Agent Engine.

--------------------------------------------------------------------------------

## Verification and Lifecycle Management

### Post-Deployment Verification

When the script finishes, verify the resources in the Google Cloud Console:

1.  **Reasoning Engines**: Navigate to *Vertex AI > Agent Builder > Reasoning
    Engines* to view your active deployment.
2.  **Discovery Engine**: Navigate to *Vertex AI Search* to verify your RAG
    datastores.

### Understanding Success Outputs

At the end of a successful run, the script dumps a block of infrastructure
endpoints.

-   **Agent engine ID**: The unique URI of your deployed engine.
-   **Agent URL**: The target HTTP endpoint for invocations.
-   **Agent invoker SA**: The IAM Service Account email used for authentication
    by external callers. (It already has `aiplatform.user` permission to invoke
    the agent engine).

### Interacting with the Agent Engine via curl (Safe Impersonation)

It is recommended to impersonate the **Agent Invoker SA** for calling our
deployed agent engine. Note that any identity you use must have the
`roles/aiplatform.user` role on the resource. Refer to the documentation inside
standard `dpi_agent_blueprint` repository for detailed API specifications and
schema models.

```shell
# Call the Agent securely using the impersonated service account
curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token --impersonate-service-account=${AGENT_INVOKER_SA_EMAIL})" \
     -H "Content-Type: application/json" \
     "${AGENT_ENGINE_URL}" \
     -d '{
       "class_method": "chat",
       "input": {
         "message": {
           "user_id": "demo-user",
           "query": [{"text": "Hello"}]
         }
       }
     }'
```

### Applying Hotfixes and Code Updates

The deployment is highly idempotent. The flow depends on what you are updating:

#### A. Updating Configuration (Environment Variables)

1.  Modify your `agent_config.env`.
2.  Re-run `make setup-agent`.

#### B. Updating Agent Application Code

1.  Replace or update the code inside the `agent_pack/dpi_agent_blueprint`
    directory (e.g., clone the updated repo again).
2.  Update `agent_config.env` and re-run `make setup-agent`. The installer will
    bundle the new code and redeploy it to Vertex AI.

--------------------------------------------------------------------------------

## State Protection & File Hygiene

The installer uses a common `installer_state.json` tracking system to
orchestrate both ONIX and Agent deployments. Because they share the same backend
Terraform state, there are strict rules regarding file safety:

> [!CAUTION] **Locked Identity Triplets**: Once deployed (either ONIX or Agent),
> you **cannot** change the core triplets (`GOOGLE_CLOUD_PROJECT`, `REGION`,
> `APP_NAME`). Changing these parameters can corrupt the installation state.

> [!IMPORTANT] **Do Not Manually Mutate Auto-generated Files**: Do not update,
> delete, or modify files inside the installer directory unless explicitly
> requested to by this guide. This includes:
>
> -   Auto-generated config files (consolidated `.env` files)
> -   State files (`installer_state.json`)
> -   Terraform variable files (`.tfvars`)
>
> Manual tampering with these files will corrupt the deployment state.
