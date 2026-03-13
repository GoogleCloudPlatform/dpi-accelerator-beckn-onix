#!/usr/bin/env python3
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

import json
import os
import sys
from jinja2 import Environment, FileSystemLoader


def print_conflict_error(field_name, new_val, existing_val):
  print(f"❌ Error: Conflict in '{field_name}'.")
  print(f"   The file 'agent_config.env' specifies {field_name.upper()}='{new_val}',")
  print(f"   but this directory has an existing deployment for {field_name.upper()}='{existing_val}'.")
  print(f"   -> To modify the existing infrastructure, please use '{existing_val}' in your env file.")
  print(f"   -> To create a brand new environment, please run the installer from a fresh, separate directory.")
  sys.exit(1)


def parse_env_file(env_file_path):
  """Parses environment variables from a .env file."""
  if not os.path.exists(env_file_path):
    print(f"Error: Configuration file '{env_file_path}' not found.")
    sys.exit(1)

  env_vars = {}
  with open(env_file_path, "r") as f:
    for line in f:
      line = line.strip()
      if not line or line.startswith("#"):
        continue
      if "=" in line:
        key, val = line.split("=", 1)
        key = key.strip()
        # Strip optional quotes and whitespace
        val = val.strip("\"' ")
        env_vars[key.strip()] = val
  return env_vars


def render_config(installer_root):
  """Renders the agent configuration and updates the installer state."""
  # 1. Capture specific environment variables from agent_config.env directly
  env_file_path = os.path.join(installer_root, "agent_pack", "agent_config.env")
  env_vars = parse_env_file(env_file_path)

  project_id = env_vars.get("PROJECT_ID")
  region = env_vars.get("REGION")
  app_name = env_vars.get("APP_NAME")
  agent_image_url = env_vars.get("AGENT_IMAGE_URL")

  if not all([project_id, region, app_name, agent_image_url]):
    print(
        "Error: Missing required environment variables (PROJECT_ID, REGION,"
        " APP_NAME, AGENT_IMAGE_URL) in agent_config.env."
    )
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
    except Exception as e:
      print(f"Failed to read existing installer state: {e}")
      sys.exit(1)

  # 4. Set paths for Jinja template
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
      "allowed_regions": state_data.get("allowed_regions", []),
      "rate_limit_count": state_data.get("rate_limit_count", 100),
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
