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
import asyncio
import json
import logging
from unittest.mock import MagicMock, AsyncMock, patch, call
import httpx

import services.health_checks as hc

class TestHealthChecks(unittest.IsolatedAsyncioTestCase):

    def setUp(self):
        # Mock logging to capture output.
        self.mock_logger = patch('services.health_checks.logger').start()
        self.mock_logger.handlers = []
        self.mock_logger.propagate = False
        self.mock_logger.setLevel(logging.NOTSET)

        # Mock WebSocket for sending messages.
        self.mock_websocket = AsyncMock()
        self.mock_websocket.send_text = AsyncMock()

        self.patcher_timeout = patch('services.health_checks.HealthCheckConstants.HEALTH_CHECK_TIMEOUT_SECONDS', 60)
        self.patcher_delay = patch('services.health_checks.HealthCheckConstants.HEALTH_CHECK_ATTEMPT_DELAY_SECONDS', 1)
        self.patcher_timeout.start()
        self.patcher_delay.start()

    def tearDown(self):
        patch.stopall()

    @patch('httpx.AsyncClient')
    async def test_perform_single_health_check_success(self, MockAsyncClient):
        """
        Test successful single health check (HTTP 200).
        """
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_client_instance = AsyncMock() 
        mock_client_instance.get.return_value = mock_response

        result = await hc.perform_single_health_check(
            "test_service", "test-url.com", self.mock_websocket, mock_client_instance
        )

        expected_health_url = "https://test-url.com/health"
        mock_client_instance.get.assert_called_once_with(expected_health_url)
        self.assertTrue(result)
        self.mock_logger.info.assert_called_once_with(
            f"Health check for test_service at test-url.com (endpoint: {expected_health_url}) PASSED (HTTP 200)."
        )
        self.mock_websocket.send_text.assert_called_once_with(json.dumps({
            "type": "success",
            "service": "test_service",
            "url": "test-url.com",
            "health_url": expected_health_url,
            "status_code": 200,
            "message": f"Health check for test_service at test-url.com (endpoint: {expected_health_url}) PASSED (HTTP 200)."
        }))
        self.mock_logger.warning.assert_not_called()
        self.mock_logger.exception.assert_not_called()


    @patch('httpx.AsyncClient')
    async def test_perform_single_health_check_http_failure(self, MockAsyncClient):
        """
        Test single health check with non-200 HTTP status.
        """
        mock_response = MagicMock()
        mock_response.status_code = 404
        mock_client_instance = AsyncMock()
        mock_client_instance.get.return_value = mock_response

        result = await hc.perform_single_health_check(
            "test_service", "test-url.com", self.mock_websocket, mock_client_instance
        )

        expected_health_url = "https://test-url.com/health"
        self.assertFalse(result)
        self.mock_logger.warning.assert_called_once_with(
            f"Health check for test_service at test-url.com (endpoint: {expected_health_url}) FAILED: HTTP 404."
        )
        self.mock_websocket.send_text.assert_called_once_with(json.dumps({
            "type": "warning",
            "service": "test_service",
            "url": "test-url.com",
            "health_url": expected_health_url,
            "status_code": 404,
            "message": f"Health check for test_service at test-url.com (endpoint: {expected_health_url}) FAILED: HTTP 404."
        }))
        self.mock_logger.info.assert_not_called()
        self.mock_logger.exception.assert_not_called()

    @patch('httpx.AsyncClient')
    async def test_perform_single_health_check_request_error(self, MockAsyncClient):
        """
        Test single health check handling httpx.RequestError.
        """
        mock_client_instance = AsyncMock()
        mock_client_instance.get.side_effect = httpx.RequestError("Connection failed", request=httpx.Request("GET", "https://test-url.com/health")) 

        result = await hc.perform_single_health_check(
            "test_service", "test-url.com", self.mock_websocket, mock_client_instance
        )

        expected_health_url = "https://test-url.com/health"
        self.assertFalse(result)
        self.mock_logger.warning.assert_called_once()
        self.assertIn("FAILED: Request Error - Connection failed", self.mock_logger.warning.call_args[0][0])
            
        self.mock_websocket.send_text.assert_called_once()
        sent_message = json.loads(self.mock_websocket.send_text.call_args[0][0])
        self.assertEqual(sent_message["type"], "warning")
        self.assertIn("FAILED: Request Error - Connection failed", sent_message["message"])
        self.assertEqual(sent_message["status_code"], None)
        self.mock_logger.info.assert_not_called()
        self.mock_logger.exception.assert_not_called()

    @patch('httpx.AsyncClient')
    async def test_perform_single_health_check_general_exception(self, MockAsyncClient):
        """
        Test single health check handling a general Exception.
        """
        mock_client_instance = AsyncMock()
        mock_client_instance.get.side_effect = Exception("Unexpected error")

        result = await hc.perform_single_health_check(
            "test_service", "test-url.com", self.mock_websocket, mock_client_instance
        )

        expected_health_url = "https://test-url.com/health"
        self.assertFalse(result)
        self.mock_logger.exception.assert_called_once_with(f"An unexpected error occurred during health check for test_service: Unexpected error.")
        self.mock_websocket.send_text.assert_called_once()
        sent_message = json.loads(self.mock_websocket.send_text.call_args[0][0])
        self.assertEqual(sent_message["type"], "error")
        self.assertIn("An unexpected error occurred during health check for test_service: Unexpected error.", sent_message["message"])
        self.assertEqual(sent_message["status_code"], None)
        self.mock_logger.info.assert_not_called()
        self.mock_logger.warning.assert_not_called()


    async def test_run_websocket_health_check_invalid_input(self):
        """
        Test run_websocket_health_check with invalid input type.
        """
        await hc.run_websocket_health_check(self.mock_websocket, "not_a_dict")

        self.mock_logger.error.assert_called_once_with("Invalid payload format received for healthCheck in service.")
        self.mock_websocket.send_text.assert_called_once_with(json.dumps({
            "type": "error",
            "message": "Invalid payload format. Expected a dictionary of serviceName: serviceUrl strings."
        }))

    async def test_run_websocket_health_check_empty_services(self):
        """
        Test run_websocket_health_check with an empty dictionary of services.
        """
        await hc.run_websocket_health_check(self.mock_websocket, {})

        self.mock_logger.warning.assert_called_once_with("No service URLs provided for health check.")
        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "info",
            "message": "Starting health checks for provided service URLs..."
        }))
        self.mock_websocket.send_text.assert_called_with(json.dumps({
            "type": "warning",
            "message": "No service URLs provided for health check. Health check skipped."
        }))
        self.assertEqual(self.mock_websocket.send_text.call_count, 2)


    @patch('services.health_checks.perform_single_health_check', new_callable=AsyncMock)
    @patch('httpx.AsyncClient')
    async def test_run_websocket_health_check_all_healthy_first_attempt(self, MockAsyncClient, mock_perform_single_health_check):
        """
        Test run_websocket_health_check where all services are healthy on the first attempt.
        """
        mock_perform_single_health_check.return_value = True

        service_urls = {
            "service_a": "url-a.com",
            "service_b": "url-b.com"
        }

        mock_client_instance = AsyncMock()
        MockAsyncClient.return_value.__aenter__.return_value = mock_client_instance

        # Mock event loop time for consistent elapsed_time.
        # start_time = 0
        # current_time for first log = 0
        mock_loop = MagicMock()
        mock_loop.time.side_effect = [0, 0] 
        with patch('asyncio.get_event_loop', return_value=mock_loop):
            await hc.run_websocket_health_check(self.mock_websocket, service_urls)

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "info",
            "message": "Starting health checks for provided service URLs..."
        }))
        self.mock_logger.info.assert_any_call("Starting comprehensive health checks for: service_a, service_b")

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "log",
            "message": "Attempt 1: Checking 2 service URL(s) for health... (0s elapsed / 60s timeout)"
        }))
        self.mock_logger.info.assert_any_call("Attempt 1: Checking 2 services. Elapsed: 0s.")

        # Verify individual health checks were called (2 calls for 2 services).
        self.assertEqual(mock_perform_single_health_check.call_count, 2)
        mock_perform_single_health_check.assert_any_call(
            "service_a", "url-a.com", self.mock_websocket, mock_client_instance
        )
        mock_perform_single_health_check.assert_any_call(
            "service_b", "url-b.com", self.mock_websocket, mock_client_instance
        )

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "success",
            "action": "all_services_healthy",
            "message": "All deployed service URLs are now healthy and reachable!"
        }))
        self.mock_logger.info.assert_any_call("All services are healthy.")
        self.mock_logger.warning.assert_not_called() # No pending services
        self.mock_logger.error.assert_not_called()
            
        # Total send_text calls: 1 (initial info) + 1 (attempt log) + 1 (final success) = 3
        self.assertEqual(self.mock_websocket.send_text.call_count, 3)


    @patch('services.health_checks.perform_single_health_check', new_callable=AsyncMock)
    @patch('httpx.AsyncClient')
    @patch('asyncio.sleep', new_callable=AsyncMock)
    async def test_run_websocket_health_check_multiple_attempts(self, mock_sleep, MockAsyncClient, mock_perform_single_health_check):
        """
        Test run_websocket_health_check where services become healthy over multiple attempts.
        """
        # Mock perform_single_health_check to pass service_b on first attempt, service_a on second.
        mock_perform_single_health_check.side_effect = [
            False,
            True,
            True
        ]
            
        service_urls = {
            "service_a": "url-a.com",
            "service_b": "url-b.com"
        }
        mock_client_instance = AsyncMock()
        MockAsyncClient.return_value.__aenter__.return_value = mock_client_instance

        # Mock event loop time
        # Sequence:
        # 0: start_time
        # 0: current_time (Attempt 1 log)
        # 1: current_time (Attempt 2 log, after 1 sec sleep)
        mock_loop = MagicMock()
        mock_loop.time.side_effect = [0, 0, 1] 
        with patch('asyncio.get_event_loop', return_value=mock_loop):
            await hc.run_websocket_health_check(self.mock_websocket, service_urls)

        self.mock_logger.info.assert_any_call("Starting comprehensive health checks for: service_a, service_b")

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "log", "message": "Attempt 1: Checking 2 service URL(s) for health... (0s elapsed / 60s timeout)"
        }))
        self.mock_logger.info.assert_any_call("Attempt 1: Checking 2 services. Elapsed: 0s.")

        # Expect one service to be reported unhealthy (service_a).
        self.mock_logger.warning.assert_any_call("1 services still pending health check. Retrying...")
        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "log", "message": "1 service URL(s) are not yet healthy. Retrying in 1 seconds..."
        }))
        mock_sleep.assert_called_once_with(1) # Assert delay

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "log", "message": "Attempt 2: Checking 1 service URL(s) for health... (1s elapsed / 60s timeout)"
        }))
        self.mock_logger.info.assert_any_call("Attempt 2: Checking 1 services. Elapsed: 1s.")

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "success", "action": "all_services_healthy", "message": "All deployed service URLs are now healthy and reachable!"
        }))
        self.mock_logger.info.assert_any_call("All services are healthy.")

        self.assertEqual(mock_perform_single_health_check.call_count, 3) # 2 on first, 1 on second
        self.mock_logger.error.assert_not_called()
            
        # Total send_text calls:
        # 1 (initial info)
        # 1 (attempt 1 log)
        # 1 (pending message after attempt 1)
        # 1 (attempt 2 log)
        # 1 (final success)
        # Total = 5
        self.assertEqual(self.mock_websocket.send_text.call_count, 5)


    @patch('services.health_checks.perform_single_health_check', new_callable=AsyncMock)
    @patch('httpx.AsyncClient')
    @patch('asyncio.sleep', new_callable=AsyncMock)
    async def test_run_websocket_health_check_timeout(self, mock_sleep, MockAsyncClient, mock_perform_single_health_check):
        """
        Test run_websocket_health_check where the health checks time out.
        """
        mock_perform_single_health_check.return_value = False
            
        service_urls = {
            "service_a": "url-a.com"
        }

        # Mock the AsyncClient context manager entry
        mock_client_instance = AsyncMock()
        MockAsyncClient.return_value.__aenter__.return_value = mock_client_instance

        # Mock event loop time to simulate timeout (patched timeout is 60s, delay is 1s)
        # Sequence of time calls needed:
        # 1. start_time (0)
        # 2. current_time for Attempt 1 log (0)
        # 3. current_time for Attempt 2 log (1) after first sleep
        # ...
        # 62. current_time for Attempt 61 log (60) after 60th sleep
        # 63. current_time that triggers timeout (61) after 61st sleep
        # So, we need 63 values in total for side_effect: [0, 0, 1, ..., 61]
        mock_loop = MagicMock()
        mock_loop.time.side_effect = [0] + list(range(62)) 
        
        with patch('asyncio.get_event_loop', return_value=mock_loop):
            await hc.run_websocket_health_check(self.mock_websocket, service_urls)

        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "error",
            "action": "health_check_timeout",
            "message": "Health checks timed out after 61 seconds (1.0 minutes). The following service URLs are still unhealthy: service_a"
        }))
        self.mock_logger.error.assert_called_once_with("Health checks timed out. Unhealthy services: service_a")
            
        self.assertEqual(mock_sleep.call_count, 61)
        self.mock_logger.info.assert_any_call("Attempt 1: Checking 1 services. Elapsed: 0s.")
        self.mock_logger.info.assert_any_call("Attempt 61: Checking 1 services. Elapsed: 60s.")
        self.mock_logger.warning.assert_any_call("1 services still pending health check. Retrying...")
            
        # Check total calls to perform_single_health_check
        self.assertEqual(mock_perform_single_health_check.call_count, 61)
            
        # Expected WebSocket send_text calls:
        # 1 (initial info)
        # 61 * (1 attempt log) = 61 messages (for attempts 1 to 61)
        # 61 * (1 pending message) = 61 messages (sent after each of the 61 failed attempts)
        # 1 (Timeout error message)
        # Total = 1 + 61 + 61 + 1 = 124
        self.assertEqual(self.mock_websocket.send_text.call_count, 124)


    @patch('services.health_checks.perform_single_health_check', new_callable=AsyncMock)
    @patch('httpx.AsyncClient')
    async def test_run_websocket_health_check_general_exception_in_loop(self, MockAsyncClient, mock_perform_single_health_check):
        """
        Test run_websocket_health_check handling a general Exception within the polling loop.
        """
        mock_perform_single_health_check.side_effect = Exception("Loop error")
            
        service_urls = {
            "service_a": "url-a.com"
        }

        mock_client_instance = AsyncMock()
        MockAsyncClient.return_value.__aenter__.return_value = mock_client_instance

        # Mock event loop time for consistent elapsed_time
        # start_time = 0, current_time = 0 for the first log.
        mock_loop = MagicMock()
        mock_loop.time.side_effect = [0, 0] 
        with patch('asyncio.get_event_loop', return_value=mock_loop):
            with self.assertRaises(Exception) as cm:
                await hc.run_websocket_health_check(self.mock_websocket, service_urls)

        self.assertEqual(str(cm.exception), "Loop error")
        self.mock_logger.exception.assert_called_once_with("An unexpected error occurred within run_websocket_health_check: Loop error")
        # Assert initial info and attempt logs are called before the error.
        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "info",
            "message": "Starting health checks for provided service URLs..."
        }))
        self.mock_websocket.send_text.assert_any_call(json.dumps({
            "type": "log",
            "message": "Attempt 1: Checking 1 service URL(s) for health... (0s elapsed / 60s timeout)"
        }))
        self.mock_logger.error.assert_not_called()
        self.mock_logger.warning.assert_not_called()

if __name__ == '__main__':
    unittest.main()