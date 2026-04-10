#!/usr/bin/env python3
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

from __future__ import annotations

import json
import os
import sys
from typing import Any

import dotenv
from jinja2 import Environment, FileSystemLoader


def print_conflict_error(
    field_name: str, new_val: Any, existing_val: Any
) -> None:
  """Prints a conflict error message and exits the script.

  Args:
    field_name: The name of the configuration field with a conflict.
    new_val: The new value attempted to be set.
    existing_val: The existing value in the deployment state.
  """
  print(f"❌ Error: Conflict in '{field_name}'.")
  print(f"   The file 'agent_config.env' specifies {field_name.upper()}='{new_val}',")
  print(f"   but this directory has an existing deployment for {field_name.upper()}='{existing_val}'.")
  print(f"   -> To modify the existing infrastructure, please use '{existing_val}' in your env file.")
  print(f"   -> To create a brand new environment, please run the installer from a fresh, separate directory.")
  sys.exit(1)


def validate_config(
    env_vars: dict[str, str], state_data: dict[str, Any]
) -> tuple[str, str, str, str | None]:
  """Validates the extracted environment variables against the state file.

  Args:
    env_vars: A dictionary of environment variables.
    state_data: A dictionary containing existing deployment state.

  Returns:
    A tuple containing (project_id, region, app_name, staging_bucket).
  """
  project_id = env_vars.get("GOOGLE_CLOUD_PROJECT")
  region = env_vars.get("REGION")
  app_name = env_vars.get("APP_NAME")
  staging_bucket = env_vars.get("STAGING_BUCKET")

  # Staging bucket is now optional as it is auto-derived in the shell script
  if not (project_id and region and app_name):
    print(
        "Error: Missing required environment variables (GOOGLE_CLOUD_PROJECT, REGION,"
        " APP_NAME) in agent_config.env."
    )
    sys.exit(1)

  if state_data:
    # CLI Lock Validation
    existing_project = state_data.get("project_id")
    existing_region = state_data.get("region")
    existing_app_name = state_data.get("app_name")

    if existing_project and existing_project != project_id:
      print_conflict_error("project_id", project_id, existing_project)

    if existing_region and existing_region != region:
      print_conflict_error("region", region, existing_region)

    if existing_app_name and existing_app_name != app_name:
      print_conflict_error("app_name", app_name, existing_app_name)

    print("✅ Verified agent config against existing deployment state.")

  return project_id, region, app_name, staging_bucket


def check_required_datastores(
    env_vars: dict[str, str], datastore_imports: dict[str, str]
) -> None:
  """Validates that necessary datastores are provided for enabled agents."""

  # Agents that strictly require a datastore
  # to be provisioned (using their exact agent name)
  agents_requiring_datastore = [
      "agri.biochar_advice",
      # Add other agents here that MUST have a datastore
  ]

  sub_agents = env_vars.get("SUB_AGENTS_OF_ROOT_AGENT", "")
  tools_agents = env_vars.get("SUB_AGENTS_AS_TOOLS_OF_ROOT_AGENT", "")

  all_enabled_agents = [
      a.strip() for a in sub_agents.split(",") if a.strip()
  ] + [a.strip() for a in tools_agents.split(",") if a.strip()]

  missing_datastores = []
  for agent in all_enabled_agents:
    if agent in agents_requiring_datastore:
      # The user must provide the exact agent name as the key
      if agent not in datastore_imports:
        missing_datastores.append(agent)

  if missing_datastores:
    print("❌ Error: Missing required Datastore mappings in DATASTORE_IMPORTS:")
    for missing in missing_datastores:
      print(f"   -> Agent '{missing}' requires datastore key '{missing}'")
    print(
        "   Please update the DATASTORE_IMPORTS JSON in agent_config.env to"
        f' include these keys (e.g. {{"{missing_datastores[0]}": "gs://..."}}).'
    )
    sys.exit(1)


