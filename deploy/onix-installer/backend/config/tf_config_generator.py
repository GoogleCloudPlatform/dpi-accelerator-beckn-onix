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

import logging
import os

from core.models import InfraDeploymentRequest
from core.utils import render_jinja_template, write_file_content
from core.constants import TERRAFORM_DIRECTORY, TEMPLATE_DIRECTORY

logger = logging.getLogger(__name__)

# Filenames
MAIN_CONFIG_TEMPLATE_NAME = "main_tfvars.tfvars.j2"
OUTPUT_TFVARS_FILENAME = "generated-terraform.tfvars"

def generate_config(deploy_infra_req: InfraDeploymentRequest):
    """
    Orchestrates the configuration generation process for Terraform.
    Processes a main template which contains all size-specific logic,
    and writes the rendered content to the final terraform.tfvars file.
    """
    logger.info("Starting Terraform Configuration File Generation.")

    template_source_dir = os.path.join(TEMPLATE_DIRECTORY, "tf_configs")

    # Define the output directory and file for the generated tfvars.
    output_tfvars_path = os.path.join(TERRAFORM_DIRECTORY, OUTPUT_TFVARS_FILENAME)

    # Prepare Jinja2 context from user overrides.
    # The deployment_type is included so conditional logic can be used within the template.
    jinja_context = {
        "project_id": deploy_infra_req.project_id,
        "region": deploy_infra_req.region,
        "suffix": deploy_infra_req.app_name,
        "deployment_size": deploy_infra_req.type.value.lower(),
        "provision_adapter_infra": deploy_infra_req.components.get('bap', False) or deploy_infra_req.components.get('bpp', False),
        "provision_gateway_infra": deploy_infra_req.components.get('gateway', False),
        "provision_registry_infra": deploy_infra_req.components.get('registry', False),
    }
    logger.debug(f"Jinja2 context for Terraform: {jinja_context}")

    # Process the main terraform configuration template.
    logger.info(f"Processing main configuration template: '{MAIN_CONFIG_TEMPLATE_NAME}'...")
    try:
        rendered_content = render_jinja_template(
            template_dir=template_source_dir,
            template_name=MAIN_CONFIG_TEMPLATE_NAME,
            context=jinja_context
        )
        logger.info("Main configuration template processed successfully.")
    except Exception as e:
        logger.error(f"Failed to process main Terraform configuration template: {e}")
        raise

    # Write the rendered content to the single output file.
    logger.info(f"Writing generated terraform.tfvars to: '{output_tfvars_path}'")
    try:
        write_file_content(output_tfvars_path, rendered_content)
        logger.info(f"Successfully generated single '{OUTPUT_TFVARS_FILENAME}' file.")
    except IOError as e:
        logger.error(f"Error writing tfvars file '{output_tfvars_path}': {e}. Aborting configuration generation.")
        raise

    logger.info("Terraform Configuration Generation Complete.")