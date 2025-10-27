#!/bin/bash
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


# This script attempts to install all prerequisites listed in the README.md
# for the BECKN Adapter Installer. It supports Debian/Ubuntu, CentOS/RHEL, and macOS.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Helper Functions ---
# Function to detect the operating system and package manager
detect_os() {
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "macOS"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        if [[ "$ID" == "ubuntu" || "$ID" == "debian" ]]; then
            echo "linux"
        elif [[ "$ID" == "centos" || "$ID" == "rhel" || "$ID" == "fedora" ]]; then
            echo "linux"
        else
            echo "Unsupported Linux distribution: $ID"
            exit 1
        fi
    else
        echo "Unsupported operating system."
        exit 1
    fi
}

# --- Main Script ---

OS=$(detect_os)
echo "Detected OS: $OS"

echo "--- Installing Prerequisites ---"

if [[ "$OS" == "linux" ]]; then
    if command -v apt-get &> /dev/null; then
        echo "Updating package list..."
        sudo apt-get update -y
        echo "Installing required packages (jq, postgresql-client, unzip)..."
        sudo apt-get install -y jq postgresql-client unzip
        echo "--- Installing gcloud components ---"
        sudo apt-get install -y kubectl google-cloud-cli-gke-gcloud-auth-plugin
    elif command -v yum &> /dev/null; then
        echo "Updating package list..."
        sudo yum update -y
        echo "Installing required packages (jq, postgresql, unzip)..."
        sudo yum install -y jq postgresql unzip
        echo "--- Installing gcloud components ---"
        gcloud components install -q kubectl gke-gcloud-auth-plugin
    fi
elif [[ "$OS" == "macOS" ]]; then
    if ! command -v brew &> /dev/null; then
        echo "Homebrew not found. Please install it from https://brew.sh/"
        exit 1
    fi
    echo "Updating Homebrew..."
    brew update
    echo "Installing required packages (jq, postgresql, unzip)..."
    brew install jq postgresql unzip
    # Add PostgreSQL to path for this session
    export PATH="/opt/homebrew/opt/postgresql/bin:$PATH"
    echo "--- Installing gcloud components ---"
    gcloud components install -q kubectl gke-gcloud-auth-plugin
fi

echo "--- Installing Terraform ---"
TERRAFORM_VERSION="1.5.7"
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
    ARCH="arm64"
fi
if [[ "$OS" == "linux" ]]; then
curl -L -o terraform.zip "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${OS,,}_${ARCH}.zip"
unzip -o terraform.zip
sudo mv terraform /usr/local/bin/
rm terraform.zip
fi
elif [[ "$OS" == "macOS" ]]; then
    brew install tfenv
    tfenv install 1.5.7
    tfenv use 1.5.7
fi


echo "--- Installing Helm ---"
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
rm get_helm.sh

echo "--- Installing Python 3 and pip ---"
if [[ "$OS" == "linux" ]]; then
    if command -v apt-get &> /dev/null; then
        sudo add-apt-repository ppa:deadsnakes/ppa -y
        sudo apt-get update
        sudo apt-get install -y python3.12 python3.12-venv python3.12-dev
    elif command -v yum &> /dev/null; then
        sudo yum install -y python3.12
    fi
elif [[ "$OS" == "macOS" ]]; then
    if command -v brew &> /dev/null; then
        brew install pyenv
        pyenv install 3.12.0
        pyenv global 3.12.0
    fi
fi

echo "--- Installing Node.js (LTS) and Angular CLI via nvm ---"
curl -o nvm_install.sh https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh
chmod 700 nvm_install.sh
./nvm_install.sh
rm nvm_install.sh
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
nvm install --lts
npm install -g @angular/cli

echo "------------------------------------"
echo "✅ All prerequisites have been installed."
echo "Please restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc) for the changes to take effect."
echo "After restarting, you can verify the installations by running the commands in the README."