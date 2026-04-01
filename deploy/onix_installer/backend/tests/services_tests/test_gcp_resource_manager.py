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

# tests/services_tests/test_gcp_resource_manager.py

import unittest
import asyncio
import json
import logging
import subprocess
from unittest.mock import MagicMock, AsyncMock, patch, call

import fastapi
from google.auth.exceptions import GoogleAuthError

# Import the module under test
from services import gcp_resource_manager

class TestGcpUtils(unittest.IsolatedAsyncioTestCase):

    def setUp(self):
        # Mock logging to capture output.
        self.mock_logger = patch('services.gcp_resource_manager.logger').start()
        self.mock_logger.handlers = []
        self.mock_logger.propagate = False
        self.mock_logger.setLevel(logging.NOTSET)

    def tearDown(self):
        patch.stopall()


    @patch('services.gcp_resource_manager.resourcemanager_v3.ProjectsClient')
    async def test_list_google_cloud_projects_success(self, MockProjectsClient):
        """
        Test successful listing of Google Cloud projects.
        """
        # Mock project objects for the pager.
        mock_project1 = MagicMock()
        mock_project1.project_id = "project-b"
        mock_project2 = MagicMock()
        mock_project2.project_id = "project-a"
        
        # Mock the pager behavior.
        mock_pager = [mock_project1, mock_project2]
        
        MockProjectsClient.return_value.search_projects.return_value = mock_pager

        projects = await gcp_resource_manager.list_google_cloud_projects()

        MockProjectsClient.assert_called_once()
        MockProjectsClient.return_value.search_projects.assert_called_once()
        
        self.assertEqual(projects, ["project-a", "project-b"])
        self.mock_logger.info.assert_called_with("Successfully retrieved 2 Google Cloud projects.")
        self.mock_logger.error.assert_not_called()
        self.mock_logger.exception.assert_not_called()

    @patch('services.gcp_resource_manager.resourcemanager_v3.ProjectsClient', side_effect=GoogleAuthError("Auth failed"))
    async def test_list_google_cloud_projects_auth_error(self, MockProjectsClient):
        """
        Test handling of GoogleAuthError when listing projects.
        """
        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_projects()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn("Authentication failed", cm.exception.detail)
        self.mock_logger.error.assert_called_once_with("Authentication failed when listing GCP projects: Auth failed")
        self.mock_logger.exception.assert_not_called()
        self.mock_logger.info.assert_not_called()

    @patch('services.gcp_resource_manager.resourcemanager_v3.ProjectsClient', side_effect=Exception("Unexpected error"))
    async def test_list_google_cloud_projects_general_exception(self, MockProjectsClient):
        """
        Test handling of a general unexpected Exception when listing projects.
        """
        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_projects()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn("An unexpected error occurred", cm.exception.detail)
        self.mock_logger.exception.assert_called_once_with("An unexpected error occurred while listing GCP projects: Unexpected error")
        self.mock_logger.error.assert_not_called()
        self.mock_logger.info.assert_not_called()

    @patch('services.gcp_resource_manager.asyncio.to_thread')
    async def test_list_google_cloud_regions_success(self, mock_to_thread):
        """
        Test successful listing of Google Cloud regions via gcloud CLI.
        """
        mock_regions_data = [
            {"name": "us-central1", "status": "UP"},
            {"name": "europe-west1", "status": "UP"},
            {"name": "asia-east1", "status": "UP"},
        ]
        
        mock_result = MagicMock()
        mock_result.stdout = json.dumps(mock_regions_data)
        mock_result.returncode = 0
        
        mock_to_thread.return_value = mock_result

        regions = await gcp_resource_manager.list_google_cloud_regions()

        mock_to_thread.assert_called_once_with(
            subprocess.run,
            ["gcloud", "compute", "regions", "list", "--format=json"],
            capture_output=True,
            text=True,
            check=True
        )
        self.assertEqual(regions, ["us-central1", "europe-west1", "asia-east1"])
        self.mock_logger.info.assert_has_calls([
            call("Listing Google Cloud regions using gcloud CLI..."),
            call("Successfully retrieved 3 Google Cloud regions via gcloud CLI.")
        ])
        self.mock_logger.error.assert_not_called()
        self.mock_logger.exception.assert_not_called()

    @patch('services.gcp_resource_manager.asyncio.to_thread', side_effect=FileNotFoundError())
    async def test_list_google_cloud_regions_gcloud_not_found(self, mock_to_thread):
        """
        Test handling of FileNotFoundError when gcloud command is not found.
        """
        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_regions()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn("Error: 'gcloud' command not found.", cm.exception.detail)
        self.mock_logger.error.assert_called_once_with("Error: 'gcloud' command not found. Please ensure the Google Cloud SDK is installed and configured in your system's PATH.")
        self.mock_logger.exception.assert_not_called()

    @patch('services.gcp_resource_manager.asyncio.to_thread')
    async def test_list_google_cloud_regions_subprocess_called_process_error(self, mock_to_thread):
        """
        Test handling of subprocess.CalledProcessError when gcloud command fails.
        """
        mock_error_string = "ERROR: (gcloud) Insufficient permissions."
        
        # When subprocess.run is called with text=True, e.stderr is a string.
        # We need to ensure the mocked CalledProcessError also presents stderr as a string.
        mock_to_thread.side_effect = subprocess.CalledProcessError(
            cmd=["gcloud", "compute", "regions", "list"],
            returncode=1,
            # Pass stderr as a string directly, as it would be if text=True was used.
            stderr=mock_error_string 
        )

        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_regions()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn(f"An error occurred while executing the gcloud command. Stderr: {mock_error_string}", cm.exception.detail)
        self.mock_logger.error.assert_called_once_with(f"An error occurred while executing the gcloud command. Stderr: {mock_error_string}")
        self.mock_logger.exception.assert_not_called()

    @patch('services.gcp_resource_manager.asyncio.to_thread')
    async def test_list_google_cloud_regions_json_decode_error(self, mock_to_thread):
        """
        Test handling of json.JSONDecodeError when gcloud output is invalid JSON.
        """
        mock_result = MagicMock()
        mock_result.stdout = "This is not JSON"
        mock_result.returncode = 0
        
        mock_to_thread.return_value = mock_result

        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_regions()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn("Error: Could not parse the JSON output from the gcloud command.", cm.exception.detail)
        self.mock_logger.error.assert_called_once_with("Error: Could not parse the JSON output from the gcloud command.")
        self.mock_logger.exception.assert_not_called()

    @patch('services.gcp_resource_manager.asyncio.to_thread', side_effect=Exception("Generic error"))
    async def test_list_google_cloud_regions_general_exception(self, mock_to_thread):
        """
        Test handling of a general unexpected Exception when listing regions.
        """
        with self.assertRaises(fastapi.HTTPException) as cm:
            await gcp_resource_manager.list_google_cloud_regions()

        self.assertEqual(cm.exception.status_code, 500)
        self.assertIn("An unexpected error occurred", cm.exception.detail)
        
        self.mock_logger.info.assert_called_once_with("Listing Google Cloud regions using gcloud CLI...")
        # The exception log should also be called.
        self.mock_logger.exception.assert_called_once_with("An unexpected error occurred while listing regions: Generic error")

        self.assertEqual(self.mock_logger.info.call_count, 1)


if __name__ == '__main__':
    unittest.main()