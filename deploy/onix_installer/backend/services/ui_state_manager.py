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
import logging
import os
from typing import Any, Dict

logger = logging.getLogger(__name__)

DB_FILE = "ui_state.json" 

def _get_db_file_path() -> str:
    current_dir = os.path.dirname(__file__)
    backend_dir = os.path.abspath(os.path.join(current_dir, '..')) 
    return os.path.join(backend_dir, DB_FILE)

def load_all_data() -> Dict[str, Any]:
    file_path = _get_db_file_path()
    logger.debug(f"Attempting to load data from: {file_path}")

    if not os.path.exists(file_path) or os.path.getsize(file_path) == 0:
        logger.info(f"UI state file '{file_path}' not found or is empty. Returning empty dictionary.")
        return {}
    try:
        with open(file_path, "r") as f:
            data = json.load(f)
            logger.debug(f"Successfully loaded data from '{file_path}'.")
            return data
    except json.JSONDecodeError as e:
        logger.warning(f"UI state file '{file_path}' is corrupted or not valid JSON: {e}. Returning empty dictionary.")
        return {}
    except IOError as e:
        logger.error(f"Error reading UI state file '{file_path}': {e}. Returning empty dictionary.")
        return {}

def _save_data(data: Dict[str, Any]):
    file_path = _get_db_file_path()
    try:
        with open(file_path, "w") as f:
            json.dump(data, f, indent=4)
        logger.debug(f"Successfully saved data to '{file_path}'.")
    except IOError as e:
        logger.error(f"Error writing to UI state file '{file_path}': {e}")
        raise 

def store_bulk_values(items: Dict[str, Any]):
    """
    Accepts a dictionary of key-value pairs for bulk storage/update.
    """
    data_store = load_all_data()
    data_store.update(items)
    _save_data(data_store)
    logger.info(f"UI State: Stored/Updated bulk keys: {list(items.keys())}.")