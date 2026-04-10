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

import threading
from unittest import mock

from absl.testing import absltest as googletest
from fastapi.testclient import TestClient
from fastapi.websockets import WebSocketDisconnect
import httpx

import main

class TestMain(googletest.TestCase):

    def setUp(self):
        super().setUp()
        self.client = TestClient(main.app)

    @mock.patch.object(main, 'list_google_cloud_projects', autospec=True)
    def test_get_projects_success(self, mock_list):
        """Verifies that the /projects endpoint returns a list of project IDs."""
        mock_list.return_value = ["project-1", "project-2"]
        response = self.client.get("/projects")
        self.assertEqual(response.status_code, 200)
        # Resilient to list order changes
        self.assertCountEqual(response.json(), ["project-1", "project-2"])

    @mock.patch.object(main, 'list_google_cloud_regions', autospec=True)
    def test_get_regions_success(self, mock_list):
        """Verifies that the /regions endpoint returns a list of GCP regions."""
        mock_list.return_value = ["us-central1"]
        response = self.client.get("/regions")
        self.assertEqual(response.status_code, 200)
        self.assertCountEqual(response.json(), ["us-central1"])

    # --- WebSocket Tests ---

    @mock.patch.object(main, 'run_app_deployment', autospec=True)
    def test_ws_deploy_app_success(self, mock_run):
        """Verifies WebSocket deployment trigger using a deterministic event."""
        call_event = threading.Event()

        async def mock_side_effect(*args, **kwargs) -> None:
            call_event.set()
        mock_run.side_effect = mock_side_effect

        with self.client.websocket_connect("/ws/deployApp") as websocket:
            websocket.send_json({
                "app_name": "test-app",
                "components": {"gateway": True},
                "domain_names": {"gateway": "test.com"},
                "image_urls": {"gateway": "gcr.io/test"},
                "registry_url": "https://reg.api",
                "registry_config": {"subscriber_id": "s", "key_id": "k"},
                "domain_config": {"domainType": "type", "baseDomain": "d", "dnsZone": "z"}
            })

            # Wait for the background task to signal completion
            if not call_event.wait(timeout=1.0):
                self.fail("run_app_deployment was not called within the timeout.")

            mock_run.assert_called()

    def test_ws_deploy_infra_json_error(self):
        """Verifies that invalid JSON in WebSocket results in an error message."""
        with self.client.websocket_connect("/ws/deployInfra") as websocket:
            websocket.send_text("invalid json")
            data = websocket.receive_json()
            self.assertEqual(data["type"], "error")
            self.assertIn("Invalid JSON payload", data["message"])

    # --- State Management Tests ---

    @mock.patch.object(main.ui_state, 'store_bulk_values', autospec=True)
    def test_store_bulk_success(self, mock_store):
        response = self.client.post("/store/bulk", json={"key": "value"})
        self.assertEqual(response.status_code, 201)
        mock_store.assert_called_once_with({"key": "value"})

    @mock.patch.object(main.ui_state, 'load_all_data', autospec=True)
    def test_get_all_stored_data(self, mock_load):
        mock_load.return_value = {"key": "value"}
        response = self.client.get("/store")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"key": "value"})

    @mock.patch.object(main.os.path, 'exists', autospec=True)
    def test_get_installer_state_not_found(self, mock_exists):
        mock_exists.return_value = False
        response = self.client.get("/api/installer-state")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {})

    # --- Dynamic Proxy Tests ---

    @mock.patch.object(httpx.AsyncClient, 'post', autospec=True)
    def test_dynamic_proxy_success(self, mock_post):
        """Verifies successful proxy forwarding without impersonation."""
        mock_res = mock.create_autospec(httpx.Response, instance=True)
        mock_res.status_code = 200
        mock_res.content = b"Success"
        mock_res.raise_for_status.return_value = None

        mock_post.return_value = mock_res

        payload = {"target_url": "https://api.test", "payload": {"k": "v"}}
        response = self.client.post("/api/dynamic-proxy", json=payload)

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), "Success")
        mock_post.assert_called_once_with(
            mock.ANY, "https://api.test", json={"k": "v"}, headers={}, timeout=10.0
        )

    @mock.patch.object(main.google_auth_requests, 'Request', autospec=True)
    @mock.patch.object(main.impersonated_credentials, 'IDTokenCredentials', autospec=True)
    @mock.patch.object(main.impersonated_credentials, 'Credentials', autospec=True)
    @mock.patch.object(main.google.auth, 'default', autospec=True)
    @mock.patch.object(httpx.AsyncClient, 'post', autospec=True)
    def test_dynamic_proxy_impersonation_success(
        self, mock_post, mock_default, mock_creds, mock_id_creds, mock_request
    ):
        """Verifies successful service account impersonation for proxy requests."""
        # Note: patch.object argument order is the reverse of decorator order
        mock_default.return_value = (mock.MagicMock(), "project-id")

        mock_id_token_inst = mock.MagicMock()
        mock_id_token_inst.token = "mock-token"
        mock_id_creds.return_value = mock_id_token_inst

        mock_res = mock.create_autospec(httpx.Response, instance=True)
        mock_res.status_code = 200
        mock_res.content = b"Success"
        mock_res.raise_for_status.return_value = None
        mock_post.return_value = mock_res

        payload = {
            "target_url": "https://api.test",
            "payload": {"k": "v"},
            "impersonate_service_account": "test-sa@project.com",
            "audience": "https://audience.api"
        }
        response = self.client.post("/api/dynamic-proxy", json=payload)

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), "Success")

        # Verify arguments using idiomatic mock assertions
        mock_creds.assert_called_once_with(
            mock_default.return_value[0],
            target_principal="test-sa@project.com",
            target_scopes=["https://www.googleapis.com/auth/cloud-platform"]
        )
        mock_id_creds.assert_called_once_with(
            mock_creds.return_value,
            target_audience="https://audience.api",
            include_email=True
        )
        mock_post.assert_called_once_with(
            mock.ANY, "https://api.test", json={"k": "v"},
            headers={"Authorization": "Bearer mock-token"}, timeout=10.0
        )

    @mock.patch.object(main.google.auth, 'default', autospec=True)
    def test_dynamic_proxy_impersonation_failure(self, mock_default):
        """Verifies that impersonation authentication failures return a 500 status."""
        mock_default.side_effect = Exception("Impersonation failed")

        payload = {
            "target_url": "https://api.test",
            "payload": {"k": "v"},
            "impersonate_service_account": "test-sa@project.com",
            "audience": "https://audience.api"
        }
        response = self.client.post("/api/dynamic-proxy", json=payload)

        self.assertEqual(response.status_code, 500)
        self.assertIn("Impersonation failed", response.text)

if __name__ == "__main__":
    googletest.main()
