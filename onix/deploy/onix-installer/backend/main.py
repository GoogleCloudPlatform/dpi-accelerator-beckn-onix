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

"""
Main FastAPI application for the Onix Installer backend.

This module sets up the FastAPI server, middleware, and defines all
API endpoints, including REST endpoints for GCP resource management
and WebSocket endpoints for long-running deployment processes.
"""

import json
import logging
import sys
from typing import Dict, Any
import httpx

from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware

from .core.models import InfraDeploymentRequest, AppDeploymentRequest, ProxyRequest
from .services.gcp_resource_manager import list_google_cloud_projects, list_google_cloud_regions
from .services.deployment_manager import run_infra_deployment, run_app_deployment
from .services import ui_state_manager as ui_state
from .services.health_checks import run_websocket_health_check

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
    """
    Root endpoint to check if the server is running.
    """
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
        logger.info("Successfully retrieved %s Google Cloud projects.", len(projects))
        return projects
    except HTTPException as e:
        logger.error("Failed to list GCP projects: %s", e.detail)
        raise e
    except Exception as e:
        logger.exception("An unexpected error occurred while listing GCP projects.")
        raise HTTPException(
            status_code=500,
            detail=f"An unexpected internal server error occurred: {e}"
        ) from e

@app.get("/regions", response_model=list[str])
async def get_regions():
    """
    Fetches and returns a list of all Google Cloud Regions.
    """
    logger.info("Attempting to list Google Cloud regions.")
    try:
        regions = await list_google_cloud_regions()
        logger.info("Successfully retrieved %s Google Cloud regions.", len(regions))
        return regions
    except HTTPException as e:
        logger.error("Failed to list GCP regions: %s", e.detail)
        raise e
    except Exception as e:
        logger.exception("An unexpected error occurred while listing GCP regions.")
        raise HTTPException(
            status_code=500,
            detail=f"An unexpected internal server error occurred: {e}"
        ) from e

@app.websocket("/ws/deployInfra")
async def websocket_deploy_infra(websocket: WebSocket):
    """
    WebSocket endpoint to handle infrastructure (Terraform) deployments.
    Receives deployment configuration, streams logs, and sends final status.
    """
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/deployInfra.")
    try:
        data = await websocket.receive_json()
        config = InfraDeploymentRequest(**data)
        logger.info("Received infrastructure deployment request with payload: %s.", config)

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
    """
    WebSocket endpoint to handle application (K8s/Helm) deployments.
    Receives deployment configuration, streams logs, and sends final status.
    """
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/deployApp.")

    try:
        app_request_payload = await websocket.receive_json()
        app_deployment_request = AppDeploymentRequest(**app_request_payload)
        logger.info("Received application deployment request with payload: %s", app_deployment_request)

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
    """
    WebSocket endpoint to run health checks on deployed services.
    Receives a dictionary of service URLs and streams back their status.
    """
    await websocket.accept()
    logger.info("WebSocket connection established for /ws/healthCheck.")
    try:
        service_urls_to_check: Dict[str, str] = await websocket.receive_json()
        logger.info("Received health check request for services: %s.", list(service_urls_to_check.keys()))
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
def store_or_update_values(items: Dict[str, Any]) -> Dict[str, Any]:
    """
    Accepts a dictionary of key-value pairs for bulk storage/update in ui_state.json.
    Args:
        items (Dict[str, Any]): A dictionary where keys are strings and values can be of any type.
    Returns:
        Dict[str, Any]: A confirmation message along with the data that was processed.
    """
    logger.info("Received request for bulk store/update of data. Keys: %s", list(items.keys()))
    try:
        ui_state.store_bulk_values(items)
        logger.info("Successfully stored/updated bulk data for keys: %s", list(items.keys()))
        return {
            "message": "Data stored or updated successfully",
            "processed_data": items
        }
    except Exception as e:
        logger.error("Failed to perform bulk store/update for keys %s: %s", list(items.keys()), e)
        raise HTTPException(status_code=500, 
        detail=f"Failed to store bulk data: {e}") from e


@app.get("/store")
def get_all_stored_data() -> Dict[str, Any]:
    """
    Retrieves all key-value pairs from ui_state.json.
    Returns:
        Dict[str, Any]: A dictionary containing all the stored data.
    """
    logger.info("Received request to retrieve all stored data.")
    try:
        data_store = ui_state.load_all_data()
        logger.info("Successfully retrieved all stored data.")
        return data_store
    except Exception as e:
        logger.error("Failed to retrieve all stored data: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to retrieve data: {e}") from e


@app.post("/api/dynamic-proxy")
async def dynamic_proxy(request: ProxyRequest) -> Any:
    """
    Forwards a POST request to a specified target URL with a given payload.
    Args:
        request (ProxyRequest): A request object containing the target URL and the payload.
    Returns:
        Any: The Text response from the target URL.
    """
    target_url = request.targetUrl
    payload = request.payload

    logger.info("Forwarding request to: %s", target_url)

    async with httpx.AsyncClient() as client:
        try:
            response = await client.post(target_url, json=payload, timeout=10.0)
            response.raise_for_status()
            # todo : send response.status_code as well
            return response.content
        except httpx.HTTPStatusError as exc:
            logger.error("HTTP error occurred: %s - %s", exc.response.status_code, exc.response.text)
            raise HTTPException(
                status_code=exc.response.status_code,
                detail=exc.response.text
            )
        except Exception as e:
            logger.error("An unexpected error occurred: %s", e)
            raise HTTPException(
                status_code=500,
                detail="An internal server error occurred."
            ) from e