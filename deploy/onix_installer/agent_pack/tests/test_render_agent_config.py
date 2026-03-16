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

from google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack import render_agent_config

_RENDER_AGENT_CONFIG_PATH = "google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack.render_agent_config"


class TestRenderAgentConfig(unittest.TestCase):
  """Tests for the render_agent_config module."""

  def setUp(self):
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

  @mock.patch("os.path.exists")
  def test_parse_env_file_success(self, mock_exists):
    mock_exists.return_value = True
    env_content = """
PROJECT_ID=test-project
REGION="us-central1"
APP_NAME = 'demo'
# Comment
AGENT_IMAGE_URL="gcr.io/test/image"

    """
    with mock.patch("builtins.open", mock.mock_open(read_data=env_content)):
      env_vars = render_agent_config.parse_env_file("fake.env")
      self.assertEqual(env_vars["PROJECT_ID"], "test-project")
      self.assertEqual(env_vars["REGION"], "us-central1")
      self.assertEqual(env_vars["APP_NAME"], "demo")
      self.assertEqual(env_vars["AGENT_IMAGE_URL"], "gcr.io/test/image")

  @mock.patch("os.path.exists")
  def test_parse_env_file_not_found(self, mock_exists):
    mock_exists.return_value = False
    with self.assertRaises(SystemExit):
      render_agent_config.parse_env_file("fake.env")

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_success_no_state(
      self, mock_open, mock_exists, mock_env
  ):
    """Tests successful config rendering when no existing state is present."""
    mock_exists.side_effect = lambda p: p in [
        self.env_file_path,
        os.path.join(self.template_dir, "main_tfvars.tfvars.j2"),
    ]

    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"
    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,  # read env
        mock.mock_open().return_value,  # write state
        mock.mock_open().return_value,  # write output
    ]

    mock_template = mock.Mock()
    mock_template.render.return_value = "rendered content"
    mock_env.return_value.get_template.return_value = mock_template

    render_agent_config.render_config(self.installer_root)

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
        "allowed_regions": [],
        "rate_limit_count": 100,
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_with_existing_state_success(
      self, mock_open, mock_exists, mock_env
  ):
    """Tests successful config rendering when valid existing state exists."""
    mock_exists.return_value = True

    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"
    state_content = json.dumps({
        "project_id": "p",
        "region": "r",
        "app_name": "a",
        "enable_onix": True,
        "deployment_size": "medium",
    })

    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,  # read env
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
        "provision_adapter_infra": False,
        "provision_gateway_infra": False,
        "provision_registry_infra": False,
        "enable_cloud_armor": False,
        "allowed_regions": [],
        "rate_limit_count": 100,
    }
    mock_template.render.assert_called_once_with(expected_context)

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_conflict_error(self, mock_open, mock_exists):
    """Tests that a conflict in project_id between env and state causes exit."""
    mock_exists.return_value = True

    env_content = "PROJECT_ID=new-p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"
    state_content = json.dumps(
        {"project_id": "old-p", "region": "r", "app_name": "a"}
    )

    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,  # read env
        mock.mock_open(read_data=state_content).return_value,  # read state
    ]

    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_region_conflict(self, mock_open, mock_exists):
    """Tests that a conflict in region between env and state causes exit."""
    mock_exists.return_value = True
    env_content = "PROJECT_ID=p\nREGION=new-r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"
    state_content = json.dumps(
        {"project_id": "p", "region": "old-r", "app_name": "a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,
        mock.mock_open(read_data=state_content).return_value,
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_app_name_conflict(self, mock_open, mock_exists):
    """Tests that a conflict in app_name between env and state causes exit."""
    mock_exists.return_value = True
    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=new-a\nAGENT_IMAGE_URL=i\n"
    state_content = json.dumps(
        {"project_id": "p", "region": "r", "app_name": "old-a"}
    )
    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,
        mock.mock_open(read_data=state_content).return_value,
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_state_read_error(self, mock_open, mock_exists):
    """Tests that an error during state file reading causes exit."""
    mock_exists.return_value = True
    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"

    # Use a dummy context manager for the second open call that raises an error
    class ErroringOpen:

      def __enter__(self):
        raise RuntimeError("Read error")

      def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,
        ErroringOpen(),
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_template_missing(self, mock_open, mock_exists):
    """Tests that a missing Jinja2 template causes exit."""
    mock_exists.side_effect = lambda p: p == self.env_file_path
    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"
    mock_open.return_value = mock.mock_open(read_data=env_content).return_value
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.Environment")
  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_write_state_error(self, mock_open, mock_exists, _):
    """Tests that an error during state file writing causes exit."""
    mock_exists.return_value = True
    env_content = "PROJECT_ID=p\nREGION=r\nAPP_NAME=a\nAGENT_IMAGE_URL=i\n"

    class ErroringOpen:

      def __enter__(self):
        raise RuntimeError("Write error")

      def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    mock_open.side_effect = [
        mock.mock_open(read_data=env_content).return_value,  # read env
        mock.mock_open(read_data="{}").return_value,  # read empty state
        ErroringOpen(),  # write state fails
    ]
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)

  @mock.patch(f"{_RENDER_AGENT_CONFIG_PATH}.render_config")
  @mock.patch("os.path.abspath")
  @mock.patch("os.path.dirname")
  def test_main(self, mock_dirname, mock_abspath, mock_render):
    """Tests that main() correctly resolves paths and calls render_config."""
    mock_dirname.return_value = "/mock/dir"
    mock_abspath.side_effect = ["/mock/dir/script.py", "/mock/installer"]
    render_agent_config.main()
    mock_render.assert_called_once_with("/mock/installer")

  @mock.patch("os.path.exists")
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_render_config_missing_vars(self, mock_open, mock_exists):
    """Tests that missing required env variables cause exit."""
    mock_exists.return_value = True
    env_content = "PROJECT_ID=p\nREGION=r\n"
    mock_open.return_value = mock.mock_open(read_data=env_content).return_value
    with self.assertRaises(SystemExit):
      render_agent_config.render_config(self.installer_root)


if __name__ == "__main__":
  unittest.main()
