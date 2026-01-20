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
import shutil
import tempfile
from unittest.mock import MagicMock, patch

import sys
# Adjust path to include project root if necessary
project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
sys.path.insert(0, project_root)

# Import the module under test
import services.config_manager as config_manager
from core.models import ConfigGenerationRequest, RegistryConfig

class TestConfigManager(unittest.TestCase):

    def setUp(self):
        # Create a temporary directory to simulate GENERATED_CONFIGS_DIR
        self.test_dir = tempfile.mkdtemp()
        
        # Patch the GENERATED_CONFIGS_DIR constant in the module under test
        self.patcher_dir = patch('services.config_manager.GENERATED_CONFIGS_DIR', self.test_dir)
        self.mock_configs_dir = self.patcher_dir.start()

        # Patch logger to keep test output clean
        self.patcher_logger = patch('services.config_manager.logger')
        self.mock_logger = self.patcher_logger.start()

        # Dummy objects for Pydantic models
        self.dummy_registry_config = RegistryConfig(
            server_url="http://mock-registry.com",
            subscriber_id="mock_subscriber_id",
            key_id="mock_key_id"
        )
        self.dummy_registry_url = "http://mock-registry.com"

    def tearDown(self):
        # Stop patches
        self.patcher_dir.stop()
        self.patcher_logger.stop()
        
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)

    @patch('services.config_manager.app_config.generate_app_configs')
    def test_generate_initial_configs(self, mock_generate_app_configs):
        """Test that generate_initial_configs calls the app_config generator."""
        req = ConfigGenerationRequest(
            project_id="test", 
            region="us-central1", 
            app_name="test-app", 
            components={},
            registry_url=self.dummy_registry_url,
            registry_config=self.dummy_registry_config
        )
        
        config_manager.generate_initial_configs(req)
        
        mock_generate_app_configs.assert_called_once_with(req)
        self.mock_logger.info.assert_called_with("Triggering initial config generation.")

    @patch('services.config_manager.ui_state_manager.load_all_data')
    def test_get_all_config_paths_basic_filtering(self, mock_load_data):
        """Test filtering of extensions and hidden files."""
        # Setup: Mock UI state to enable everything so we don't trigger the routing logic yet
        mock_load_data.return_value = {'deploymentGoal': {'bap': True}}

        # Create dummy files in the temp dir
        os.makedirs(os.path.join(self.test_dir, "subdir"))
        
        # Valid files
        with open(os.path.join(self.test_dir, "config.yaml"), 'w') as f: f.write("content")
        with open(os.path.join(self.test_dir, "subdir", "deep.yml"), 'w') as f: f.write("content")
        
        # Invalid files (wrong extension, hidden)
        with open(os.path.join(self.test_dir, "notes.txt"), 'w') as f: f.write("content")
        with open(os.path.join(self.test_dir, ".hidden.yaml"), 'w') as f: f.write("content")

        paths = config_manager.get_all_config_paths()

        # Check results
        self.assertIn("config.yaml", paths)
        self.assertIn(os.path.join("subdir", "deep.yml"), paths)
        self.assertNotIn("notes.txt", paths)
        self.assertNotIn(".hidden.yaml", paths)

    @patch('services.config_manager.ui_state_manager.load_all_data')
    def test_get_all_config_paths_adapter_logic_disabled(self, mock_load_data):
        """Test that routing_configs are hidden when Adapter (BAP/BPP) is not selected."""
        # Setup: User selected only Gateway, NO bap/bpp
        mock_load_data.return_value = {'deploymentGoal': {'bap': False, 'bpp': False, 'gateway': True}}

        # Create routing config structure
        routing_dir = os.path.join(self.test_dir, "routing_configs")
        os.makedirs(routing_dir)
        with open(os.path.join(routing_dir, "route.yaml"), 'w') as f: f.write("content")
        
        # Create standard config
        with open(os.path.join(self.test_dir, "main.yaml"), 'w') as f: f.write("content")

        paths = config_manager.get_all_config_paths()

        self.assertIn("main.yaml", paths)
        # Should be excluded because bap/bpp are false
        self.assertNotIn(os.path.join("routing_configs", "route.yaml"), paths)

    @patch('services.config_manager.ui_state_manager.load_all_data')
    def test_get_all_config_paths_adapter_logic_enabled(self, mock_load_data):
        """Test that routing_configs are SHOWN when Adapter is selected."""
        # Setup: User selected BAP
        mock_load_data.return_value = {'deploymentGoal': {'bap': True, 'bpp': False}}

        # Create routing config structure
        routing_dir = os.path.join(self.test_dir, "routing_configs")
        os.makedirs(routing_dir)
        with open(os.path.join(routing_dir, "route.yaml"), 'w') as f: f.write("content")

        paths = config_manager.get_all_config_paths()

        # Should be included
        self.assertIn(os.path.join("routing_configs", "route.yaml"), paths)

    @patch('services.config_manager.ui_state_manager.load_all_data')
    def test_get_all_config_paths_ui_error_fail_open(self, mock_load_data):
        """Test that if UI state fails to load, we default to showing everything."""
        mock_load_data.side_effect = Exception("DB Error")

        routing_dir = os.path.join(self.test_dir, "routing_configs")
        os.makedirs(routing_dir)
        with open(os.path.join(routing_dir, "route.yaml"), 'w') as f: f.write("content")

        paths = config_manager.get_all_config_paths()

        # Should be included because of fail-open logic in exception handler
        self.assertIn(os.path.join("routing_configs", "route.yaml"), paths)
        self.mock_logger.warning.assert_called()

    def test_get_all_config_paths_no_dir(self):
        """Test behavior when artifacts directory doesn't exist."""
        # Remove the setup dir to simulate missing dir
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)
        
        paths = config_manager.get_all_config_paths()
        
        self.assertEqual(paths, [])
        self.mock_logger.warning.assert_called_with(f"Artifacts directory {self.test_dir} does not exist.")

    @patch('services.config_manager.utils.read_file_content')
    def test_get_config_content_success(self, mock_read):
        """Test reading a valid file."""
        mock_read.return_value = "file_content"
        filename = "valid_config.yaml"
        
        content = config_manager.get_config_content(filename)
        
        expected_path = os.path.abspath(os.path.join(self.test_dir, filename))
        mock_read.assert_called_once_with(expected_path)
        self.assertEqual(content, "file_content")

    def test_get_config_content_path_traversal(self):
        """Test that path traversal attempts raise ValueError."""
        # Attempt to access a file outside the artifacts dir
        malicious_path = "../../../etc/passwd"
        
        with self.assertRaises(ValueError) as context:
            config_manager.get_config_content(malicious_path)
        
        self.assertIn("Invalid path", str(context.exception))
        self.assertIn("Access denied", str(context.exception))

    @patch('services.config_manager.utils.write_file_content')
    def test_update_config_content_success(self, mock_write):
        """Test writing a valid file."""
        filename = "subdir/config.yaml"
        new_content = "new_data: 123"
        
        # Ensure the subdir exists in our real temp dir so abspath resolution works as expected logic-wise
        # though strictly _validate_path works on strings, making the dir makes the test cleaner
        os.makedirs(os.path.join(self.test_dir, "subdir"), exist_ok=True)

        config_manager.update_config_content(filename, new_content)
        
        expected_path = os.path.abspath(os.path.join(self.test_dir, filename))
        mock_write.assert_called_once_with(expected_path, new_content)

    def test_update_config_content_path_traversal(self):
        """Test that path traversal attempts during update raise ValueError."""
        malicious_path = "../hack.sh"
        
        with self.assertRaises(ValueError):
            config_manager.update_config_content(malicious_path, "malicious code")