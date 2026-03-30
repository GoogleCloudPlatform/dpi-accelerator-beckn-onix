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

"""Entry point for managing DPI Agent deployments on Vertex AI Agent Engine.

This module provides a command-line interface to create, update, or delete
the DPI agent on Vertex AI Agent Engine.

It configures the deployment by setting up resource limits, environment
variables, networking (Private Service Connect), and conditionally enabled
plugins like long-term memory. The main agent is packaged via the `app` module
for seamless integration with Vertex AI.
"""

import argparse
import json
import os
import pprint
import sys
from typing import Any

import dotenv

# Remove reserved and deployment-time variables from the
# agent's runtime environment to reduce environment variable count
EXCLUDED_RUNTIME_VARS: tuple[str, ...] = (
    "GOOGLE_CLOUD_PROJECT",
    "REGION",
    "STAGING_BUCKET",
    "AGENT_SA_EMAIL",
    "APP_NAME",
    "MIN_INSTANCES",
    "MAX_INSTANCES",
    "CONTAINER_CONCURRENCY",
    "CPU_LIMIT",
    "MEMORY_LIMIT",
    "PYTHON_VERSION",
    "GCS_DIR_NAME",
)
# The user clones the repo into
# 'dpi_agent_blueprint' within the agent_pack directory
script_dir = os.path.dirname(os.path.abspath(__file__))
dpi_agent_blueprint_repo = os.path.join(script_dir, "dpi_agent_blueprint")
installer_root = os.path.abspath(os.path.join(script_dir, ".."))

if not os.path.exists(dpi_agent_blueprint_repo):
  print(f"Error: Agent blueprint repository not found at {dpi_agent_blueprint_repo}")
  print(
      "Please ensure you have cloned the dpi_agent_blueprint repository there."
  )
  sys.exit(1)

# Add script_dir to sys.path so we can import 'dpi_agent_blueprint' directly
sys.path.append(script_dir)

try:
  from dpi_agent_blueprint.agent_engine import app
  from dpi_agent_blueprint.agent_engine import memory
except ImportError as e:
  print(
      f"Error: Could not import dpi_agent_blueprint from {dpi_agent_blueprint_repo}"
  )
  print(f"ImportError: {repr(e)}")
  sys.exit(1)

try:
  from google.cloud.aiplatform import vertexai
  from google.cloud.aiplatform.vertexai._genai.types import AgentServerMode
except ImportError:
  import vertexai
  from vertexai._genai.types import AgentServerMode


def update_installer_state(agent_engine_id: str) -> None:
  """Persists the Agent Engine ID to the installer's state file."""
  state_file_path = os.path.join(
      installer_root, "backend", "installer_kit", "installer_state.json"
  )
  try:
    with open(state_file_path, "r") as f:
      state_data = json.load(f)

    state_data["agent_engine_id"] = agent_engine_id

    with open(state_file_path, "w") as f:
      json.dump(state_data, f, indent=4)
    print("Updated installer_state.json with agent_engine_id.")
  except FileNotFoundError:
    print(
        f"Warning: installer_state.json not found at {state_file_path}. Cannot"
        " persist agent_engine_id."
    )
  except Exception as e:
    print(f"Error: Failed to update installer_state.json: {e}")


def build_agent_config(
    args: argparse.Namespace, env_vars: dict[str, str]
) -> tuple[dict[str, Any], str, str, str | None]:
  """Validates environment variables and builds the agent configuration.

  Args:
    args: Command line arguments from argparse.
    env_vars: A dictionary of environment variables.

  Returns:
    A tuple containing:
      config: The Agent Engine configuration dictionary.
      project: The Google Cloud Project ID.
      location: The Google Cloud Region.
      staging_bucket: The GCS bucket name.

  Raises:
    ValueError: If required environment variables are missing.
  """

  project = env_vars.get("GOOGLE_CLOUD_PROJECT")
  if not project:
    raise ValueError("GOOGLE_CLOUD_PROJECT must be set in the env file.")

  location = env_vars.get("REGION")
  if not location:
    raise ValueError("REGION must be set in the env file.")

  staging_bucket = env_vars.get("STAGING_BUCKET")
  if not staging_bucket and not args.delete:
    raise ValueError("STAGING_BUCKET must be set in the env file.")

  app_name = env_vars.get("APP_NAME", "test")
  display_name = f"dpi-{app_name}-agent-engine"
  description = f"Reasoning Engine for Becknonix APP: {app_name}"

  config = {
      "display_name": env_vars.get("DISPLAY_NAME", display_name),
      "description": env_vars.get("DESCRIPTION", description),
      "agent_framework": "google-adk",
      "staging_bucket": staging_bucket,
      "gcs_dir_name": env_vars.get("GCS_DIR_NAME", "agent-engine-artifacts"),
      "min_instances": int(env_vars.get("MIN_INSTANCES", "1")),
      "max_instances": int(env_vars.get("MAX_INSTANCES", "10")),
      "container_concurrency":
          int(env_vars.get("CONTAINER_CONCURRENCY", "4")),
      "resource_limits": {
          "cpu": str(env_vars.get("CPU_LIMIT", "4")),
          "memory": str(env_vars.get("MEMORY_LIMIT", "16Gi")),
      },
      "python_version": env_vars.get("PYTHON_VERSION", args.python_version),
      "requirements": os.path.join(dpi_agent_blueprint_repo, "requirements.txt"),
      "extra_packages": [
          "dpi_agent_blueprint"
      ],
      "env_vars": env_vars,
      # Set the agent server mode to EXPERIMENTAL for BiDi
      "agent_server_mode": AgentServerMode.EXPERIMENTAL,
  }

  service_account = env_vars.get("AGENT_SA_EMAIL")
  if service_account:
    config["service_account"] = service_account

  session_db_url = env_vars.get("SESSION_DB_URL")
  redis_host = env_vars.get("REDIS_HOST")
  if session_db_url or redis_host:
    network_attachment = env_vars.get("NETWORK_ATTACHMENT_ID")
    if not network_attachment:
      raise ValueError(
          "NETWORK_ATTACHMENT_ID must be set if using private integrations."
      )

    network_attachment_id = network_attachment
    if not network_attachment_id.startswith("projects/"):
      network_attachment_id = (
          f"projects/{project}/regions/{location}/networkAttachments/"
          f"{network_attachment}"
      )

    config["psc_interface_config"] = {"network_attachment": network_attachment_id}

  if env_vars.get("ENABLE_LONG_TERM_MEMORY", "false").lower() == "true":
    gemini_model_location = env_vars.get("GOOGLE_CLOUD_LOCATION")
    if not gemini_model_location:
      raise ValueError("GOOGLE_CLOUD_LOCATION must be set in the env file.")
    config["context_spec"] = {
        "memory_bank_config": memory.get_memory_bank_config(
            project=project, location=gemini_model_location
        )
    }

  return config, project, location, staging_bucket


