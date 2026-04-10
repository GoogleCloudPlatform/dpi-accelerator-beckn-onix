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

import logging
import os
from typing import Any, Dict, List

from config import app_config_generator as app_config
from core import utils
from core.constants import GENERATED_CONFIGS_DIR
from core.models import ConfigGenerationRequest
from services import ui_state_manager

VALID_CONFIG_EXTENSIONS = {'.yaml', '.yml'}

logger = logging.getLogger(__name__)

def generate_initial_configs(request: ConfigGenerationRequest) -> None:
    """Generates the initial configuration files based on the deployment request.
    This function triggers the generation of application configuration YAMLs.
    If files already exist, they are NOT overwritten.
    Args:
        request: A ConfigGenerationRequest object containing component flags and settings.
    """
    logger.info("Triggering initial config generation.")
    app_config.generate_app_configs(request)


def get_all_config_paths() -> List[str]:
    """Retrieves a sorted list of valid configuration file paths relative to the artifacts directory.
    Filters files based on valid extensions and visibility rules (e.g., hiding 
    'routing_configs' if the Adapter component is not selected in the UI state).
    Returns:
        A sorted list of string paths relative to GENERATED_CONFIGS_DIR.
    """
    file_paths = []
    if not os.path.exists(GENERATED_CONFIGS_DIR):
        logger.warning(f"Artifacts directory {GENERATED_CONFIGS_DIR} does not exist.")
        return file_paths

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
            if file.startswith('.'):
                continue

            _, ext = os.path.splitext(file)
            if ext.lower() not in VALID_CONFIG_EXTENSIONS:
                continue

            full_path = os.path.join(root, file)
            relative_path = os.path.relpath(full_path, GENERATED_CONFIGS_DIR)

            # Routing configs are only relevant for the Adapter service.
            if not is_adapter_enabled and relative_path.startswith("routing_configs"):
                continue

            file_paths.append(relative_path)

    return sorted(file_paths)


def get_config_content(relative_path: str) -> str:
    """Reads the raw text content of a specified configuration file.
    Args:
        relative_path: The file path relative to the GENERATED_CONFIGS_DIR.
    Returns:
        The content of the file as a string.
    Raises:
        ValueError: If the path resolves to a location outside the allowed directory.
        FileNotFoundError: If the file does not exist.
        IOError: If there is an error reading the file.
    """
    safe_path = _validate_path(relative_path)
    return utils.read_file_content(safe_path)


def update_config_content(relative_path: str, content: str) -> None:
    """Overwrites the content of a specified configuration file.
    Args:
        relative_path: The file path relative to the GENERATED_CONFIGS_DIR.
        content: The new text content to write to the file.
    Raises:
        ValueError: If the path resolves to a location outside the allowed directory.
        IOError: If there is an error writing to the file.
    """
    safe_path = _validate_path(relative_path)
    utils.write_file_content(safe_path, content)


def _validate_path(relative_path: str) -> str:
    """Validates and resolves a relative path to ensure it remains within the allowed directory.
    Prevents directory traversal attacks by checking the absolute path.
    Args:
        relative_path: The user-provided relative path.
    Returns:
        The absolute, normalized path to the file.
    Raises:
        ValueError: If the resolved path attempts to escape the GENERATED_CONFIGS_DIR.
    """
    # Normalize path to remove .. and .
    full_path = os.path.abspath(os.path.join(GENERATED_CONFIGS_DIR, relative_path))

    # Check if the resolved path starts with the artifacts directory path
    base_dir = os.path.abspath(GENERATED_CONFIGS_DIR)

    # Ensure the base directory path ends with a separator to prevent prefix-based escapes.
    if not base_dir.endswith(os.sep):
        base_dir += os.sep

    # Check if the resolved path is within the artifacts directory.
    if not full_path.startswith(base_dir):
        raise ValueError(f"Invalid path: {relative_path}. Access denied.")
    return full_path