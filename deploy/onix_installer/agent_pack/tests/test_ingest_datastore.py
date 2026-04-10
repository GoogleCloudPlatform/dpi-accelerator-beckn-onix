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
import sys
import unittest
from unittest import mock

from google3.third_party.dpi_becknonix.deploy.onix_installer.agent_pack import ingest_datastore


class TestIngestDatastore(unittest.TestCase):

  def setUp(self):
    self.patcher_client = mock.patch.object(
        ingest_datastore.discoveryengine, "DocumentServiceClient", autospec=True
    )
    self.mock_client_class = self.enterContext(self.patcher_client)
    self.mock_client_instance = self.mock_client_class.return_value
    self.mock_client_instance.__enter__.return_value = self.mock_client_instance

    # Setup basic success operation mock
    self.mock_operation = mock.Mock()
    self.mock_operation.operation.name = (
        "projects/123/locations/global/operations/456"
    )
    self.mock_result = mock.Mock()
    self.mock_result.error_samples = None  # No errors
    self.mock_operation.result.return_value = self.mock_result
    self.mock_client_instance.import_documents.return_value = (
        self.mock_operation
    )

  def test_parse_datastore_id_success(self):
    fqdn = "projects/123/locations/us-central1/collections/default_collection/dataStores/my-datastore-123"
    short_id, location = ingest_datastore.parse_datastore_id(fqdn)
    self.assertEqual(short_id, "my-datastore-123")
    self.assertEqual(location, "us-central1")

  def test_parse_datastore_id_trailing_slash(self):
    # Ensure empty short ID raises ValueError
    fqdn = "projects/123/locations/global/collections/default_collection/dataStores/"
    with self.assertRaisesRegex(
        ValueError, "Invalid fully qualified datastore ID"
    ):
      ingest_datastore.parse_datastore_id(fqdn)

  def test_parse_datastore_id_failure(self):
    fqdn = "projects/123/locations/europe-west1/collections/default_collection/somethingElse/my-datastore"
    with self.assertRaisesRegex(
        ValueError, "Invalid fully qualified datastore ID"
    ):
      ingest_datastore.parse_datastore_id(fqdn)

  def test_parse_datastore_id_empty(self):
    with self.assertRaisesRegex(
        ValueError, "Invalid fully qualified datastore ID"
    ):
      ingest_datastore.parse_datastore_id("")

  def test_ingest_to_datastore_success(self):
    project_id = "test-project"
    location = "global"
    datastore_id = "test-datastore"
    gcs_uris = ["gs://my-bucket/docs/*.pdf"]

    result_op_name = ingest_datastore.ingest_to_datastore(
        project_id, location, datastore_id, gcs_uris
    )

    self.assertEqual(
        result_op_name, "projects/123/locations/global/operations/456"
    )
    self.mock_client_class.assert_called_once_with(client_options=None)
    self.mock_client_instance.import_documents.assert_called_once()

    # Assert request payload
    call_args = self.mock_client_instance.import_documents.call_args[1]
    request = call_args["request"]

    expected_parent = "projects/test-project/locations/global/collections/default_collection/dataStores/test-datastore/branches/default_branch"
    self.assertEqual(request.parent, expected_parent)
    self.assertEqual(
        request.gcs_source.input_uris, ["gs://my-bucket/docs/*.pdf"]
    )

  def test_ingest_to_datastore_non_global_location(self):
    project_id = "test-project"
    location = "us-central1"
    datastore_id = "test-datastore"
    gcs_uris = ["gs://my-bucket/docs/*.pdf"]

    ingest_datastore.ingest_to_datastore(
        project_id, location, datastore_id, gcs_uris
    )

    expected_options = {
        "api_endpoint": "us-central1-discoveryengine.googleapis.com"
    }
    self.mock_client_class.assert_called_once_with(
        client_options=expected_options
    )

  def test_ingest_to_datastore_document_errors(self):
    # Setup mock to return error samples
    self.mock_result.error_samples = [
        {"error": "Bad PDF format", "file": "gs://b/bad.pdf"}
    ]

    with self.assertRaisesRegex(
        ValueError, "Import operation encountered document errors"
    ):
      ingest_datastore.ingest_to_datastore(
          "test_proj", "global", "test_ds", ["gs://bucket/*"]
      )

  def test_ingest_to_datastore_api_exception(self):
    self.mock_client_instance.import_documents.side_effect = Exception(
        "API Quota Exceeded"
    )

    with self.assertRaisesRegex(Exception, "API Quota Exceeded"):
      ingest_datastore.ingest_to_datastore(
          "test_proj", "global", "test_ds", ["gs://bucket/*"]
      )

  @mock.patch.object(
      sys,
      "argv",
      [
          "ingest_datastore.py",
          "test-proj",
          (
              '{"agent_a":'
              ' "projects/1/locations/europe-west1/collections/default_collection/dataStores/ds-a"}'
          ),
          '{"agent_a": "gs://bucket/a.pdf"}',
      ],
  )
  @mock.patch.object(ingest_datastore, "ingest_to_datastore", autospec=True)
  def test_main_success(self, mock_ingest):
    with mock.patch.object(builtins, "print"):  # Suppress output
      ingest_datastore.main()

    mock_ingest.assert_called_once_with(
        project_id="test-proj",
        location="europe-west1",
        datastore_id="ds-a",
        gcs_uris=["gs://bucket/a.pdf"],
    )

  @mock.patch.object(
      sys,
      "argv",
      [
          "ingest_datastore.py",
          "test-proj",
          '{"agent_a": "projects/.../dataStores/ds-a"}',
          '{"agent_b": "gs://bucket/b.pdf"}',
      ],
  )  # Mismatched keys
  @mock.patch.object(ingest_datastore, "ingest_to_datastore", autospec=True)
  def test_main_skip_unmatched_agent(self, mock_ingest):
    with mock.patch.object(builtins, "print") as mock_print:
      ingest_datastore.main()

    mock_ingest.assert_not_called()
    mock_print.assert_any_call(
        "Warning: Key 'agent_b' found in imports but matching datastore was not"
        " created by Terraform. Skipping."
    )

  @mock.patch.object(
      sys,
      "argv",
      [
          "ingest_datastore.py",
          "test-proj",
          '{"agent_a": "projects/.../dataStores/ds-a"}',  # Malformed FQDN (missing location)
          '{"agent_a": "gs://bucket/a.pdf"}',
      ],
  )
  @mock.patch.object(ingest_datastore, "ingest_to_datastore", autospec=True)
  def test_main_parsing_failure_handled(self, mock_ingest):
    """Tests that a malformed fully_qualified_id raises a ValueError seamlessly caught by the main loop."""

    # Using mock_print to inspect the printed logs specifically to ensure the error was caught
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit) as cm:
        ingest_datastore.main()

      self.assertEqual(cm.exception.code, 1)

    # Ingestion shouldn't have been called at all since parsing threw up
    mock_ingest.assert_not_called()

    # The exception should be nicely formatted via our main loop's print
    mock_print.assert_any_call(
        "Ingestion failed for agent_a: Invalid fully qualified datastore ID:"
        " 'projects/.../dataStores/ds-a'"
    )

  @mock.patch.object(
      sys, "argv", ["ingest_datastore.py", "test-proj", "{invalid json}", "{}"]
  )
  def test_main_invalid_json_args_exits(self):
    with mock.patch.object(builtins, "print") as mock_print:
      with self.assertRaises(SystemExit) as cm:
        ingest_datastore.main()

      self.assertEqual(cm.exception.code, 1)
      mock_print.assert_any_call(
          "Failed to parse JSON arguments: Expecting property name enclosed in"
          " double quotes: line 1 column 2 (char 1)"
      )

  @mock.patch.object(
      sys,
      "argv",
      ["ingest_datastore.py", "test-proj", "{}", '{"agent": "gs://..."}'],
  )
  @mock.patch.object(ingest_datastore, "ingest_to_datastore", autospec=True)
  def test_main_no_datastores_provisioned(self, mock_ingest):
    with mock.patch.object(builtins, "print") as mock_print:
      ingest_datastore.main()

    mock_ingest.assert_not_called()
    mock_print.assert_any_call(
        "No datastore IDs found in Terraform outputs. Skipping ingestion."
    )


if __name__ == "__main__":
  unittest.main()
