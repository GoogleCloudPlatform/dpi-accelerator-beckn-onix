#!/usr/bin/env python3
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

from collections.abc import Sequence
import json
import sys

from google.cloud import discoveryengine_v1 as discoveryengine


def ingest_to_datastore(
    project_id: str, location: str, datastore_id: str, gcs_uris: Sequence[str]
) -> str:
  """Ingests documents from GCS into a Discovery Engine Datastore.

  Args:
    project_id: The Google Cloud project ID.
    location: The location of the Discovery Engine service (e.g., "global").
    datastore_id: The short ID of the Discovery Engine Datastore.
    gcs_uris: A sequence of GCS URIs pointing to the documents to ingest.

  Returns:
    The name of the long-running operation for the import.

  Raises:
    ValueError: If the import operation completes but the response contains
      `error_samples`, indicating document-level ingestion failures.
    Exception: For any other exceptions that occur during the ingestion process
      or while waiting for the operation to complete.
  """

  client_options = (
      {"api_endpoint": f"{location}-discoveryengine.googleapis.com"}
      if location != "global"
      else None
  )
  with discoveryengine.DocumentServiceClient(
      client_options=client_options
  ) as client:
    # Note: Extracting the actual ID from the fully qualified ID
    # The terraform fully qualified ID is: projects/{{project_number}}/locations/global/collections/default_collection/dataStores/{{datastore_id}}
    # The API expects just the raw datastore ID, so we parse it or pass the full string depending on the API parent parameter.
    # The parent string is the full path up to the collections

    # The parent string for import is strictly: projects/{project}/locations/{location}/collections/default_collection/dataStores/{datastore_id}/branches/default_branch
    # Hardcoding this string rather than using client.branch_path prevents missing argument errors
    # across different google-cloud-discoveryengine SDK versions where 'collections' might not be a parameter yet.
    parent = f"projects/{project_id}/locations/{location}/collections/default_collection/dataStores/{datastore_id}/branches/default_branch"

    request = discoveryengine.ImportDocumentsRequest(
        parent=parent,
        gcs_source=discoveryengine.GcsSource(
            input_uris=gcs_uris, data_schema="content"
        ),
        # Unstructured documents (PDFs, txt, html, etc)
        reconciliation_mode=discoveryengine.ImportDocumentsRequest.ReconciliationMode.INCREMENTAL,
    )

    # Make the request
    print(
        f"Triggering import job for datastore: {datastore_id} with URIs:"
        f" {gcs_uris}"
    )
    operation = client.import_documents(request=request)

    print(f"Waiting for operation to complete: {operation.operation.name}")

    try:
      response = operation.result()
      if getattr(response, "error_samples", None):
        error_msg = (
            "Import operation encountered document errors:"
            f" {response.error_samples}"
        )
        print(f"Failed during document ingestion: {error_msg}")
        raise ValueError(error_msg)

      print(f"Import completed successfully: {response}")
      return operation.operation.name
    except Exception as e:
      print(f"Failed during document ingestion: {e}")
      raise


def parse_datastore_id(fully_qualified_id: str) -> tuple[str, str]:
  """Extracts the short ID and location from a fully qualified Datastore ID.

  Terraform exports:
  projects/<number>/locations/<location>/collections/default_collection/dataStores/<short-id>

  Args:
    fully_qualified_id: The full ID of the datastore from Terraform.

  Returns:
    A tuple of (short_id, location). 
  
  Raises:
    ValueError: If the fully_qualified_id cannot be parsed to extract a short ID
      and location.
  """
  parts = fully_qualified_id.split("/")

  try:
    idx_ds = parts.index("dataStores")
    short_id = parts[idx_ds + 1]

    idx_loc = parts.index("locations")
    location = parts[idx_loc + 1]

    if not short_id or not location:
        raise ValueError("Extracted short_id or location is empty.")
  except (ValueError, IndexError) as e:
    raise ValueError(
        f"Invalid fully qualified datastore ID: '{fully_qualified_id}'"
    ) from e

  return short_id, location


def main():
  if len(sys.argv) != 4:
    print(
        "Usage: python3 ingest_datastore.py <project_id> <datastore_ids_json>"
        " <datastore_imports_json>"
    )
    sys.exit(1)

  project_id = sys.argv[1]
  datastore_ids_json = sys.argv[2]
  datastore_imports_json = sys.argv[3]

  try:
    datastore_ids = json.loads(datastore_ids_json)
    datastore_imports = json.loads(datastore_imports_json)
  except json.JSONDecodeError as e:
    print(f"Failed to parse JSON arguments: {e}")
    sys.exit(1)

  if not datastore_ids:
    print(
        "No datastore IDs found in Terraform outputs. Skipping ingestion."
    )
    return

  print(f"Found datastore mapping to ingest: {datastore_imports}")

  failed_details = []

  for key, gcs_uri in datastore_imports.items():
    if key not in datastore_ids:
      print(
          f"Warning: Key '{key}' found in imports but matching datastore was not created"
          " by Terraform. Skipping."
      )
      continue

    fully_qualified_id = datastore_ids[key]

    try:
      short_id, location = parse_datastore_id(fully_qualified_id)
      print(f"Starting ingestion process for {key} -> {short_id} in {location}")

      ingest_to_datastore(
          project_id=project_id,
          location=location,
          datastore_id=short_id,
          gcs_uris=[gcs_uri],  # API expects a list of URIs
      )
    except Exception as e:
      failed_details.append((key, e))
      print(f"Ingestion failed for {key}: {e}")

  total_attempted = len(datastore_imports)
  failed_count = len(failed_details)
  success_count = total_attempted - failed_count

  print("\n--- Ingestion Summary ---")
  print(f"Successful: {success_count}")
  print(f"Failed: {failed_count}")

  if failed_count > 0:
    print("\n❌ Failed Ingestion Details:")
    for failed_key, reason in failed_details:
      print(f"  -> Agent: {failed_key}")
      print(f"     Reason: {reason}")
    print("\nHalting installation due to datastore ingestion failures. Please fix the above issues and retry.")
    sys.exit(1)


if __name__ == "__main__":
  main()
