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

import os
import unittest
from unittest.mock import patch, MagicMock

from services import config_manager
from core.constants import GENERATED_CONFIGS_DIR

class TestConfigManager(unittest.TestCase):

    @patch('services.config_manager.app_config')
    def test_generate_initial_configs(self, mock_app_config):
        mock_request = MagicMock()
        config_manager.generate_initial_configs(mock_request)
        mock_app_config.generate_app_configs.assert_called_once_with(mock_request)

    @patch('os.path.exists')
    def test_get_all_config_paths_dir_not_exists(self, mock_exists):
        mock_exists.return_value = False
        paths = config_manager.get_all_config_paths()
        self.assertEqual(paths, [])

    @patch('os.path.exists')
    @patch('os.walk')
    @patch('services.ui_state_manager.load_all_data')
    def test_get_all_config_paths_adapter_enabled(self, mock_load_data, mock_walk, mock_exists):
        mock_exists.return_value = True
        mock_load_data.return_value = {'deploymentGoal': {'bap': True}}
        
        # Setup mock directory structure
        mock_walk.return_value = [
            (GENERATED_CONFIGS_DIR, [], ['config1.yaml', 'config2.yml', '.hidden.yaml', 'invalid.txt', 'routing_configs.yaml']),
            (os.path.join(GENERATED_CONFIGS_DIR, 'routing_configs'), [], ['route1.yaml']),
        ]
        
        paths = config_manager.get_all_config_paths()
        
        expected_paths = [
            'config1.yaml',
            'config2.yml',
            'routing_configs.yaml',
            os.path.join('routing_configs', 'route1.yaml')
        ]
        self.assertEqual(paths, sorted(expected_paths))

    @patch('os.path.exists')
    @patch('os.walk')
    @patch('services.ui_state_manager.load_all_data')
    def test_get_all_config_paths_adapter_disabled(self, mock_load_data, mock_walk, mock_exists):
        mock_exists.return_value = True
        mock_load_data.return_value = {'deploymentGoal': {'bap': False, 'bpp': False}}
        
        mock_walk.return_value = [
            (GENERATED_CONFIGS_DIR, [], ['config1.yaml', 'routing_configs_base.yaml']),
            (os.path.join(GENERATED_CONFIGS_DIR, 'routing_configs'), [], ['route1.yaml']),
        ]
        
        paths = config_manager.get_all_config_paths()
        
        expected_paths = ['config1.yaml']
        self.assertEqual(paths, sorted(expected_paths))

    @patch('services.config_manager._validate_path')
    @patch('services.config_manager.utils')
    def test_get_config_content(self, mock_utils, mock_validate_path):
        mock_validate_path.return_value = '/safe/path.yaml'
        mock_utils.read_file_content.return_value = 'content'
        
        content = config_manager.get_config_content('path.yaml')
        
        mock_validate_path.assert_called_once_with('path.yaml')
        mock_utils.read_file_content.assert_called_once_with('/safe/path.yaml')
        self.assertEqual(content, 'content')

    @patch('services.config_manager._validate_path')
    @patch('services.config_manager.utils')
    def test_update_config_content(self, mock_utils, mock_validate_path):
        mock_validate_path.return_value = '/safe/path.yaml'
        
        config_manager.update_config_content('path.yaml', 'new content')
        
        mock_validate_path.assert_called_once_with('path.yaml')
        mock_utils.write_file_content.assert_called_once_with('/safe/path.yaml', 'new content')

    def test_validate_path_valid(self):
        valid_path = 'valid_file.yaml'
        expected_full_path = os.path.abspath(os.path.join(GENERATED_CONFIGS_DIR, valid_path))
        
        result = config_manager._validate_path(valid_path)
        self.assertEqual(result, expected_full_path)

    def test_validate_path_invalid_traversal(self):
        invalid_path = '../outside.yaml'
        with self.assertRaises(ValueError) as context:
            config_manager._validate_path(invalid_path)
        
        self.assertIn("Access denied", str(context.exception))

if __name__ == '__main__':
    unittest.main()
