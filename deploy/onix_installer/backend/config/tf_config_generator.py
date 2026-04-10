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
from typing import Any

from core.constants import INSTALLER_KIT_PATH, TEMPLATE_DIRECTORY, TERRAFORM_DIRECTORY
from core.models import InfraDeploymentRequest
from core.utils import render_jinja_template, write_file_content

logger = logging.getLogger(__name__)

# Filenames
MAIN_CONFIG_TEMPLATE_NAME = "main_tfvars.tfvars.j2"
OUTPUT_TFVARS_FILENAME = "generated-terraform.tfvars"


def validate_immutable_triplet(
    state_data: dict[str, Any], deploy_req: InfraDeploymentRequest
) -> None:
  """Validates immutable state identifiers in deployment requests.

  Throws a ValueError if a user tries to deploy to a different project, region,
  or app_name
  than the one already recorded in installer_state.json.

  Args:
      state_data: A dictionary containing the current installer state, typically
        loaded from installer_state.json.
      deploy_req: An InfraDeploymentRequest object representing the new
        deployment request.
  """
  checks = [
      ("project_id", deploy_req.project_id),
      ("region", deploy_req.region),
      ("app_name", deploy_req.app_name),
  ]

  for key, new_val in checks:
    existing_val = state_data.get(key)
    if existing_val and new_val != existing_val:
      raise ValueError(
          f"Cannot change {key} from '{existing_val}' to '{new_val}' "
          "on an existing deployment."
      )


def generate_config(deploy_infra_req: InfraDeploymentRequest) -> None:
  """Orchestrates the configuration generation process for Terraform.

  Processes a main template which contains all size-specific logic, and writes
  the rendered content to the final terraform.tfvars file.

  Args:
      deploy_infra_req: An InfraDeploymentRequest object containing the
        deployment parameters.
  """
  logger.info("Starting Terraform Configuration File Generation.")

  template_source_dir = os.path.join(TEMPLATE_DIRECTORY, "tf_configs")

  # Define the output directory and file for the generated tfvars.
  output_tfvars_path = os.path.join(TERRAFORM_DIRECTORY, OUTPUT_TFVARS_FILENAME)

  # 1. Read existing state if present
  state_file_path = os.path.join(INSTALLER_KIT_PATH, "installer_state.json")
  state_data = {}
  if os.path.exists(state_file_path):
    try:
      with open(state_file_path, "r") as f:
        state_data = json.load(f)
        logger.info("Loaded core intent from existing installer_state.json")

        validate_immutable_triplet(state_data, deploy_infra_req)

    except Exception as e:
      logger.error("Error reading %s: %s", state_file_path, e)
      raise

  # 3. Prepare updated Jinja2 context from user overrides.
  jinja_context = {
      "project_id": deploy_infra_req.project_id,
      "region": deploy_infra_req.region,
      "app_name": deploy_infra_req.app_name,
      "deployment_size": deploy_infra_req.type.value.lower(),
      "provision_adapter_infra": (
          deploy_infra_req.components.get("bap", False)
          or deploy_infra_req.components.get("bpp", False)
      ),
      "provision_gateway_infra": deploy_infra_req.components.get(
          "gateway", False
      ),
      "provision_registry_infra": deploy_infra_req.components.get(
          "registry", False
      ),
      "enable_cloud_armor": deploy_infra_req.enable_cloud_armor,
      "rate_limit_count": deploy_infra_req.rate_limit_count,
      "enable_onix": any(
          deploy_infra_req.components.get(k, False)
          for k in ["registry", "gateway", "bap", "bpp"]
      ),
      "enable_agent": state_data.get("enable_agent", False),
      "provision_agent_db": state_data.get("enable_agent", False),
      "agent_engine_id": state_data.get("agent_engine_id", ""),
      "datastore_imports": state_data.get("datastore_imports", {}),
  }
  logger.debug("Jinja2 context for Terraform: %s", jinja_context)

  # 4. Write intent back to purely local installer_state.json
  try:
    with open(state_file_path, "w") as f:
      json.dump(jinja_context, f, indent=4)
      logger.info("Persisted deployment intent to %s", state_file_path)
  except Exception as e:
    logger.error("Error writing to %s: %s", state_file_path, e)
    raise

  # Process the main terraform configuration template.
  logger.info(
      "Processing main configuration template: '%s'...",
      MAIN_CONFIG_TEMPLATE_NAME,
  )
  try:
    rendered_content = render_jinja_template(
        template_dir=template_source_dir,
        template_name=MAIN_CONFIG_TEMPLATE_NAME,
        context=jinja_context,
    )
    logger.info("Main configuration template processed successfully.")
  except Exception as e:
    logger.error(
        "Failed to process main Terraform configuration template: %s", e
    )
    raise

  # Write the rendered content to the single output file.
  logger.info("Writing generated terraform.tfvars to: '%s'", output_tfvars_path)
  try:
    write_file_content(output_tfvars_path, rendered_content)
    logger.info(
        "Successfully generated single '%s' file.", OUTPUT_TFVARS_FILENAME
    )
  except IOError as e:
    logger.error(
        "Error writing tfvars file '%s': %s. Aborting"
        " configuration generation.",
        output_tfvars_path,
        e,
    )
    raise

  logger.info("Terraform Configuration Generation Complete.")
