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

import asyncio
import json
import logging
import subprocess
from typing import List

from fastapi import HTTPException
from google.auth.exceptions import GoogleAuthError
from google.cloud import resourcemanager_v3

logger = logging.getLogger(__name__)

async def list_google_cloud_projects() -> List[str]:
    """
    Lists all Google Cloud projects accessible by the authenticated user/service account.
    Mimics the logic from the provided main.py using google-cloud-resource-manager library.
    """
    project_ids = []
    try:
        client = resourcemanager_v3.ProjectsClient()

        request = resourcemanager_v3.SearchProjectsRequest()
        project_pager = client.search_projects(request=request)

        for project in project_pager:
            project_ids.append(project.project_id)

        # Sort the list for consistent output.
        project_ids.sort()
        logger.info(f"Successfully retrieved {len(project_ids)} Google Cloud projects.")
        return project_ids

    except GoogleAuthError as e:
        logger.error(f"Authentication failed when listing GCP projects: {e}")
        raise HTTPException(
            status_code=500,
            detail="Authentication failed. Please configure your environment with Google Cloud credentials. "
                   "Try running 'gcloud auth application-default login' in your terminal.",
        )
    except Exception as e:
        logger.exception(f"An unexpected error occurred while listing GCP projects: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"An unexpected error occurred: {e}"
        )

async def list_google_cloud_regions() -> List[str]:
    """
    Lists all available Google Cloud regions for Compute Engine using the gcloud CLI.
    Mimics the logic from the provided main.py using subprocess.
    """
    logger.info("Listing Google Cloud regions using gcloud CLI...")
    try:
        command = ["gcloud", "compute", "regions", "list", "--format=json"]

        result = await asyncio.to_thread(
            subprocess.run,
            command,
            capture_output=True,
            text=True,
            check=True
        )

        regions_data = json.loads(result.stdout)
        region_names = [region['name'] for region in regions_data]
        logger.info(f"Successfully retrieved {len(region_names)} Google Cloud regions via gcloud CLI.")
        return region_names

    except FileNotFoundError:
        error_msg = "Error: 'gcloud' command not found. Please ensure the Google Cloud SDK is installed and configured in your system's PATH."
        logger.error(error_msg)
        raise HTTPException(status_code=500, detail=error_msg) # Propagate as HTTPException
    except subprocess.CalledProcessError as e:
        error_msg = f"An error occurred while executing the gcloud command. Stderr: {e.stderr}"
        logger.error(error_msg)
        raise HTTPException(status_code=500, detail=error_msg)
    except json.JSONDecodeError:
        error_msg = "Error: Could not parse the JSON output from the gcloud command."
        logger.error(error_msg)
        raise HTTPException(status_code=500, detail=error_msg)
    except Exception as e:
        logger.exception(f"An unexpected error occurred while listing regions: {e}")
        raise HTTPException(status_code=500, detail=f"An unexpected error occurred: {e}")