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

import asyncio
import json
import logging
from typing import Dict
import httpx

logger = logging.getLogger(__name__)

class HealthCheckConstants:
    HEALTH_CHECK_URL_SUFFIX: str = "/health"
    HEALTH_CHECK_TIMEOUT_SECONDS: int = 15 * 60 # 15 minutes total timeout
    HEALTH_CHECK_ATTEMPT_DELAY_SECONDS: int = 10 # Seconds between polls


async def perform_single_health_check(
    service_name: str,
    base_url: str,
    websocket,
    client: httpx.AsyncClient
) -> bool:
    """
    Performs a single health check against a given service URL and sends the result via WebSocket.
    Returns True if the check passes, False otherwise.
    """
    health_url = f"https://{base_url.rstrip('/')}{HealthCheckConstants.HEALTH_CHECK_URL_SUFFIX}"
    try:
        response = await client.get(health_url)
        if response.status_code == 200:
            message = f"Health check for {service_name} at {base_url} (endpoint: {health_url}) PASSED (HTTP 200)."
            log_level = "success"
            logger.info(message)
        else:
            message = f"Health check for {service_name} at {base_url} (endpoint: {health_url}) FAILED: HTTP {response.status_code}."
            log_level = "warning"
            logger.warning(message)

        await websocket.send_text(json.dumps({
            "type": log_level,
            "service": service_name,
            "url": base_url,
            "health_url": health_url,
            "status_code": response.status_code,
            "message": message
        }))
        return response.status_code == 200

    except httpx.RequestError as e:
        message = f"Health check for {service_name} at {base_url} (endpoint: {health_url}) FAILED: Request Error - {e}."
        logger.warning(message)
        await websocket.send_text(json.dumps({
            "type": "warning",
            "service": service_name,
            "url": base_url,
            "health_url": health_url,
            "status_code": None,
            "message": message
        }))
        return False
    except Exception as e:
        message = f"An unexpected error occurred during health check for {service_name}: {e}."
        logger.exception(message)
        await websocket.send_text(json.dumps({
            "type": "error",
            "service": service_name,
            "url": base_url,
            "health_url": health_url,
            "status_code": None,
            "message": message
        }))
        return False


async def run_websocket_health_check(websocket, service_urls_to_check: Dict[str, str]):
    """
    Manages the health check polling process for multiple services over a WebSocket.
    Sends updates and final status to the connected client.
    """
    if not isinstance(service_urls_to_check, dict) or \
       not all(isinstance(k, str) and isinstance(v, str) for k, v in service_urls_to_check.items()):
        logger.error("Invalid payload format received for healthCheck in service.")
        await websocket.send_text(json.dumps({
            "type": "error",
            "message": "Invalid payload format. Expected a dictionary of serviceName: serviceUrl strings."
        }))
        return

    await websocket.send_text(json.dumps({
        "type": "info",
        "message": "Starting health checks for provided service URLs..."
    }))
    logger.info(f"Starting comprehensive health checks for: {', '.join(service_urls_to_check.keys())}")

    if not service_urls_to_check:
        await websocket.send_text(json.dumps({
            "type": "warning",
            "message": "No service URLs provided for health check. Health check skipped."
        }))
        logger.warning("No service URLs provided for health check.")
        return

    total_timeout_seconds = HealthCheckConstants.HEALTH_CHECK_TIMEOUT_SECONDS
    delay_between_attempts = HealthCheckConstants.HEALTH_CHECK_ATTEMPT_DELAY_SECONDS

    start_time = asyncio.get_event_loop().time()
    pending_services = {name: url for name, url in service_urls_to_check.items()}

    try:
        async with httpx.AsyncClient(timeout=30) as client:
            attempt = 0
            while pending_services:
                attempt += 1
                current_time = asyncio.get_event_loop().time()
                elapsed_time = current_time - start_time

                if elapsed_time > total_timeout_seconds:
                    await websocket.send_text(json.dumps({
                        "type": "error",
                        "action": "health_check_timeout",
                        "message": f"Health checks timed out after {int(elapsed_time)} seconds ({total_timeout_seconds / 60} minutes). The following service URLs are still unhealthy: {', '.join(pending_services.keys())}"
                    }))
                    logger.error(f"Health checks timed out. Unhealthy services: {', '.join(pending_services.keys())}")
                    break

                await websocket.send_text(json.dumps({
                    "type": "log",
                    "message": f"Attempt {attempt}: Checking {len(pending_services)} service URL(s) for health... ({int(elapsed_time)}s elapsed / {total_timeout_seconds}s timeout)"
                }))
                logger.info(f"Attempt {attempt}: Checking {len(pending_services)} services. Elapsed: {int(elapsed_time)}s.")

                tasks = [
                    perform_single_health_check(service_name, service_url, websocket, client)
                    for service_name, service_url in pending_services.items()
                ]

                results = await asyncio.gather(*tasks)

                services_that_passed_this_round = [
                    service_name
                    for i, (service_name, _) in enumerate(pending_services.items())
                    if results[i] # If the check passed (True)
                ]

                for service_name in services_that_passed_this_round:
                    del pending_services[service_name]

                if not pending_services:
                    await websocket.send_text(json.dumps({
                        "type": "success",
                        "action": "all_services_healthy",
                        "message": "All deployed service URLs are now healthy and reachable!"
                    }))
                    logger.info("All services are healthy.")
                    break

                if pending_services:
                    await websocket.send_text(json.dumps({
                        "type": "log",
                        "message": f"{len(pending_services)} service URL(s) are not yet healthy. Retrying in {delay_between_attempts} seconds..."
                    }))
                    logger.warning(f"{len(pending_services)} services still pending health check. Retrying...")
                    await asyncio.sleep(delay_between_attempts)

    except Exception as e:
        logger.exception(f"An unexpected error occurred within run_websocket_health_check: {e}")
        raise