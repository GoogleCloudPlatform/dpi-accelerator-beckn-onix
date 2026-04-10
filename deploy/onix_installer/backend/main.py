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

import json
import logging
import os
import sys
from typing import Any
import httpx

import google.auth
from google.auth import impersonated_credentials
from google.auth.transport import requests as google_auth_requests


from fastapi.applications import FastAPI
from fastapi.exceptions import HTTPException
from fastapi.websockets import WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware

from core.models import InfraDeploymentRequest, AppDeploymentRequest, ProxyRequest, ConfigGenerationRequest, ConfigUpdateRequest
from services.gcp_resource_manager import list_google_cloud_projects, list_google_cloud_regions
from services.deployment_manager import run_infra_deployment, run_app_deployment
from services import ui_state_manager as ui_state
import services.config_manager as config_manager
from services.health_checks import run_websocket_health_check

logging.basicConfig(level=logging.DEBUG,
                    format='%(name)s - %(levelname)s - %(message)s',
                    handlers=[
                        logging.StreamHandler(sys.stdout)
                    ])
logger = logging.getLogger(__name__)

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/root")
def read_root():
    logger.info("Root endpoint accessed.")
    return {"message": "FastAPI deployment server is running"}

@app.get("/projects", response_model=list[str])
async def get_projects():
    """
    Fetches and returns a list of all Google Cloud Project IDs accessible
    by the authenticated user or service account.

    """
    logger.info("Attempting to list Google Cloud projects.")
    try:
        projects = await list_google_cloud_projects()
        logger.info(f"Successfully retrieved {len(projects)} Google Cloud projects.")
        return projects
    except HTTPException as e:
        logger.error(f"Failed to list GCP projects: {e.detail}")
        raise e
    except Exception as e:
        logger.exception("An unexpected error occurred while listing GCP projects.")
        raise HTTPException(
            status_code=500,
            detail=f"An unexpected internal server error occurred: {e}"
        )

@app.get("/regions", response_model=list[str])
async def get_regions():
    """
    Fetches and returns a list of all Google Cloud Regions.

    """
    logger.info("Attempting to list Google Cloud regions.")
    try:
        regions = await list_google_cloud_regions()
        logger.info(f"Successfully retrieved {len(regions)} Google Cloud regions.")
        return regions
    except HTTPException as e:
        logger.error(f"Failed to list GCP regions: {e.detail}")
        raise e
    except Exception as e:
        logger.exception("An unexpected error occurred while listing GCP regions.")
        raise HTTPException(
            status_code=500,
            detail=f"An unexpected internal server error occurred: {e}"
        )

@app.post("/api/configs")
def generate_configs(request: ConfigGenerationRequest):
    """Generates initial configuration files based on the provided request.

    Does not overwrite existing files if they are already present.
    Args:
        request: A ConfigGenerationRequest object containing component flags and settings.
    Returns:
        A dictionary containing a success message.
    Raises:
        HTTPException: If an internal error occurs during generation (500).
    """
    logger.info("Received request to generate initial configurations.")
    try:
        config_manager.generate_initial_configs(request)
        return {"message": "Initial configurations generated successfully."}
    except Exception as e:
        logger.error(f"Failed to generate configs: {e}")
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/api/configs/path")
def get_config_paths() -> dict[str, Any]:
    """Retrieves a list of all existing configuration file paths.

    Returns:
        A dictionary containing the count of files and a list of file paths.
    Raises:
        HTTPException: If an internal error occurs while listing paths (500).
    """
    logger.info("Received request to list config paths.")
    try:
        paths = config_manager.get_all_config_paths()
        return {"count": len(paths), "files": paths}
    except Exception as e:
        logger.error(f"Failed to list config paths: {e}")
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/api/config/data")
def get_config_data(path: str) -> dict[str, str]:
    """Retrieves the raw text content of a specific configuration file.

    Args:
        path: The relative file path to the configuration file (e.g., 'generated_configs/adapter.yaml').
    Returns:
        A dictionary containing the content string of the file.
    Raises:
        HTTPException: If the path is invalid (403), the file is not found (404), or a server error occurs (500).
    """
    logger.info(f"Received request to read config: {path}")
    try:
        return {"content": config_manager.get_config_content(path)}
    except ValueError as e:
        logger.warning(f"Invalid path access attempt: {e}")
        raise HTTPException(status_code=403, detail="Invalid path")
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail="File not found")
    except Exception as e:
        logger.error(f"Failed to read config: {e}")
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.put("/api/config/data")
def update_config_data(request: ConfigUpdateRequest):
    """Updates the content of a specific configuration file.

    Args:
        request: A ConfigUpdateRequest object containing the file path and the new content.
    Returns:
        A dictionary containing a success message.
    Raises:
        HTTPException: If the path is invalid/forbidden (403) or a server error occurs (500).
    """
    logger.info(f"Received request to update config: {request.path}")
    try:
        config_manager.update_config_content(request.path, request.content)
        return {"message": "Configuration updated successfully."}
    except ValueError as e:
        logger.warning(f"Invalid path update attempt: {e}")
        raise HTTPException(status_code=403, detail="Invalid path")
    except Exception as e:
        logger.error(f"Failed to update config: {e}")
        raise HTTPException(status_code=500, detail=str(e)) from e

