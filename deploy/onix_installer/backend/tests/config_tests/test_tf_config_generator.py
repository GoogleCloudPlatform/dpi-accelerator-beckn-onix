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

import unittest
import os
import logging
from unittest.mock import patch

from absl.testing import absltest as googletest
from core import models
from config import tf_config_generator

class TestTerraformConfigGenerator(unittest.TestCase):

    def setUp(self):
        self.patcher_tf_dir = patch('config.tf_config_generator.TERRAFORM_DIRECTORY', '/mock/tf_output_dir')
        self.patcher_template_dir = patch('config.tf_config_generator.TEMPLATE_DIRECTORY', '/mock/templates_base')
        self.patcher_installer_kit_path = patch('config.tf_config_generator.INSTALLER_KIT_PATH', '/mock/installer_kit')

        self.mock_tf_dir = self.patcher_tf_dir.start()
        self.mock_template_dir = self.patcher_template_dir.start()
        self.mock_installer_kit_path = self.patcher_installer_kit_path.start()

        # Reset logger handlers
        self.original_handlers = tf_config_generator.logger.handlers[:]
        tf_config_generator.logger.handlers = []
        tf_config_generator.logger.propagate = False
        tf_config_generator.logger.setLevel(logging.NOTSET)

    def tearDown(self):
        # Stop all patchers
        self.patcher_tf_dir.stop()
        self.patcher_template_dir.stop()
        self.patcher_installer_kit_path.stop()

        # Restore logger state
        tf_config_generator.logger.handlers = self.original_handlers
        tf_config_generator.logger.propagate = True

    def test_validate_immutable_triplet_success(self):
        state_data = {
            "project_id": "test-proj",
            "region": "us-west1",
            "app_name": "my-app"
        }

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        # Should not raise exception
        tf_config_generator.validate_immutable_triplet(state_data, req)

        # Test empty state also passes
        tf_config_generator.validate_immutable_triplet({}, req)

    def test_validate_immutable_triplet_project_mismatch(self):
        state_data = {
            "project_id": "original-proj",
            "region": "us-west1",
            "app_name": "my-app"
        }

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(ValueError, "Cannot change project_id from 'original-proj' to 'test-proj'"):
            tf_config_generator.validate_immutable_triplet(state_data, req)

    def test_validate_immutable_triplet_region_mismatch(self):
        state_data = {
            "project_id": "test-proj",
            "region": "us-east1",
            "app_name": "my-app"
        }

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(ValueError, "Cannot change region from 'us-east1' to 'us-west1'"):
            tf_config_generator.validate_immutable_triplet(state_data, req)

    def test_validate_immutable_triplet_app_name_mismatch(self):
        state_data = {
            "project_id": "test-proj",
            "region": "us-west1",
            "app_name": "original-app"
        }

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(ValueError, "Cannot change app_name from 'original-app' to 'my-app'"):
            tf_config_generator.validate_immutable_triplet(state_data, req)

    @patch('config.tf_config_generator.write_file_content')
    @patch('config.tf_config_generator.render_jinja_template')
    @patch('config.tf_config_generator.os.path.exists', return_value=False)
    @patch('builtins.open', create=True)
    def test_generate_config_success_small(self, mock_open, mock_exists, mock_render_template, mock_write_file):
        """
        Test successful generation of Terraform config for 'small' deployment type (no existing state).
        """
        # Mock open for write (success)
        mock_write_handle = unittest.mock.mock_open().return_value
        mock_open.return_value = mock_write_handle

        mock_render_template.return_value = "main_config_content"

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={
                "bap": True,
                "gateway": False,
                "registry": True
            }
        )

        tf_config_generator.generate_config(req)

        # Expected Jinja2 context
        expected_jinja_context = {
            "project_id": "test-proj",
            "region": "us-west1",
            "app_name": "my-app",
            "deployment_size": "small",
            "provision_adapter_infra": True, # BAP is True
            "provision_gateway_infra": False, # GATEWAY is False
            "provision_registry_infra": True, # REGISTRY is True
            "enable_cloud_armor": False,
            "rate_limit_count": 100,
            "enable_onix": True,
            "enable_agent": False,
            "provision_agent_db": False,
            "agent_engine_id": "",
            "datastore_imports": {},
        }

        # Expected paths for mocking os.path.join (relative to patched constants)
        expected_template_source_dir = os.path.join(self.mock_template_dir, "tf_configs")
        expected_output_tfvars_path = os.path.join(self.mock_tf_dir, "generated-terraform.tfvars")

        # Assert render_jinja_template was called correctly
        mock_render_template.assert_called_once_with(
            template_dir=expected_template_source_dir,
            template_name="main_tfvars.tfvars.j2",
            context=expected_jinja_context
        )

        # Assert write_file_content was called with merged content
        mock_write_file.assert_called_once_with(expected_output_tfvars_path, "main_config_content")

        # Verify open calls
        expected_state_file = os.path.join(self.mock_installer_kit_path, "installer_state.json")
        # In a fresh deployment (exists=False), we only open for write
        mock_open.assert_called_once_with(expected_state_file, "w")

    @patch('config.tf_config_generator.write_file_content')
    @patch('config.tf_config_generator.render_jinja_template')
    @patch('config.tf_config_generator.os.path.exists', return_value=False)
    @patch('builtins.open', create=True)
    def test_generate_config_success_medium_no_bap_bpp(self, mock_open, mock_exists, mock_render_template, mock_write_file):
        """
        Test successful generation for 'medium' type with no BAP/BPP components (no existing state).
        """
        # Mock open for write (success)
        mock_write_handle = unittest.mock.mock_open().return_value
        mock_open.return_value = mock_write_handle

        mock_render_template.return_value = "main_config_content_medium"

        req = models.InfraDeploymentRequest(
            project_id="test-medium",
            region="us-east1",
            app_name="my-app-medium",
            type=models.DeploymentType.MEDIUM,
            components={
                "bap": False,
                "bpp": False,
                "gateway": True,
                "registry": False
            }
        )
        tf_config_generator.generate_config(req)

        expected_jinja_context = {
            "project_id": "test-medium",
            "region": "us-east1",
            "app_name": "my-app-medium",
            "deployment_size": "medium",
            "provision_adapter_infra": False, # Both BAP and BPP are False
            "provision_gateway_infra": True,
            "provision_registry_infra": False,
            "enable_cloud_armor": False,
            "rate_limit_count": 100,
            "enable_onix": True,
            "enable_agent": False,
            "provision_agent_db": False,
            "agent_engine_id": "",
            "datastore_imports": {},
        }

        mock_render_template.assert_called_once_with(
            template_dir=os.path.join(self.mock_template_dir, "tf_configs"),
            template_name="main_tfvars.tfvars.j2",
            context=expected_jinja_context
        )
        mock_write_file.assert_called_once_with(
            os.path.join(self.mock_tf_dir, "generated-terraform.tfvars"),
            "main_config_content_medium"
        )
        # Verify open calls
        expected_state_file = os.path.join(self.mock_installer_kit_path, "installer_state.json")
        # In a fresh deployment (exists=False), we only open for write
        mock_open.assert_called_once_with(expected_state_file, "w")

    @patch('config.tf_config_generator.write_file_content', side_effect=IOError("Disk full"))
    @patch('config.tf_config_generator.render_jinja_template', return_value="main_config_content")
    @patch('config.tf_config_generator.os.path.exists', return_value=False)
    @patch('builtins.open', create=True)
    @patch('config.tf_config_generator.logger')
    def test_generate_config_write_error(self, mock_logger, mock_open, mock_exists, mock_render_template, mock_write_file):
        """
        Test error handling when writing the merged tfvars file fails.
        """
        # Mock open for write (success)
        mock_write_handle = unittest.mock.mock_open().return_value
        mock_open.return_value = mock_write_handle

        req = models.InfraDeploymentRequest(
            project_id="test-proj-write",
            region="us-west1",
            app_name="my-app-write",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(IOError, "Disk full"):
            tf_config_generator.generate_config(req)

        mock_render_template.assert_called_once()
        mock_write_file.assert_called_once()
        # Adjusted assert to be more specific to avoid failing if other errors are generated
        mock_logger.error.assert_called_with(
            "Error writing tfvars file '%s': %s. Aborting configuration generation.",
            os.path.join('/mock/tf_output_dir', 'generated-terraform.tfvars'),
            unittest.mock.ANY
        )

    @patch('config.tf_config_generator.render_jinja_template', side_effect=Exception("Jinja Error"))
    @patch('config.tf_config_generator.logger')
    @patch('config.tf_config_generator.write_file_content')
    @patch('config.tf_config_generator.os.path.exists', return_value=False)
    @patch('builtins.open', create=True)
    def test_generate_config_template_render_error(self, mock_open, mock_exists, mock_write_file, mock_logger, mock_render_template):
        """
        Test error handling when rendering the main template fails.
        """
        # Mock open for write (success)
        mock_write_handle = unittest.mock.mock_open().return_value
        mock_open.return_value = mock_write_handle

        req = models.InfraDeploymentRequest(
            project_id="test-proj-render",
            region="us-west1",
            app_name="my-app-render",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(Exception, "Jinja Error"):
            tf_config_generator.generate_config(req)

        mock_render_template.assert_called_once()
        mock_logger.error.assert_called_with("Failed to process main Terraform configuration template: %s", unittest.mock.ANY)
        self.assertFalse(mock_write_file.called)

    @patch('config.tf_config_generator.write_file_content')
    @patch('config.tf_config_generator.render_jinja_template')
    @patch('config.tf_config_generator.os.path.exists', return_value=True)
    @patch('builtins.open', create=True)
    def test_generate_config_with_existing_valid_state(self, mock_open, mock_exists, mock_render_template, mock_write_file):
        """
        Test generate_config when a valid installer_state.json already exists.
        """
        # Mock reading state
        read_state = '{"project_id": "test-proj", "region": "us-west1", "app_name": "my-app", "enable_agent": true, "agent_engine_id": "test-id"}'
        
        mock_read_handle = unittest.mock.mock_open(read_data=read_state).return_value
        mock_write_handle = unittest.mock.mock_open().return_value

        def open_side_effect(file, mode='r', **kwargs):
            if mode == 'r':
                return mock_read_handle
            return mock_write_handle

        mock_open.side_effect = open_side_effect

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        tf_config_generator.generate_config(req)

        # Assert `enable_agent` was correctly carried over from the mock read_data
        context_args = mock_render_template.call_args[1]['context']
        self.assertEqual(context_args['enable_agent'], True)
        self.assertEqual(context_args['provision_agent_db'], True)
        self.assertEqual(context_args['agent_engine_id'], "test-id")

    @patch('config.tf_config_generator.os.path.exists', return_value=True)
    @patch('builtins.open', side_effect=Exception("Read Permission Denied"))
    @patch('config.tf_config_generator.logger')
    def test_generate_config_state_file_read_error(self, mock_logger, mock_open, mock_exists):
        """
        Test that an exception is raised when reading the state file fails.
        """
        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(Exception, "Read Permission Denied"):
            tf_config_generator.generate_config(req)

        mock_logger.error.assert_any_call("Error reading %s: %s", '/mock/installer_kit/installer_state.json', unittest.mock.ANY)

    @patch('config.tf_config_generator.os.path.exists', return_value=False)
    @patch('builtins.open', create=True)
    @patch('config.tf_config_generator.logger')
    def test_generate_config_state_file_write_error(self, mock_logger, mock_open, mock_exists):
        """
        Test that an exception is raised when writing the state file fails.
        """
        mock_open.side_effect = Exception("Write Permission Denied")

        req = models.InfraDeploymentRequest(
            project_id="test-proj",
            region="us-west1",
            app_name="my-app",
            type=models.DeploymentType.SMALL,
            components={}
        )

        with self.assertRaisesRegex(Exception, "Write Permission Denied"):
            tf_config_generator.generate_config(req)

        mock_logger.error.assert_any_call("Error writing to %s: %s", '/mock/installer_kit/installer_state.json', unittest.mock.ANY)


if __name__ == '__main__':
    googletest.main()
