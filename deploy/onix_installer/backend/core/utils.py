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
from jinja2 import Environment, FileSystemLoader, TemplateNotFound
import yaml

from core.constants import ANSI_ESCAPE_PATTERN

logger = logging.getLogger(__name__)

def render_jinja_template(template_dir: str, template_name: str, context: dict) -> str:
    """
    Renders a Jinja2 template with the given context.

    Args:
        template_dir (str): The directory where the template file is located.
        template_name (str): The name of the template file.
        context (dict): A dictionary of variables to pass to the template.

    Returns:
        str: The rendered content of the template.

    Raises:
        FileNotFoundError: If the Jinja2 template is not found.
        RuntimeError: If an error occurs during template rendering.
    """
    try:
        env = Environment(loader=FileSystemLoader(template_dir), trim_blocks=True, lstrip_blocks=True)
        template = env.get_template(template_name)
        rendered_content = template.render(context)
        logger.debug(f"Successfully rendered template: '{template_name}' from '{template_dir}'")
        return rendered_content
    except TemplateNotFound:
        logger.error(f"Jinja2 template not found: '{os.path.join(template_dir, template_name)}'")
        raise FileNotFoundError(
            f"Jinja2 template not found: '{os.path.join(template_dir, template_name)}'"
        )
    except Exception as e:
        logger.exception(f"Error rendering template '{template_name}': {e}")
        raise RuntimeError(f"Error rendering template '{template_name}': {e}")

def read_file_content(file_path: str) -> str:
    """
    Reads and returns the entire content of a file as a string.
    """
    try:
        with open(file_path, 'r') as f:
            content = f.read()
        logger.debug(f"Successfully read content from file: '{file_path}'")
        return content
    except FileNotFoundError:
        logger.error(f"File not found: '{file_path}'")
        raise FileNotFoundError(f"File not found: '{file_path}'")
    except IOError as e:
        logger.exception(f"Error reading file '{file_path}': {e}")
        raise IOError(f"Error reading file '{file_path}': {e}")

def read_json_file(file_path: str) -> dict:
    """
    Reads and returns the content of a JSON file as a dictionary.
    """
    try:
        with open(file_path, 'r') as f:
            content = json.load(f)
        logger.debug(f"Successfully read JSON from file: '{file_path}'")
        return content
    except FileNotFoundError:
        logger.error(f"JSON file not found: '{file_path}'")
        raise FileNotFoundError(f"JSON file not found: '{file_path}'")
    except json.JSONDecodeError as e:
        logger.error(f"Error decoding JSON from '{file_path}': {e}")
        raise ValueError(f"Error decoding JSON from '{file_path}': {e}")
    except IOError as e:
        logger.exception(f"Error reading JSON file '{file_path}': {e}")
        raise IOError(f"Error reading file '{file_path}': {e}")

def read_yaml_file(file_path: str) -> dict:
    """
    Reads and returns the content of a YAML file as a dictionary.
    """
    try:
        with open(file_path, 'r') as f:
            content = yaml.safe_load(f) # Use safe_load for security
        logger.debug(f"Successfully read YAML from file: '{file_path}'")
        return content
    except FileNotFoundError:
        logger.error(f"YAML file not found: '{file_path}'")
        raise FileNotFoundError(f"YAML file not found: '{file_path}'")
    except yaml.YAMLError as e:
        logger.error(f"Error parsing YAML from '{file_path}': {e}")
        raise ValueError(f"Error parsing YAML from '{file_path}': {e}")
    except IOError as e:
        logger.exception(f"Error reading YAML file '{file_path}': {e}")
        raise IOError(f"Error reading file '{file_path}': {e}")

def write_file_content(file_path: str, content: str):
    """
    Writes content to a specified file, ensuring the directory exists.
    """
    try:
        # Ensure the directory exists
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        with open(file_path, 'w') as f:
            f.write(content)
        logger.info(f"Successfully wrote content to file: '{file_path}'")
    except IOError as e:
        logger.exception(f"Error writing to file '{file_path}': {e}")
        raise IOError(f"Error writing to file '{file_path}': {e}")

async def stream_subprocess_output(process: asyncio.subprocess.Process, websocket, stream_name: str):
    """
    Asynchronously streams output from a subprocess to the WebSocket.
    This helper is common and efficient, so retaining it.
    """
    if process.stdout:
        while True:
            line = await process.stdout.readline()
            if not line:
                break
            decoded_line = line.decode(errors='replace').strip()
            # Clean ANSI escape codes before sending to frontend.
            cleaned_line = ANSI_ESCAPE_PATTERN.sub('', decoded_line)
            if cleaned_line: # Only send non-empty lines after cleaning.
                log_message = {"type": "log", "action": stream_name, "message": cleaned_line}
                await websocket.send_text(json.dumps(log_message))
                logger.info(f"[{stream_name}] {cleaned_line}")