@app.websocket("/ws/deployInfra")
async def websocket_deploy_infra(websocket: WebSocket):
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/deployInfra.")
    try:
        data = await websocket.receive_json()
        config = InfraDeploymentRequest(**data)
        logger.info(f"Received infrastructure deployment request with payload: {config}.")

        await run_infra_deployment(config, websocket)

    except WebSocketDisconnect:
        logger.info("Client disconnected from /ws/deployInfra.")
    except json.JSONDecodeError:
        logger.error("Received invalid JSON payload from client for /ws/deployInfra.")
        await websocket.send_text(json.dumps({"type": "error", "message": "Invalid JSON payload."}))
    except Exception as e:
        error_message = f"An error occurred during infrastructure deployment: {str(e)}"
        logger.exception(error_message)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))
    finally:
        # Check if websocket is still connected (State 1 means CONNECTED)
        if websocket.client_state == 1:
            await websocket.close()
        logger.info("WebSocket connection for /ws/deployInfra closed.")


@app.websocket("/ws/deployApp")
async def websocket_deploy_application(websocket: WebSocket):
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/deployApp.")

    try:
        app_request_payload = await websocket.receive_json()
        app_deployment_request = AppDeploymentRequest(**app_request_payload)
        logger.info(f"Received application deployment request with payload: {app_deployment_request}")

        await run_app_deployment(app_deployment_request, websocket)

    except WebSocketDisconnect:
        logger.info("Client disconnected from /ws/deployApp.")
    except json.JSONDecodeError:
        logger.error("Received invalid JSON payload from client for /ws/deployApp.")
        await websocket.send_text(json.dumps({"type": "error", "message": "Invalid JSON payload."}))
    except Exception as e:
        error_message = f"An unexpected error occurred during application deployment: {str(e)}"
        logger.exception(error_message)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))
    finally:
        # Check if websocket is still connected (State 1 means CONNECTED)
        if websocket.client_state == 1:
            await websocket.close()
        logger.info("WebSocket connection for /ws/deployApp closed.")


@app.websocket("/ws/healthCheck")
async def websocket_health_check(websocket: WebSocket):
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/healthCheck.")
    try:
        service_urls_to_check: dict[str, str] = await websocket.receive_json()
        logger.info(f"Received health check request for services: {list(service_urls_to_check.keys())}.")
        await run_websocket_health_check(websocket, service_urls_to_check)
    except WebSocketDisconnect:
        logger.info("Client disconnected from /ws/healthCheck.")
    except json.JSONDecodeError:
        logger.error("Received invalid JSON from client for healthCheck. Expected a dictionary of serviceName: serviceUrl strings.")
        await websocket.send_text(json.dumps({
            "type": "error",
            "message": "Invalid JSON received. Expected a dictionary of serviceName: serviceUrl strings."
        }))
    except Exception as e:
        error_message = f"An unexpected error occurred during health check: {str(e)}"
        logger.exception(error_message)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))
    finally:
        # Check if websocket is still connected (State 1 means CONNECTED)
        if websocket.client_state == 1:
            await websocket.close()
        logger.info("WebSocket connection for /ws/healthCheck closed.")



