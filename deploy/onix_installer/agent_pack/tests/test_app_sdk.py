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

import importlib
import json
import os
import sys
import unittest
from unittest import mock

# 1. Substitute modules in sys.modules for the initial import phase of app_sdk.py
# We use global mocks so they are captured by app_sdk's top-level scope.
mock_app_global = mock.Mock()
mock_memory_global = mock.Mock()
mock_vx_global = mock.Mock()
mock_agent_server_mode = mock.Mock()
mock_dotenv_global = mock.Mock()

sys.modules['dpi_agent_blueprint'] = mock.Mock()
sys.modules['dpi_agent_blueprint.agent_engine'] = mock.Mock(app=mock_app_global, memory=mock_memory_global)
sys.modules['google.cloud.aiplatform'] = mock.Mock(vertexai=mock_vx_global)
sys.modules['google.cloud.aiplatform.vertexai'] = mock_vx_global
sys.modules['google.cloud.aiplatform.vertexai._genai'] = mock.Mock()
sys.modules['google.cloud.aiplatform.vertexai._genai.types'] = mock.Mock(AgentServerMode=mock_agent_server_mode)
sys.modules['vertexai'] = mock_vx_global
sys.modules['vertexai._genai'] = mock.Mock()
sys.modules['vertexai._genai.types'] = mock.Mock(AgentServerMode=mock_agent_server_mode)
sys.modules['dotenv'] = mock_dotenv_global

# Patch os.path.exists and sys.exit for the initial import phase to prevent script exit
_real_exists = os.path.exists
def mocked_exists(path):
    if "dpi_agent_blueprint" in str(path) or "installer_state.json" in str(path):
        return True
    return _real_exists(path)

with mock.patch('os.path.exists', side_effect=mocked_exists), \
     mock.patch('sys.exit'), \
     mock.patch('builtins.print'):
    from google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack import app_sdk


