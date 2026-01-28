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
import os
from typing import List
import core.utils as utils
from core.models import InfraDeploymentRequest, AppDeploymentRequest
from core.constants import INFRA_SCRIPT_PATH, APP_SCRIPT_PATH, TERRAFORM_DIRECTORY, ANSI_ESCAPE_PATTERN
import config.tf_config_generator as tf_config
import config.app_config_generator as app_config

logger = logging.getLogger(__name__)

async def run_infra_deployment(config: InfraDeploymentRequest, websocket):
    """
    Generates Terraform configurations and executes the infrastructure deployment script.
    Mimics the logic from the provided main.py.
    """
    logger.info(f"Initiating infrastructure deployment for project: {config.project_id}, region: {config.region}")

    await websocket.send_text(json.dumps({"type": "info", "message": "Generating Terraform configurations..."}))
    try:
        # Calling generate_config() to populate/process the terraform.tfvars file.
        tf_config.generate_config(config)
        await websocket.send_text(json.dumps({"type": "info", "message": "Terraform configurations generated successfully."}))
        logger.info("Terraform configurations generated successfully.")
    except Exception as e:
        error_message = f"Failed to generate Terraform configurations: {e}"
        logger.error(error_message, exc_info=True)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))
        return # Exit if config generation fails.

    await websocket.send_text(json.dumps({"type": "info", "message": "Executing infrastructure deployment script..."}))
    logger.info(f"Executing infrastructure deployment script: {INFRA_SCRIPT_PATH}")

    # Prepare the command to run the infrastructure deployment script.
    command = [
        '/bin/bash',
        INFRA_SCRIPT_PATH,
    ]

    try:
        #  Running the script as an asynchronous subprocess.
        process = await asyncio.create_subprocess_exec(
            *command,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
            cwd=TERRAFORM_DIRECTORY
        )

        await utils.stream_subprocess_output(process, websocket, "infra_deploy_log")

        return_code = await process.wait()
        logger.info(f"Infrastructure deployment script finished with exit code: {return_code}")

        if return_code == 0:
            outputs_json_path = os.path.join(TERRAFORM_DIRECTORY, "outputs.json")
            try:
                with open(outputs_json_path, "r") as f:
                    outputs_data = json.load(f)
                
                logger.info(f"Successfully loaded outputs.json: {outputs_data}")
                await websocket.send_text(json.dumps({
                    "type": "success",
                    "message": outputs_data
                }))
                logger.info("Successfully sent outputs.json content to client.")
            except FileNotFoundError:
                error_msg = f"Error: outputs.json not found at {outputs_json_path}"
                logger.error(error_msg)
                await websocket.send_text(json.dumps({"type": "error", "message": error_msg}))
            except json.JSONDecodeError:
                error_msg = f"Error: Could not decode outputs.json at {outputs_json_path}"
                logger.error(error_msg)
                await websocket.send_text(json.dumps({"type": "error", "message": error_msg}))
        else:
            await websocket.send_text(json.dumps({
                "type": "error",
                "message": f"Script failed with exit code: {return_code}"
            }))

    except FileNotFoundError:
        error_message = f"Infrastructure deployment script not found at: {INFRA_SCRIPT_PATH}"
        logger.error(error_message)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))
    except Exception as e:
        error_message = f"An error occurred during infrastructure deployment: {str(e)}"
        logger.exception(error_message)
        await websocket.send_text(json.dumps({"type": "error", "message": error_message}))

def _get_services_to_deploy(app_deployment_request: AppDeploymentRequest) -> List[str]:
    """
    Calculates and returns a sorted list of services to be deployed based on the
    AppDeploymentRequest components.
    """
    services_to_deploy = set()

    if app_deployment_request.components.get("bap", False) or app_deployment_request.components.get("bpp", False):
        services_to_deploy.add("adapter")
    if app_deployment_request.components.get("gateway", False):
        services_to_deploy.add("gateway")
    if app_deployment_request.components.get("registry", False):
        services_to_deploy.add("registry")
        services_to_deploy.add("registry_admin")
    if app_config.should_deploy_subscriber(app_deployment_request.components):
        services_to_deploy.add("subscriber")
    sorted_services = sorted(list(services_to_deploy))
    logger.debug(f"Determined services to deploy: {sorted_services}")
    return sorted_services

async def run_app_deployment(app_deployment_request: AppDeploymentRequest, websocket):
    """
    Generates application configurations and executes the application deployment script.
    Mimics the logic from the provided main.py.
    """
    logger.info(f"Initiating application deployment with payload: {app_deployment_request}")

    await websocket.send_text(json.dumps({"type": "info", "message": "Generating application configurations..."}))
    try:
        app_config.generate_app_configs(app_deployment_request)
        logger.info("Application configuration YAMLs generated successfully.")
        await websocket.send_text(json.dumps({"type": "info", "message": "Application configurations generated successfully."}))

        services = _get_services_to_deploy(app_deployment_request)
    
        env_vars_for_script = app_config.get_deployment_environment_variables(app_deployment_request, services)
        logger.info("Environment variables for app script prepared.")
        
        # Combine generated environment variables with current process environment.
        subprocess_env = os.environ.copy()
        subprocess_env.update(env_vars_for_script)

        # Prepare and Run the deploy-app.sh shell script
        shell_command = [
            '/bin/bash',
            APP_SCRIPT_PATH
        ]

        await websocket.send_text(json.dumps({"type": "info", "message": "Executing application deployment script..."}))
        logger.info(f"Executing application deployment script: {APP_SCRIPT_PATH}")

        subprocess_handle = await asyncio.create_subprocess_exec(
            *shell_command,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
            cwd=TERRAFORM_DIRECTORY,
            env=subprocess_env
        )

        await utils.stream_subprocess_output(subprocess_handle, websocket, "app_deploy_log")

        return_code = await subprocess_handle.wait()
        logger.info(f"Application deployment script finished with exit code: {return_code}")

        # Handle Script Result.
        if return_code == 0:
            log_exploer_urls = app_config.generate_logs_explorer_urls(services)

            service_urls = app_config.extract_final_urls(app_deployment_request.domain_names, services)
            logger.info(json.dumps({
                "type": "success",
                "action": "app_deploy_complete",
                "message": "Application deployment successful!",
                "data": {
                    "service_urls": service_urls,
                    "services_deployed": services,
                    "logs_explorer_urls": log_exploer_urls,
                }
            }))
            await websocket.send_text(json.dumps({
                "type": "success",
                "action": "app_deploy_complete",
                "message": "Application deployment successful!",
                "data": {
                    "service_urls": service_urls,
                    "services_deployed": services,
                    "logs_explorer_urls": log_exploer_urls,
                }
            }))
        else:
            await websocket.send_text(json.dumps({
                "type": "error",
                "action": "app_deploy_failed",
                "message": f"Application deployment script failed with exit code: {return_code}. Check logs above for details."
            }))

    except FileNotFoundError as e:
        error_message = f"Error generating application configs: Required outputs.json not found or script not found: {str(e)}. Ensure infrastructure is deployed."
        logger.error(error_message)
        await websocket.send_text(json.dumps({"type": "error", "action": "app_config_error", "message": error_message}))
    except Exception as e:
        error_message = f"An unexpected error occurred during app deployment: {str(e)}"
        logger.exception(error_message)
        await websocket.send_text(json.dumps({"type": "error", "action": "app_deploy_exception", "message": error_message}))