@app.post("/store/bulk", status_code=201)
def store_or_update_values(items: dict[str, Any]) -> dict[str, Any]:
    """
    Accepts a dictionary of key-value pairs for bulk storage/update in ui_state.json.

    Args:
        items (dict[str, Any]): A dictionary where keys are strings and values can be of any type.
    Returns:
        dict[str, Any]: A confirmation message along with the data that was processed.
    """
    logger.info(f"Received request for bulk store/update of data. Keys: {list(items.keys())}")
    try:
        ui_state.store_bulk_values(items)
        logger.info(f"Successfully stored/updated bulk data for keys: {list(items.keys())}")
        return {
            "message": "Data stored or updated successfully",
            "processed_data": items
        }
    except Exception as e:
        logger.error(f"Failed to perform bulk store/update for keys {list(items.keys())}: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to store bulk data: {e}")


@app.get("/store")
def get_all_stored_data() -> dict[str, Any]:
    """
    Retrieves all key-value pairs from ui_state.json.

    Returns:
        dict[str, Any]: A dictionary containing all the stored data.
    """
    logger.info("Received request to retrieve all stored data.")
    try:
        data_store = ui_state.load_all_data()
        logger.info("Successfully retrieved all stored data.")
        return data_store
    except Exception as e:
        logger.error(f"Failed to retrieve all stored data: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to retrieve data: {e}")


@app.get("/api/installer-state")
def get_installer_state() -> dict[str, Any]:
  """Retrieves the authoritative deployment intent state.

  Returns:
    A dictionary containing the deployment intent state if
    `installer_state.json` exists and is valid.
    Returns an empty dictionary if the file does not exist,
    indicating no deployment has been attempted yet.
  """
  logger.info("Received request to retrieve installer state.")
  state_file_path = os.path.join(
      os.path.dirname(__file__),
      "installer_kit",
      "installer_state.json",
  )
  try:
    if not os.path.exists(state_file_path):
      return {}
    with open(state_file_path, "r") as f:
      return json.load(f)
  except Exception as e:
    logger.error("Failed to retrieve installer state: %s", e)
    raise HTTPException(
        status_code=500, detail=f"Failed to parse installer_state.json: {e}"
    ) from e


@app.post("/api/dynamic-proxy")
async def dynamic_proxy(request: ProxyRequest) -> Any:
    """
    Forwards a POST request to a specified target URL with a given payload.

    Args:
        request (ProxyRequest): A request object containing the target URL and the payload.
    Returns:
        Any: The Text response from the target URL.
    """
    target_url = request.target_url
    payload = request.payload

    headers = {}
    if request.impersonate_service_account and request.audience:
        try:
            logger.info(f"Attempting to impersonate service account: {request.impersonate_service_account} for audience: {request.audience}")
            source_credentials, _ = google.auth.default()
            target_credentials = impersonated_credentials.Credentials(
                source_credentials,
                target_principal=request.impersonate_service_account,
                target_scopes=["https://www.googleapis.com/auth/cloud-platform"],
            )
            impersonated_creds = impersonated_credentials.IDTokenCredentials(
                target_credentials,
                target_audience=request.audience,
                include_email=True,
            )
            request_adapter = google_auth_requests.Request()
            impersonated_creds.refresh(request_adapter)
            headers["Authorization"] = f"Bearer {impersonated_creds.token}"
            logger.info("Successfully generated impersonated OIDC token.")
        except Exception as e:
            logger.error(f"Failed to impersonate service account: {e}")
            raise HTTPException(
                status_code=500,
                detail=f"Failed to authenticate proxy request: {e}"
            )

    logger.info(f"Forwarding request to: {target_url}")

    async with httpx.AsyncClient() as client:
        try:
            response = await client.post(target_url, json=payload, headers=headers, timeout=10.0)
            response.raise_for_status()
            # todo : send response.status_code as well
            return response.content
        except httpx.HTTPStatusError as exc:
            logger.error(f"HTTP error occurred: {exc.response.status_code} - {exc.response.text}")
            raise HTTPException(
                status_code=exc.response.status_code,
                detail=exc.response.text
            )
        except Exception as e:
            logger.error(f"An unexpected error occurred: {e}")
            raise HTTPException(
                status_code=500,
                detail="An internal server error occurred."
            )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