def render_config(installer_root: str) -> None:
  """Renders the agent configuration and updates the installer state.

  Args:
    installer_root: The root directory of the installer.
  """
  # 1. Capture specific environment variables from agent_config.env directly
  env_file_path = os.path.join(installer_root, "agent_pack", "agent_config.env")

  try:
    raw_env_vars = dotenv.dotenv_values(dotenv_path=env_file_path)
    if not raw_env_vars:
      raise FileNotFoundError(f"Configuration file '{env_file_path}' empty.")
    # Normalize values to ensure all are strings (mapping None -> "")
    env_vars = {k: v or "" for k, v in raw_env_vars.items()}
  except FileNotFoundError:
    print(f"Error: Configuration file '{env_file_path}' not found.")
    sys.exit(1)

  # 3. Handle Dedicated State Store (Locking & Additive Logic)
  state_file_path = os.path.join(
      installer_root, "backend", "installer_kit", "installer_state.json"
  )
  state_data = {}
  if os.path.exists(state_file_path):
    try:
      with open(state_file_path, "r") as f:
        state_data = json.load(f)
    except Exception as e:
      print(f"Failed to read existing installer state: {e}")
      sys.exit(1)

  # 4. Extract and Validate
  project_id, region, app_name, staging_bucket = validate_config(
      env_vars, state_data
  )
  session_db_type = env_vars.get("SESSION_DB_TYPE", "database")
  provision_agent_db = (session_db_type == "database")

  # Extract and validate DATASTORE_IMPORTS
  datastore_imports_str = env_vars.get("DATASTORE_IMPORTS", "{}")
  try:
    datastore_imports = json.loads(datastore_imports_str)
    if not isinstance(datastore_imports, dict):
      raise ValueError("DATASTORE_IMPORTS must be a JSON dictionary.")
  except json.JSONDecodeError as e:
    print(f"❌ Error: Invalid JSON in DATASTORE_IMPORTS: {e}")
    sys.exit(1)
  except ValueError as e:
    print(f"❌ Error: {e}")
    sys.exit(1)

  # Check if the enabled agents have their required datastore mappings provided
  check_required_datastores(env_vars, datastore_imports)

  # 5. Set paths for Jinja template
  templates_dir = os.path.join(
      installer_root, "backend", "installer_kit", "templates", "tf_configs"
  )
  template_name = "main_tfvars.tfvars.j2"
  output_file = os.path.join(
      installer_root,
      "backend",
      "installer_kit",
      "terraform",
      "generated-terraform.tfvars",
  )

  if not os.path.exists(os.path.join(templates_dir, template_name)):
    print(f"Error: Template {template_name} not found in {templates_dir}")
    sys.exit(1)

  # 5. Build the context dictionary for the single template
  was_onix_enabled = state_data.get("enable_onix", False)

  context = {
      # Core & Toggles
      "project_id": project_id,
      "region": region,
      "app_name": app_name,
      "deployment_size": state_data.get("deployment_size", "small"),
      # Feature Toggles
      "enable_onix": was_onix_enabled,
      "enable_agent": True,
      # Preserve existing ONIX configuration if it exists,
      # otherwise provide defaults
      "provision_adapter_infra": state_data.get(
          "provision_adapter_infra", False
      ),
      "provision_gateway_infra": state_data.get(
          "provision_gateway_infra", False
      ),
      "provision_registry_infra": state_data.get(
          "provision_registry_infra", False
      ),
      "enable_cloud_armor": state_data.get("enable_cloud_armor", False),
      "rate_limit_count": state_data.get("rate_limit_count", 100),
      "provision_agent_db": provision_agent_db,
      "agent_engine_id": state_data.get("agent_engine_id", ""),
      "datastore_imports": datastore_imports,
  }

  # 6. Persist the updated state back to disk
  try:
    with open(state_file_path, "w") as f:
      json.dump(context, f, indent=4)
  except Exception as e:
    print(f"Error writing to {state_file_path}: {e}")
    sys.exit(1)

  env = Environment(loader=FileSystemLoader(templates_dir))
  template = env.get_template(template_name)

  rendered_output = template.render(context)

  with open(output_file, "w") as f:
    f.write(rendered_output)

  print(
      f"Successfully generated {output_file} from main_tfvars.tfvars.j2"
      " template."
  )


def main():
  """Main entry point for path resolution and rendering."""
  script_dir = os.path.dirname(os.path.abspath(__file__))
  installer_root = os.path.abspath(os.path.join(script_dir, ".."))
  render_config(installer_root)


if __name__ == "__main__":
  main()
