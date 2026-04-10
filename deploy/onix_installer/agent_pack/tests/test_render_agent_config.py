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

from __future__ import annotations

import builtins
import json
import os
import unittest
from typing import Any
from unittest import mock

from google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack import render_agent_config


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

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(render_agent_config, "Environment")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
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
        "GOOGLE_CLOUD_PROJECT": "p",
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
        "datastore_imports": {},
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(render_agent_config, "Environment")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
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
        "GOOGLE_CLOUD_PROJECT": "p",
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
        "datastore_imports": {},
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_conflict_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in project_id between env and state causes exit."""
    mock_exists.return_value = True

    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "new-p",
        "REGION": "r",
        "APP_NAME": "a",
    }
    state_content = json.dumps(
        {"project_id": "old-p", "region": "r", "app_name": "a"}
    )

    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,  # read state
    ]

    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'project_id'.")

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_region_conflict(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in region between env and state causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "p",
        "REGION": "new-r",
        "APP_NAME": "a",
    }
    state_content = json.dumps(
        {"project_id": "p", "region": "old-r", "app_name": "a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,
    ]
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'region'.")

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_app_name_conflict(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that a conflict in app_name between env and state causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "p",
        "REGION": "r",
        "APP_NAME": "new-a",
    }
    state_content = json.dumps(
        {"project_id": "p", "region": "r", "app_name": "old-a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=state_content).return_value,
    ]
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_any_call("❌ Error: Conflict in 'app_name'.")

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_state_read_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that an error during state file reading causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r", "APP_NAME": "a"}

    mock_open.side_effect = Exception("Read error")
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  def test_render_config_template_missing(
      self, mock_exists: mock.Mock, mock_dotenv: mock.Mock
  ) -> None:
    """Tests that a missing Jinja2 template causes exit."""
    # Env file exists, but template doesn't
    mock_exists.side_effect = lambda p: p == self.env_file_path
    mock_dotenv.return_value = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(render_agent_config, "Environment")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_write_state_error(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      _: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that an error during state file writing causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r", "APP_NAME": "a"}

    mock_open.side_effect = [
        mock.mock_open(read_data="{}").return_value,  # read empty state
        Exception("Write error"),  # write state fails
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config, "render_config")
  @mock.patch.object(os.path, "abspath")
  @mock.patch.object(os.path, "dirname")
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

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_missing_project_id(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing GOOGLE_CLOUD_PROJECT in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"REGION": "r", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_missing_region(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing REGION in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"GOOGLE_CLOUD_PROJECT": "p", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_missing_app_name(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that missing APP_NAME in env causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r"}
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
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
    env_vars: dict[str, str] = {"GOOGLE_CLOUD_PROJECT": "p", "APP_NAME": "a"}
    with self.assertRaises(SystemExit):
      render_agent_config.validate_config(env_vars, {})

  def test_validate_config_missing_app_name(self) -> None:
    env_vars: dict[str, str] = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r"}
    with self.assertRaises(SystemExit):
      render_agent_config.validate_config(env_vars, {})

  def test_validate_config_project_id_conflict(self) -> None:
    env_vars = {"GOOGLE_CLOUD_PROJECT": "new-p", "REGION": "r", "APP_NAME": "a"}
    state_data = {"project_id": "old-p", "region": "r", "app_name": "a"}
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'project_id'.")

  def test_validate_config_region_conflict(self) -> None:
    env_vars = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "new-r", "APP_NAME": "a"}
    state_data = {"project_id": "p", "region": "old-r", "app_name": "a"}
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'region'.")

  def test_validate_config_app_name_conflict(self) -> None:
    env_vars = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r", "APP_NAME": "new-a"}
    state_data = {"project_id": "p", "region": "r", "app_name": "old-a"}
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.validate_config(env_vars, state_data)
      mock_print.assert_any_call("❌ Error: Conflict in 'app_name'.")

  def test_validate_config_success_with_state(self) -> None:
    env_vars: dict[str, str] = {"GOOGLE_CLOUD_PROJECT": "p", "REGION": "r", "APP_NAME": "a"}
    state_data: dict[str, Any] = {"project_id": "p", "region": "r", "app_name": "a"}
    # No exception means success
    render_agent_config.validate_config(env_vars, state_data)

  def test_check_required_datastores_success(self) -> None:
    env_vars = {
        "SUB_AGENTS_OF_ROOT_AGENT": "agri.biochar_advice, another_agent",
        "SUB_AGENTS_AS_TOOLS_OF_ROOT_AGENT": "",
    }
    datastore_imports = {"agri.biochar_advice": "gs://some-bucket"}
    # Should not raise any exception
    render_agent_config.check_required_datastores(env_vars, datastore_imports)

  def test_check_required_datastores_missing(self) -> None:
    env_vars = {
        "SUB_AGENTS_OF_ROOT_AGENT": "agri.biochar_advice, another_agent",
        "SUB_AGENTS_AS_TOOLS_OF_ROOT_AGENT": "",
    }
    datastore_imports = {"another_agent": "gs://some-bucket"} # Missing agri.biochar_advice
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.check_required_datastores(env_vars, datastore_imports)
      mock_print.assert_any_call("❌ Error: Missing required Datastore mappings in DATASTORE_IMPORTS:")
      mock_print.assert_any_call("   -> Agent 'agri.biochar_advice' requires datastore key 'agri.biochar_advice'")

  def test_check_required_datastores_empty(self) -> None:
    env_vars = {
        "SUB_AGENTS_OF_ROOT_AGENT": "",
        "SUB_AGENTS_AS_TOOLS_OF_ROOT_AGENT": "",
    }
    datastore_imports = {}
    # Should not raise any exception when no agents are enabled
    render_agent_config.check_required_datastores(env_vars, datastore_imports)

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(render_agent_config, "Environment")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_invalid_datastore_imports_json(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_env: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that invalid JSON in DATASTORE_IMPORTS causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "p",
        "REGION": "r",
        "APP_NAME": "a",
        "DATASTORE_IMPORTS": "{invalid json}",
    }
    mock_open.side_effect = [
        mock.mock_open(read_data="{}").return_value,
    ]

    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_called_with("❌ Error: Invalid JSON in DATASTORE_IMPORTS: Expecting property name enclosed in double quotes: line 1 column 2 (char 1)")

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(render_agent_config, "Environment")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_datastore_imports_not_dict(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_env: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that non-dictionary JSON in DATASTORE_IMPORTS causes exit."""
    mock_exists.return_value = True
    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "p",
        "REGION": "r",
        "APP_NAME": "a",
        "DATASTORE_IMPORTS": '["not", "a", "dict"]',
    }
    mock_open.side_effect = [
        mock.mock_open(read_data="{}").return_value,
    ]

    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)
      mock_print.assert_called_with("❌ Error: DATASTORE_IMPORTS must be a JSON dictionary.")

  @mock.patch.object(render_agent_config.dotenv, "dotenv_values")
  @mock.patch.object(os.path, "exists")
  @mock.patch.object(builtins, "open", new_callable=mock.mock_open)
  def test_render_config_fails_when_missing_required_datastores(
      self,
      mock_open: mock.Mock,
      mock_exists: mock.Mock,
      mock_dotenv: mock.Mock,
  ) -> None:
    """Tests that render_config fails when a required agent datastore is missing."""
    mock_exists.return_value = True

    # We enable 'agri.biochar_advice' which requires a datastore, but provide None
    mock_dotenv.return_value = {
        "GOOGLE_CLOUD_PROJECT": "p",
        "REGION": "r",
        "APP_NAME": "a",
        "SUB_AGENTS_OF_ROOT_AGENT": "agri.biochar_advice",
        "DATASTORE_IMPORTS": "{}",
    }

    mock_open.side_effect = [
        mock.mock_open(read_data="{}").return_value,  # state read
    ]

    with mock.patch.object(builtins, "print") as mock_print:
      # We expect the whole render_config flow to exit
      with self.assertRaises(SystemExit):
        render_agent_config.render_config(self.installer_root)

      # Verify the specific inner validation error was properly bubbled up
      mock_print.assert_any_call(
          "❌ Error: Missing required Datastore mappings in DATASTORE_IMPORTS:"
      )
      mock_print.assert_any_call(
          "   -> Agent 'agri.biochar_advice' requires datastore key"
          " 'agri.biochar_advice'"
      )


if __name__ == "__main__":
  unittest.main()
