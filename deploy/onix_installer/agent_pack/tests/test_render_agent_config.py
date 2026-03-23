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
import unittest
from unittest import mock

from typing import Any
from google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack import render_agent_config

_RENDER_AGENT_CONFIG_PATH = "google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack.render_agent_config"


class TestRenderAgentConfig(unittest.TestCase):
  """Tests for the render_agent_config module."""

  def setUp(self) -> None:
    super().setUp()
    self.installer_root = "/mock/installer"
    self.env_file_path = os.path.join(
        self.installer_root, "agent_pack", "agent_config.env"
    )
    self.state_file_path = os.path.join(
        self.installer_root, "backend", "installer_kit", "installer_state.json"
    )
    self.template_dir = os.path.join(
        self.installer_root,
        "backend",
        "installer_kit",
        "templates",
        "tf_configs",
    )
    self.output_file = os.path.join(
        self.installer_root,
        "backend",
        "installer_kit",
        "terraform",
        "generated-terraform.tfvars",
    )

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_success_no_state(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_env: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests successful config rendering when no existing state is present."""
    mock_exists.side_effect = lambda p: p in [
        os.path.join(self.template_dir, "main_tfvars.tfvars.j2"),
    ]

    mock_dotenv.return_value = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "APP_NAME": "a",
        "SESSION_DB_TYPE": "database",
    }
    
    # Mock for template rendering
    mock_template = mock.Mock()
    mock_template.render.return_value = "rendered content"
    mock_env.return_value.get_template.return_value = mock_template

    render_agent_config.render_config(self.installer_root)

    # Check file operations: write state, then write output
    mock_open.assert_any_call(self.state_file_path, "w")
    mock_open.assert_any_call(self.output_file, "w")

    expected_context = {
        "project_id": "p",
        "region": "r",
        "app_name": "a",
        "deployment_size": "small",
        "enable_onix": False,
        "enable_agent": True,
        "provision_adapter_infra": False,
        "provision_gateway_infra": False,
        "provision_registry_infra": False,
        "enable_cloud_armor": False,
        "rate_limit_count": 100,
        "provision_agent_db": True,
        "agent_engine_id": "",
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_with_existing_state_success(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_env: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests successful config rendering when valid existing state exists."""
    mock_exists.return_value = True

    mock_dotenv.return_value = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "APP_NAME": "a",
        "SESSION_DB_TYPE": "none",
    }
    state_content = json.dumps({
        "project_id": "p",
        "region": "r",
        "app_name": "a",
        "enable_onix": True,
        "deployment_size": "medium",
        "provision_adapter_infra": True,
    })

    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,  # read state
        mock.mock_open().return_value,  # write updated state
        mock.mock_open().return_value,  # write output
    ]

    mock_template = mock.Mock()
    mock_template.render.return_value = "rendered content"
    mock_env.return_value.get_template.return_value = mock_template

    render_agent_config.render_config(self.installer_root)

    expected_context = {
        "project_id": "p",
        "region": "r",
        "app_name": "a",
        "deployment_size": "medium",
        "enable_onix": True,
        "enable_agent": True,
        "provision_adapter_infra": True,
        "provision_gateway_infra": False,
        "provision_registry_infra": False,
        "enable_cloud_armor": False,
        "rate_limit_count": 100,
        "provision_agent_db": False,
        "agent_engine_id": "",
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_conflict_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in project_id between env and state causes exit."""
    mock_exists.return_value = True

    mock_dotenv.return_value = {
        "PROJECT_ID": "new-p",
        "REGION": "r",
        "APP_NAME": "a",
    }
    state_content = json.dumps(
        {"project_id": "old-p", "region": "r", "app_name": "a"}
    )

    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,  # read state
    ]

    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'project_id'.")

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_region_conflict(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in region between env and state causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "PROJECT_ID": "p",
        "REGION": "new-r",
        "APP_NAME": "a",
    }
    state_content = json.dumps(
        {"project_id": "p", "region": "old-r", "app_name": "a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,
    ]
    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'region'.")

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_app_name_conflict(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in app_name between env and state causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "APP_NAME": "new-a",
    }
    state_content = json.dumps(
        {"project_id": "p", "region": "r", "app_name": "old-a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,
    ]
    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'app_name'.")

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_state_read_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that an error during state file reading causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"PROJECT_ID": "p", "REGION": "r", "APP_NAME": "a"}

    mock_open.side_effect = Exception("Read error")
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  def test_render_config_template_missing(
      self, mock_exists: mock.Mock, mock_dotenv: mock.Mock
  ) -> None:
    """Tests that a missing Jinja2 template causes exit."""
    # Env file exists, but template doesn't
    mock_exists.side_effect = lambda p: p == self.env_file_path
    mock_dotenv.return_value = {"PROJECT_ID": "p", "REGION": "r", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_write_state_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      _: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that an error during state file writing causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"PROJECT_ID": "p", "REGION": "r", "APP_NAME": "a"}

    mock_open.side_effect = [
        mock.mock_open(read_data="{}").return_value,  # read empty state
        Exception("Write error"),  # write state fails
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.render_config")
  @mock.patch("os.path.abspath")
  @mock.patch("os.path.dirname")
  def test_main(
      self,
      mock_dirname: mock.Mock,
      mock_abspath: mock.Mock,
      mock_render: mock.Mock,
  ) -> None:
    """Tests that main() correctly resolves paths and calls render_config."""
    mock_dirname.return_value = "/mock/dir"
    mock_abspath.side_effect = ["/mock/dir/script.py", "/mock/installer"]
    render_agent_config.main()
    mock_render.assert_called_once_with("/mock/installer")

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_missing_project_id(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing PROJECT_ID in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"REGION": "r", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_missing_region(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing REGION in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"PROJECT_ID": "p", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_missing_app_name(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing APP_NAME in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"PROJECT_ID": "p", "REGION": "r"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.dotenv.dotenv_values")
  def test_render_config_env_file_missing(self, mock_dotenv: mock.Mock) -> None:
    """Tests that a missing configuration file causes exit."""
    mock_dotenv.side_effect = FileNotFoundError()
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  def test_validate_config_missing_project_id(self) -> None:
    env_vars: dict[str, str] = {"REGION": "r", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.validate_config(env_vars, {})

  def test_validate_config_missing_region(self) -> None:
    env_vars: dict[str, str] = {"PROJECT_ID": "p", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.validate_config(env_vars, {})

  def test_validate_config_missing_app_name(self) -> None:
    env_vars: dict[str, str] = {"PROJECT_ID": "p", "REGION": "r"}
    with self.assertRaises(SystemExit):
      render_agent_config.validate_config(env_vars, {})

  def test_validate_config_project_id_conflict(self) -> None:
    env_vars = {"PROJECT_ID": "new-p", "REGION": "r", "APP_NAME": "a"}
    state_data = {"project_id": "old-p", "region": "r", "app_name": "a"}
    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'project_id'.")

  def test_validate_config_region_conflict(self) -> None:
    env_vars = {"PROJECT_ID": "p", "REGION": "new-r", "APP_NAME": "a"}
    state_data = {"project_id": "p", "region": "old-r", "app_name": "a"}
    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'region'.")

  def test_validate_config_app_name_conflict(self) -> None:
    env_vars = {"PROJECT_ID": "p", "REGION": "r", "APP_NAME": "new-a"}
    state_data = {"project_id": "p", "region": "r", "app_name": "old-a"}
    with mock.patch("builtins.print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'app_name'.")

  def test_validate_config_success_with_state(self) -> None:
    env_vars: dict[str, str] = {"PROJECT_ID": "p", "REGION": "r", "APP_NAME": "a"}
    state_data: dict[str, Any] = {"project_id": "p", "region": "r", "app_name": "a"}
    # No exception means success
    render_agent_config.validate_config(env_vars, state_data)


if __name__ == "__main__":
  unittest.main()
