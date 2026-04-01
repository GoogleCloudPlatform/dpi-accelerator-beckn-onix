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

import os
import re

# Regex to clean ANSI escape codes from script output
ANSI_ESCAPE_PATTERN = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')

# Base directory of the application
# Assumes this file is in backend/core and the root is 'backend'
BASE_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))

# Path to the installer_kit directory (relative to BASE_DIR)
INSTALLER_KIT_PATH = os.path.join(BASE_DIR, "installer_kit")

# The working directory for Terraform operations
TERRAFORM_DIRECTORY = os.path.join(INSTALLER_KIT_PATH, "terraform")

TEMPLATE_DIRECTORY = os.path.join(INSTALLER_KIT_PATH, "templates")

# Paths to the specific installer scripts
INFRA_SCRIPT_PATH = os.path.join(INSTALLER_KIT_PATH, "installer_scripts", "deploy-infra.sh")
APP_SCRIPT_PATH = os.path.join(INSTALLER_KIT_PATH, "installer_scripts", "deploy-app.sh")

GENERATED_CONFIGS_DIR = os.path.join(os.path.dirname(BASE_DIR), 'generated_configs')
