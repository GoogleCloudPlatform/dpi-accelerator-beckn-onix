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

import os
import logging
from typing import List, Dict, Any
from core.constants import ARTIFACTS_DIR, GENERATED_CONFIGS_DIR
from core import utils
import config.app_config_generator as app_config
from core.models import ConfigGenerationRequest
from services import ui_state_manager

logger = logging.getLogger(__name__)

# Define allowed configuration file extensions
VALID_CONFIG_EXTENSIONS = {'.yaml', '.yml'}

def generate_initial_configs(request: ConfigGenerationRequest):
    """
    Generates the initial configuration files based on the deployment request.
    If files already exist, they are NOT overwritten.
    """
    logger.info("Triggering initial config generation.")
    app_config.generate_app_configs(request)


def get_all_config_paths() -> List[str]:
    """
    Returns a list of all file paths relative to the ARTIFACTS_DIR.
    Filters out files that are not valid configuration types.
    Hides 'routing_configs' if the Adapter (BAP/BPP) is not being deployed.
    """
    file_paths = []
    if not os.path.exists(GENERATED_CONFIGS_DIR):
        logger.warning(f"Artifacts directory {GENERATED_CONFIGS_DIR} does not exist.")
        return file_paths

    # 1. Check User's Deployment Goal to determine if Adapter is needed
    try:
        ui_data = ui_state_manager.load_all_data()
        deployment_goal = ui_data.get('deploymentGoal', {})
        # Adapter is required if either BAP or BPP is selected
        is_adapter_enabled = deployment_goal.get('bap', False) or deployment_goal.get('bpp', False)
    except Exception as e:
        logger.warning(f"Could not load UI state to filter configs, defaulting to show all: {e}")
        is_adapter_enabled = True

    for root, _, files in os.walk(GENERATED_CONFIGS_DIR):
        for file in files:
            # 2. Skip hidden files
            if file.startswith('.'):
                continue

            # 3. Filter by extension
            _, ext = os.path.splitext(file)
            if ext.lower() not in VALID_CONFIG_EXTENSIONS:
                continue

            full_path = os.path.join(root, file)
            relative_path = os.path.relpath(full_path, GENERATED_CONFIGS_DIR)

            # 4. Filter Routing Configs based on Adapter selection
            # Routing configs are only relevant for the Adapter service.
            if not is_adapter_enabled and relative_path.startswith("routing_configs"):
                continue

            file_paths.append(relative_path)

    return sorted(file_paths)

def get_config_content(relative_path: str) -> str:
    """
    Reads the content of a configuration file.
    Validates that the path is safely within ARTIFACTS_DIR.
    """
    safe_path = _validate_path(relative_path)
    return utils.read_file_content(safe_path)

def update_config_content(relative_path: str, content: str):
    """
    Updates the content of a configuration file.
    Validates that the path is safely within ARTIFACTS_DIR.
    """
    safe_path = _validate_path(relative_path)
    utils.write_file_content(safe_path, content)

def _validate_path(relative_path: str) -> str:
    """
    Ensures the requested path is inside the ARTIFACTS_DIR to prevent directory traversal.
    """
    # Normalize path to remove .. and .
    full_path = os.path.abspath(os.path.join(GENERATED_CONFIGS_DIR, relative_path))
    
    # Check if the resolved path starts with the artifacts directory path
    if not full_path.startswith(os.path.abspath(GENERATED_CONFIGS_DIR)):
        raise ValueError(f"Invalid path: {relative_path}. Access denied.")
    return full_path