class TestAppSdk(unittest.TestCase):

  def setUp(self):
    super().setUp()
    # Reset all global mocks to a clean state before each test
    for m in [mock_app_global, mock_memory_global, mock_vx_global, mock_dotenv_global]:
        m.reset_mock(side_effect=True, return_value=True)
    
    # Re-apply necessary default return values
    mock_dotenv_global.dotenv_values.return_value = {
        "PROJECT_ID": "p", "REGION": "r", "STAGING_BUCKET": "s", "MIN_INSTANCES": "1"
    }

    # Ensure app_sdk is using these global mocks
    app_sdk.installer_root = "/mock/installer"
    app_sdk.vertexai = mock_vx_global
    app_sdk.dotenv = mock_dotenv_global
    app_sdk.app = mock_app_global
    app_sdk.memory = mock_memory_global
    self.state_file_path = os.path.join(
        app_sdk.installer_root, "backend", "installer_kit", "installer_state.json"
    )

  @mock.patch("builtins.open", new_callable=mock.mock_open, read_data='{"project_id": "p"}')
  @mock.patch("os.path.exists", return_value=True)
  def test_update_installer_state_success(self, mock_exists, mock_open):
    app_sdk.update_installer_state("new-engine-id")
    # Verify both read and write calls
    mock_open.assert_any_call(self.state_file_path, "r")
    mock_open.assert_any_call(self.state_file_path, "w")
    handle = mock_open()
    written_data = "".join(call.args[0] for call in handle.write.call_args_list)
    self.assertIn('"agent_engine_id": "new-engine-id"', written_data)

  @mock.patch("builtins.print")
  @mock.patch("os.path.exists", return_value=False)
  def test_update_installer_state_not_found(self, mock_exists, mock_print):
    app_sdk.update_installer_state("new-engine-id")
    mock_print.assert_any_call(mock.ANY)

  def test_sys_path_contains_script_dir(self):
    # This verifies that sys.path.append(script_dir) was called.
    self.assertIn(app_sdk.script_dir, sys.path)

  @mock.patch("os.path.exists", side_effect=lambda p: "dpi_agent_blueprint" not in str(p))
  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_top_level_blueprint_not_found_exit(self, mock_exit, mock_print, mock_exists):
    # Trigger the reload to re-run the top-level repo check.
    with mock.patch("os.path.abspath", return_value="/mock/path"):
      try:
        importlib.reload(app_sdk)
      except (SystemExit, ImportError):
         pass
    mock_exit.assert_called_with(1)
    found = any("Error: Agent blueprint repository not found" in str(call.args[0]) for call in mock_print.call_args_list)
    self.assertTrue(found, "Blueprint not found error message not printed.")

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_import_blueprint_failure_exit(self, mock_exit, mock_print, mock_exists):
    # We mock the submodule to raise ImportError during a reload of app_sdk
    with mock.patch.dict(sys.modules, {'dpi_agent_blueprint.agent_engine': None}):
       with mock.patch("os.path.abspath", return_value="/mock/path"):
         try:
           importlib.reload(app_sdk)
         except (SystemExit, ImportError):
           pass
    mock_exit.assert_called_with(1)
    found = any("Error: Could not import dpi_agent_blueprint" in str(call.args[0]) for call in mock_print.call_args_list)
    self.assertTrue(found, "Import blueprint failure error message not printed.")

  def test_build_agent_config_success_basic(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "APP_NAME": "a",
        "MIN_INSTANCES": "1",
    }
    config, project, location, staging_bucket = app_sdk.build_agent_config(args, env_vars)
    self.assertEqual(project, "p")
    self.assertEqual(location, "r")
    self.assertEqual(staging_bucket, "s")
    self.assertEqual(config["display_name"], "dpi-a-agent-engine")
    self.assertEqual(config["python_version"], "3.11")

  def test_build_agent_config_missing_region(self):
    args = mock.Mock(delete=False)
    env_vars = {"PROJECT_ID": "p", "STAGING_BUCKET": "s"}
    with self.assertRaises(ValueError) as cm:
      app_sdk.build_agent_config(args, env_vars)
    self.assertIn("REGION", str(cm.exception))

  def test_build_agent_config_psc(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "REDIS_HOST": "1.2.3.4",
        "NETWORK_ATTACHMENT_ID": "na1",
        "MIN_INSTANCES": "1",
    }
    config, _, _, _ = app_sdk.build_agent_config(args, env_vars)
    self.assertEqual(config["psc_interface_config"]["network_attachment"], "projects/p/regions/r/networkAttachments/na1")

  def test_build_agent_config_psc_session_db_url(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "SESSION_DB_URL": "postgres://abc",
        "NETWORK_ATTACHMENT_ID": "na2",
        "MIN_INSTANCES": "1",
    }
    config, _, _, _ = app_sdk.build_agent_config(args, env_vars)
    self.assertEqual(config["psc_interface_config"]["network_attachment"], "projects/p/regions/r/networkAttachments/na2")

  def test_build_agent_config_long_term_memory(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars: dict[str, str] = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "ENABLE_LONG_TERM_MEMORY": "true",
        "MIN_INSTANCES": "1",
    }
    mock_memory_global.get_memory_bank_config.return_value = {"mock": "config"}

    config, _, _, _ = app_sdk.build_agent_config(args, env_vars)

    self.assertIn("context_spec", config)
    self.assertEqual(config["context_spec"]["memory_bank_config"], {"mock": "config"})
    mock_memory_global.get_memory_bank_config.assert_called_once_with(
        project="p", location="r"
    )

  def test_build_agent_config_missing_network_attachment(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars: dict[str, str] = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "REDIS_HOST": "1.2.3.4",
        "MIN_INSTANCES": "1",
    }
    with self.assertRaises(ValueError) as cm:
      app_sdk.build_agent_config(args, env_vars)
    self.assertIn("NETWORK_ATTACHMENT_ID", str(cm.exception))

  def test_build_agent_config_missing_network_attachment_session_db_url(self):
    args = mock.Mock(delete=False, python_version="3.11")
    env_vars: dict[str, str] = {
        "PROJECT_ID": "p",
        "REGION": "r",
        "STAGING_BUCKET": "s",
        "SESSION_DB_URL": "postgres://abc",
        "MIN_INSTANCES": "1",
    }
    with self.assertRaises(ValueError) as cm:
      app_sdk.build_agent_config(args, env_vars)
    self.assertIn("NETWORK_ATTACHMENT_ID", str(cm.exception))


  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  def test_main_create_success(self, mock_print, mock_exists):
    mock_client = mock.Mock()
    mock_vx_global.Client.return_value = mock_client
    mock_remote_agent = mock.Mock()
    mock_remote_agent.api_resource.name = "remote-engine-id"
    mock_client.agent_engines.create.return_value = mock_remote_agent
    
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "mock.env"]), \
         mock.patch.object(app_sdk, 'update_installer_state') as mock_update_state:
      app_sdk.main()
      mock_client.agent_engines.create.assert_called_once()
      mock_update_state.assert_called_once_with("remote-engine-id")

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  def test_main_update_success(self, mock_print, mock_exists):
    mock_client = mock.Mock()
    mock_vx_global.Client.return_value = mock_client
    mock_remote_agent = mock.Mock()
    mock_remote_agent.api_resource.name = "updated-engine-id"
    mock_client.agent_engines.update.return_value = mock_remote_agent
    
    with mock.patch("sys.argv", ["prog", "--update", "--agent-engine", "my-engine", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      mock_client.agent_engines.update.assert_called_once()

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  def test_main_delete_success(self, mock_print, mock_exists):
    mock_client = mock.Mock()
    mock_vx_global.Client.return_value = mock_client
    
    with mock.patch("sys.argv", ["prog", "--delete", "--agent-engine", "my-engine", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      mock_client.agent_engines.delete.assert_called_once_with(
          name="projects/p/locations/r/reasoningEngines/my-engine", force=True
      )

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_main_delete_failure(self, mock_exit, mock_print, mock_exists):
    mock_client = mock.Mock()
    mock_vx_global.Client.return_value = mock_client
    mock_client.agent_engines.delete.side_effect = Exception("Delete failed")

    with mock.patch("sys.argv", ["prog", "--delete", "--agent-engine", "my-engine", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      mock_print.assert_any_call("Error: Agent delete failed: Delete failed")
      mock_exit.assert_called_once_with(1)

  @mock.patch("builtins.print")
  def test_main_missing_engine_for_update(self, mock_print):
    # Mocking parser.error to raise SystemExit allows catching it and stopping main()
    with mock.patch("sys.argv", ["prog", "--update", "--env-vars-file", "mock.env"]):
      with mock.patch("argparse.ArgumentParser.error", side_effect=SystemExit(2)) as mock_error:
        with self.assertRaises(SystemExit):
          app_sdk.main()
        mock_error.assert_called_once()

  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  @mock.patch("os.path.exists")
  def test_main_missing_env_file(self, mock_exists, mock_exit, mock_print):
    def exists_side_effect(path):
        if str(path) == "missing.env": return False
        return True
    mock_exists.side_effect = exists_side_effect
    
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "missing.env"]):
      app_sdk.main()
      mock_print.assert_any_call("Error: Environment file not found at missing.env")
      mock_exit.assert_called_with(1)

  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_main_config_error_exit(self, mock_exit, mock_print):
    mock_dotenv_global.dotenv_values.return_value = {} # Missing PROJECT_ID
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      # Configuration Error: print is called then sys.exit(1)
      mock_exit.assert_called_with(1)

  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_main_init_error_exit(self, mock_exit, mock_print):
    mock_vx_global.init.side_effect = Exception("Init failed")
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      mock_print.assert_any_call("Initialization Error: Init failed")
      mock_exit.assert_called_with(1)

  @mock.patch("builtins.print")
  @mock.patch("sys.exit")
  def test_main_operation_error_exit(self, mock_exit, mock_print):
    mock_client = mock.Mock()
    mock_vx_global.Client.return_value = mock_client
    mock_client.agent_engines.create.side_effect = Exception("Create failed")
    
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "mock.env"]):
      app_sdk.main()
      mock_print.assert_any_call("Error: Agent operation failed: Create failed")
      mock_exit.assert_called_with(1)

  def test_main_no_action_fails(self):
    # This verifies that action_group is required=True (line 196).
    # Since we removed the fallback to --create, argparse should now fail on naked "prog"
    with mock.patch("sys.argv", ["prog", "--env-vars-file", "mock.env"]), \
         mock.patch("argparse.ArgumentParser.error", side_effect=SystemExit(2)) as mock_error:
      with self.assertRaises(SystemExit):
        app_sdk.main()
      mock_error.assert_called_once()

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("builtins.print")
  def test_main_with_explicit_env_file(self, mock_print, mock_exists):
    with mock.patch("sys.argv", ["prog", "--create", "--env-vars-file", "my.env"]), \
         mock.patch.object(app_sdk, 'update_installer_state'):
      app_sdk.main()
      mock_dotenv_global.dotenv_values.assert_called_with(dotenv_path="my.env")
      mock_print.assert_any_call("Loaded environment variables from my.env")


if __name__ == "__main__":
  unittest.main()