def main():
  """Main function for Agent Engine ADK App Deployment."""
  parser = argparse.ArgumentParser()
  parser.add_argument("--python-version", default="3.13")
  parser.add_argument(
      "--env-vars-file",
      required=True,
      help=(
          "Path to the .env file containing the environment variables to deploy"
          " with."
      ),
  )
  action_group = parser.add_mutually_exclusive_group(required=True)
  action_group.add_argument(
      "--create", action="store_true", help="Create the agent engine app."
  )
  action_group.add_argument(
      "--update", action="store_true", help="Update the agent engine app."
  )
  action_group.add_argument(
      "--delete", action="store_true", help="Delete the agent engine app."
  )
  parser.add_argument(
      "--agent-engine",
      help=(
          "The ID or full resource name of the agent engine app to update or"
          " delete."
      ),
  )
  args = parser.parse_args()

  if (args.update or args.delete) and not args.agent_engine:
    parser.error("--agent-engine is required for update and delete operations.")

  # 2. Load environment variables
  env_file = args.env_vars_file
  if not os.path.exists(env_file):
    print(f"Error: Environment file not found at {env_file}")
    sys.exit(1)

  env_vars = {}
  try:
    raw_env_vars = dotenv.dotenv_values(dotenv_path=env_file)
    if not raw_env_vars:
      raise FileNotFoundError(f"No environment variables found in {env_file}")
    # Normalize values to ensure all are strings (mapping None -> "")
    env_vars = {k: v or "" for k, v in raw_env_vars.items()}
    print(f"Loaded environment variables from {env_file}")
  except Exception as e:
    print(f"Error loading environment variables: {e}")
    sys.exit(1)

  try:
    config, project, location, staging_bucket = build_agent_config(args, env_vars)
  except ValueError as e:
    print(f"Configuration Error: {e}")
    sys.exit(1)

  # Remove reserved deployment-time variables from the
  # agent's runtime environment to reduce environment variable count
  for env_key in EXCLUDED_RUNTIME_VARS:
    if env_key in env_vars:
      del env_vars[env_key]

  try:
    vertexai.init(project=project, location=location)
    client = vertexai.Client(project=project, location=location)
  except Exception as e:
    print(f"Initialization Error: {e}")
    sys.exit(1)

  if args.delete:
    try:
      print(f"Deleting ADK agent: {args.agent_engine}")
      agent_name = args.agent_engine
      if not agent_name.startswith("projects/"):
        agent_name = (
            f"projects/{project}/locations/{location}/reasoningEngines/"
            f"{args.agent_engine}"
        )
      client.agent_engines.delete(name=agent_name, force=True)
      print("Successfully deleted ADK agent.")
    except Exception as e:
      print(f"Error: Agent delete failed: {e}")
      sys.exit(1)
    return

  try:
    if args.create:
      print(
          "Deploying ADK agent with Agent Engine config:\n"
          f"{pprint.pformat(config)}"
      )
      remote_agent = client.agent_engines.create(
          agent=app.app,
          config=config,
      )
      print(
          f"Successfully deployed ADK agent: {remote_agent.api_resource.name}"
      )
      update_installer_state(remote_agent.api_resource.name)

    elif args.update:
      print(
          "Updating ADK agent with Agent Engine config:\n"
          f"{pprint.pformat(config)}"
      )
      agent_name = args.agent_engine
      if not agent_name.startswith("projects/"):
        agent_name = (
            f"projects/{project}/locations/{location}/reasoningEngines/"
            f"{args.agent_engine}"
        )
      remote_agent = client.agent_engines.update(
          name=agent_name,
          agent=app.app,
          config=config,
      )
      print(f"Successfully updated ADK agent: {remote_agent.api_resource.name}")

  except Exception as e:
    print(f"Error: Agent operation failed: {e}")
    sys.exit(1)


if __name__ == "__main__":
  main()
