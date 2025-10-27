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

import unittest
import os
import sys
import json
import asyncio
from unittest.mock import patch, mock_open, MagicMock, AsyncMock, call
from jinja2 import TemplateNotFound
import logging

# Ensure the project root is in the path for imports
project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
sys.path.insert(0, project_root)

from core import utils
from core.constants import ANSI_ESCAPE_PATTERN # Import constants to use in test

class TestCoreUtils(unittest.IsolatedAsyncioTestCase): # Changed to IsolatedAsyncioTestCase for async tests

    def setUp(self):
        # Patch the logger used within core.utils
        self.mock_logger = patch('core.utils.logger').start()
        self.mock_logger.handlers = []
        self.mock_logger.propagate = False
        self.mock_logger.setLevel(logging.NOTSET) # Set level to NOTSET to ensure all logs are captured by mock

    def tearDown(self):
        patch.stopall() # Stop all patches started in setUp and within tests

    @patch('core.utils.FileSystemLoader')
    @patch('core.utils.Environment')
    def test_render_jinja_template_success(self, MockEnvironment, MockFileSystemLoader):
        """
        Test successful rendering of a Jinja2 template.
        """
        mock_env_instance = MockEnvironment.return_value
        mock_template = MagicMock()
        mock_env_instance.get_template.return_value = mock_template
        mock_template.render.return_value = "rendered content"

        template_dir = "/tmp/templates"
        template_name = "test.j2"
        context = {"name": "World"}
        
        result = utils.render_jinja_template(template_dir, template_name, context)
        
        MockFileSystemLoader.assert_called_once_with(template_dir)
        
        mock_loader_instance = MockFileSystemLoader.return_value

        MockEnvironment.assert_called_once_with(
            loader=mock_loader_instance,
            trim_blocks=True,
            lstrip_blocks=True
        )
        
        mock_env_instance.get_template.assert_called_once_with(template_name)
        mock_template.render.assert_called_once_with(context)
        self.assertEqual(result, "rendered content")
        self.mock_logger.debug.assert_called_once_with(f"Successfully rendered template: '{template_name}' from '{template_dir}'")


    @patch('core.utils.Environment')
    def test_render_jinja_template_not_found(self, MockEnvironment):
        """
        Test case where the Jinja2 template is not found.
        """
        mock_env_instance = MockEnvironment.return_value
        mock_env_instance.get_template.side_effect = TemplateNotFound("test.j2")

        template_dir = "/tmp/templates"
        template_name = "non_existent.j2"
        context = {}

        with self.assertRaisesRegex(FileNotFoundError, "Jinja2 template not found"):
            utils.render_jinja_template(template_dir, template_name, context)
        
        self.mock_logger.error.assert_called_once_with(
            f"Jinja2 template not found: '{os.path.join(template_dir, template_name)}'"
        )

    @patch('core.utils.Environment')
    def test_render_jinja_template_rendering_error(self, MockEnvironment):
        """
        Test case where an error occurs during template rendering.
        """
        mock_env_instance = MockEnvironment.return_value
        mock_template = MagicMock()
        mock_env_instance.get_template.return_value = mock_template
        mock_template.render.side_effect = Exception("Rendering error")

        template_dir = "/tmp/templates"
        template_name = "error.j2"
        context = {}

        with self.assertRaisesRegex(RuntimeError, "Error rendering template 'error.j2': Rendering error"):
            utils.render_jinja_template(template_dir, template_name, context)
        
        self.mock_logger.exception.assert_called_once()

    @patch('builtins.open', new_callable=mock_open, read_data="file content")
    def test_read_file_content_success(self, mock_file):
        """
        Test successful reading of file content.
        """
        file_path = "/tmp/test.txt"
        content = utils.read_file_content(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.assertEqual(content, "file content")
        self.mock_logger.debug.assert_called_once_with(f"Successfully read content from file: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open)
    def test_read_file_content_not_found(self, mock_file):
        """
        Test FileNotFoundError when reading file content.
        """
        mock_file.side_effect = FileNotFoundError
        file_path = "/tmp/non_existent.txt"
        with self.assertRaisesRegex(FileNotFoundError, "File not found"):
            utils.read_file_content(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.error.assert_called_once_with(f"File not found: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open)
    def test_read_file_content_io_error(self, mock_file):
        """
        Test IOError when reading file content.
        """
        mock_file.side_effect = IOError("Permission denied")
        file_path = "/tmp/no_permission.txt"
        with self.assertRaisesRegex(IOError, "Error reading file"):
            utils.read_file_content(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.exception.assert_called_once()

    @patch('builtins.open', new_callable=mock_open, read_data=json.dumps({"key": "value"}))
    def test_read_json_file_success(self, mock_file):
        """
        Test successful reading of a JSON file.
        """
        file_path = "/tmp/test.json"
        content = utils.read_json_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.assertEqual(content, {"key": "value"})
        self.mock_logger.debug.assert_called_once_with(f"Successfully read JSON from file: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open)
    def test_read_json_file_not_found(self, mock_file):
        """
        Test FileNotFoundError when reading JSON file.
        """
        mock_file.side_effect = FileNotFoundError
        file_path = "/tmp/non_existent.json"
        with self.assertRaisesRegex(FileNotFoundError, "JSON file not found"):
            utils.read_json_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.error.assert_called_once_with(f"JSON file not found: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open, read_data="invalid json")
    def test_read_json_file_decode_error(self, mock_file):
        """
        Test JSONDecodeError when reading an invalid JSON file.
        """
        file_path = "/tmp/invalid.json"
        with self.assertRaises(ValueError):
            utils.read_json_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.error.assert_called_once()

    @patch('builtins.open', new_callable=mock_open)
    def test_read_json_file_io_error(self, mock_file):
        """
        Test IOError when reading JSON file.
        """
        mock_file.side_effect = IOError("Disk full")
        file_path = "/tmp/disk_full.json"
        with self.assertRaisesRegex(IOError, "Error reading file"):
            utils.read_json_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.exception.assert_called_once()

    @patch('builtins.open', new_callable=mock_open)
    @patch('os.makedirs')
    @patch('os.path.dirname', return_value="/tmp/dir")
    def test_write_file_content_success(self, mock_dirname, mock_makedirs, mock_file):
        """
        Test successful writing of content to a file.
        """
        file_path = "/tmp/dir/output.txt"
        content_to_write = "hello world"
        utils.write_file_content(file_path, content_to_write)
        
        mock_dirname.assert_called_once_with(file_path)
        mock_makedirs.assert_called_once_with("/tmp/dir", exist_ok=True)
        mock_file.assert_called_once_with(file_path, 'w')
        mock_file().write.assert_called_once_with(content_to_write)
        self.mock_logger.info.assert_called_once_with(f"Successfully wrote content to file: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open)
    @patch('os.makedirs')
    @patch('os.path.dirname', return_value="/tmp/dir")
    def test_write_file_content_io_error(self, mock_dirname, mock_makedirs, mock_file):
        """
        Test IOError during writing content to a file.
        """
        mock_file.side_effect = IOError("Disk full")
        file_path = "/tmp/dir/output.txt"
        content_to_write = "hello world"

        with self.assertRaisesRegex(IOError, "Disk full"):
            utils.write_file_content(file_path, content_to_write)
        
        mock_dirname.assert_called_once_with(file_path)
        mock_makedirs.assert_called_once_with("/tmp/dir", exist_ok=True)
        mock_file.assert_called_once_with(file_path, 'w')
        self.mock_logger.exception.assert_called_once()

    @patch('builtins.open', new_callable=mock_open, read_data="key: value\nlist:\n  - item1\n  - item2")
    def test_read_yaml_file_success(self, mock_file):
        """
        Test successful reading of a YAML file.
        """
        file_path = "/tmp/test.yaml"
        expected_content = {"key": "value", "list": ["item1", "item2"]}
        content = utils.read_yaml_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.assertEqual(content, expected_content)
        self.mock_logger.debug.assert_called_once_with(f"Successfully read YAML from file: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open)
    def test_read_yaml_file_not_found(self, mock_file):
        """
        Test FileNotFoundError when reading YAML file.
        """
        mock_file.side_effect = FileNotFoundError
        file_path = "/tmp/non_existent.yaml"
        with self.assertRaisesRegex(FileNotFoundError, "YAML file not found"):
            utils.read_yaml_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.error.assert_called_once_with(f"YAML file not found: '{file_path}'")

    @patch('builtins.open', new_callable=mock_open, read_data="key: : invalid yaml")
    def test_read_yaml_file_parse_error(self, mock_file):
        """
        Test YAMLError when reading an invalid YAML file.
        """
        file_path = "/tmp/invalid.yaml"
        with self.assertRaises(ValueError):
            utils.read_yaml_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.error.assert_called_once()

    @patch('builtins.open', new_callable=mock_open)
    def test_read_yaml_file_io_error(self, mock_file):
        """
        Test IOError when reading YAML file.
        """
        mock_file.side_effect = IOError("Device not ready")
        file_path = "/tmp/device_not_ready.yaml"
        with self.assertRaisesRegex(IOError, "Error reading file"):
            utils.read_yaml_file(file_path)
        mock_file.assert_called_once_with(file_path, 'r')
        self.mock_logger.exception.assert_called_once()

    # --- Tests for stream_subprocess_output ---

    async def test_stream_subprocess_output_success(self):
        """
        Test successful streaming of subprocess output.
        """
        mock_process = MagicMock(spec=asyncio.subprocess.Process)
        mock_process.stdout = AsyncMock()

        # Simulate stdout lines including one with ANSI escape codes
        mock_process.stdout.readline.side_effect = [
            b"Hello from subprocess\n",
            b"\x1b[31mError message\x1b[0m\n", # ANSI red text
            b"Another line\n",
            b"", # End of stream
        ]

        mock_websocket = AsyncMock()
        stream_name = "test_stream"

        await utils.stream_subprocess_output(mock_process, mock_websocket, stream_name)

        # Assertions for readline calls
        mock_process.stdout.readline.assert_has_calls([call(), call(), call(), call()])

        # Assertions for websocket send_text calls
        expected_calls = [
            call(json.dumps({"type": "log", "action": stream_name, "message": "Hello from subprocess"})),
            call(json.dumps({"type": "log", "action": stream_name, "message": "Error message"})), # Cleaned ANSI
            call(json.dumps({"type": "log", "action": stream_name, "message": "Another line"})),
        ]
        mock_websocket.send_text.assert_has_calls(expected_calls)
        self.assertEqual(mock_websocket.send_text.call_count, 3)

        # Assertions for logger info calls
        expected_log_calls = [
            call(f"[{stream_name}] Hello from subprocess"),
            call(f"[{stream_name}] Error message"),
            call(f"[{stream_name}] Another line"),
        ]
        self.mock_logger.info.assert_has_calls(expected_log_calls)
        self.assertEqual(self.mock_logger.info.call_count, 3)

    async def test_stream_subprocess_output_empty_output(self):
        """
        Test streaming with empty subprocess output.
        """
        mock_process = MagicMock(spec=asyncio.subprocess.Process)
        mock_process.stdout = AsyncMock()
        mock_process.stdout.readline.side_effect = [b""] # Immediately end stream

        mock_websocket = AsyncMock()
        stream_name = "empty_stream"

        await utils.stream_subprocess_output(mock_process, mock_websocket, stream_name)

        mock_process.stdout.readline.assert_called_once_with()
        mock_websocket.send_text.assert_not_called()
        self.mock_logger.info.assert_not_called()

    async def test_stream_subprocess_output_no_stdout(self):
        """
        Test handling of a process with no stdout.
        """
        mock_process = MagicMock(spec=asyncio.subprocess.Process)
        mock_process.stdout = None # Simulate no stdout

        mock_websocket = AsyncMock()
        stream_name = "no_stdout_stream"

        await utils.stream_subprocess_output(mock_process, mock_websocket, stream_name)

        # Ensure no attempts to read from stdout were made
        self.assertIsNone(mock_process.stdout)
        mock_websocket.send_text.assert_not_called()
        self.mock_logger.info.assert_not_called()

    async def test_stream_subprocess_output_lines_with_only_ansi(self):
        """
        Test that lines containing only ANSI escape codes are not sent.
        """
        mock_process = MagicMock(spec=asyncio.subprocess.Process)
        mock_process.stdout = AsyncMock()
        mock_process.stdout.readline.side_effect = [
            b"\x1b[32m\x1b[0m\n", # Only ANSI codes, should be cleaned to empty
            b"Actual content here\n",
            b"",
        ]

        mock_websocket = AsyncMock()
        stream_name = "ansi_only_stream"

        await utils.stream_subprocess_output(mock_process, mock_websocket, stream_name)

        expected_calls = [
            call(json.dumps({"type": "log", "action": stream_name, "message": "Actual content here"})),
        ]
        mock_websocket.send_text.assert_has_calls(expected_calls)
        self.assertEqual(mock_websocket.send_text.call_count, 1)
        self.mock_logger.info.assert_called_once_with(f"[{stream_name}] Actual content here")


if __name__ == '__main__':
    unittest.main()