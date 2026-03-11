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

#!/bin/bash
set -e

# Save original directory and ensure we return to it on exit (even on error)
ORIG_DIR="$(pwd)"
trap 'echo "Returning to original directory: $ORIG_DIR"; cd "$ORIG_DIR"' EXIT

# Determine the source directory to copy generated files back
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Parse command line flags
RUN_COVERAGE=false
RUN_GO=false
RUN_PYTHON=false
RUN_ANGULAR=false
BASE_DIR="$(pwd)"

usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Description:"
  echo "  This script builds and tests code for the DPI Becknonix project."
  echo "  It runs builds and tests for Go, Python, and Angular components."
  echo "  Additionally, it can generate coverage reports and copy generated files (e.g. go.mod, package.json) back to the source directory."
  echo ""
  echo "Options:"
  echo "  -h, --help     Show this help message and exit"
  echo "  -b, -b=<dir>   Specify the base directory (default: current directory)"
  echo "  -c             Run tests with coverage tracking"
  echo "  -g             Run Go build and tests"
  echo "  -p             Run Python build and tests"
  echo "  -a             Run Angular build and tests"
  echo ""
  echo "If no language flags (-g, -p, -a) are provided, the script defaults to running all of them."
}

while [[ "$#" -gt 0 ]]; do
  case $1 in
    -h|--help) usage; exit 0 ;;
    -b) BASE_DIR="$2"; shift 2 ;;
    -b=*) BASE_DIR="${1#*=}"; shift ;;
    -c) RUN_COVERAGE=true; shift ;;
    -g) RUN_GO=true; shift ;;
    -p) RUN_PYTHON=true; shift ;;
    -a) RUN_ANGULAR=true; shift ;;
    *) echo "Unknown parameter passed: $1"; usage; exit 1 ;;
  esac
done

if [ ! -d "$BASE_DIR" ]; then
  echo "Error: BASE_DIR '$BASE_DIR' does not exist."
  exit 1
fi
BASE_DIR="$(cd "${BASE_DIR}" && pwd)"

# Change to the base directory for the remainder of the script
echo "Changing to base directory: $BASE_DIR"
cd "$BASE_DIR"

# If no specific language flag was provided, default to all
if [ "$RUN_GO" = false ] && [ "$RUN_PYTHON" = false ] && [ "$RUN_ANGULAR" = false ]; then
  RUN_GO=true
  RUN_PYTHON=true
  RUN_ANGULAR=true
fi

if [ "$RUN_GO" = true ]; then
  echo "========================================"
  echo "1. Building and testing Go code"
  echo "========================================"
  cd "${BASE_DIR}"
  echo "Running go mod tidy..."
  go mod tidy
  echo "Copying back go.mod and go.sum..."
  cp go.mod "${SRC_DIR}/go.mod"
  cp go.sum "${SRC_DIR}/go.sum"
  echo "Running go build..."
  go build ./...
  echo "Running go test..."
  if [ "$RUN_COVERAGE" = true ]; then
    go test -coverprofile=coverage.out ./...
  else
    go test ./...
  fi
fi

if [ "$RUN_PYTHON" = true ]; then
  echo "========================================"
  echo "2. Building and testing Python Backend"
  echo "========================================"
  cd "${BASE_DIR}/deploy/onix_installer/backend"
  echo "Setting up virtual environment..."
  python3 -m venv venv
  source venv/bin/activate
  echo "Using Python version:"
  python --version
  echo "Installing dependencies..."
  python -m pip install --upgrade pip pip-tools --index-url=https://pypi.org/simple
  echo "Generating requirements.txt..."
  pip-compile --generate-hashes requirements.in --index-url=https://pypi.org/simple
  echo "Copying back requirements.in and requirements.txt..."
  cp requirements.in "${SRC_DIR}/deploy/onix_installer/backend/requirements.in"
  cp requirements.txt "${SRC_DIR}/deploy/onix_installer/backend/requirements.txt"
  echo "Installing requirements..."
  pip install -r requirements.txt --require-hashes --index-url=https://pypi.org/simple
  echo "Installing pytest..."
  pip install pytest httpx pytest-cov --index-url=https://pypi.org/simple
  echo "Running pytest..."
  if [ "$RUN_COVERAGE" = true ]; then
    PYTHONPATH=. pytest --cov=core --cov=services --cov=installer_kit --cov-report=term-missing
  else
    PYTHONPATH=. pytest
  fi
  deactivate
fi

if [ "$RUN_ANGULAR" = true ]; then
  echo "========================================"
  echo "3. Building and testing Angular Frontend"
  echo "========================================"
  cd "${BASE_DIR}/deploy/onix_installer/frontend"
  echo "Installing npm dependencies..."
  npm install --registry=https://registry.npmjs.org/
  echo "Copying back package.json and package-lock.json..."
  cp package.json "${SRC_DIR}/deploy/onix_installer/frontend/package.json"
  cp package-lock.json "${SRC_DIR}/deploy/onix_installer/frontend/package-lock.json"
  echo "Running ng build..."
  node node_modules/@angular/cli/bin/ng.js build
  echo "Running ng test..."
  # Use ChromeHeadless and disable watch mode for script execution
  if [ "$RUN_COVERAGE" = true ]; then
  npm test -- --watch=false --browsers=ChromeHeadless --code-coverage
  else
  npm test -- --watch=false --browsers=ChromeHeadless
  fi
fi

if [ "$RUN_COVERAGE" = true ]; then
  echo "========================================"
  echo "             COVERAGE SUMMARY           "
  echo "========================================"
  if [ "$RUN_GO" = true ]; then
    echo "--- Go Backend ---"
    cd "${BASE_DIR}"
    if [ -f "coverage.out" ]; then
      go tool cover -func=coverage.out | grep total: || echo "Could not parse go coverage."
    else
      echo "Coverage file coverage.out not found."
    fi
    echo ""
  fi

  if [ "$RUN_PYTHON" = true ]; then
    echo "--- Python Backend ---"
    cd "${BASE_DIR}/deploy/onix_installer/backend"
    if [ -f ".coverage" ]; then
      source venv/bin/activate
      python -m coverage report | grep TOTAL || echo "Could not parse python coverage."
      deactivate
    else
      echo "Python .coverage data not found."
    fi
    echo ""
  fi

  if [ "$RUN_ANGULAR" = true ]; then
    echo "--- Angular Frontend ---"
    cd "${BASE_DIR}/deploy/onix_installer/frontend"
    if [ -d "coverage" ]; then
      echo "Angular HTML coverage reports generated in: deploy/onix_installer/frontend/coverage"
    else
      echo "Angular coverage report not found."
    fi
  fi
  echo "========================================"
fi

echo "All builds and tests completed successfully!"
echo "========================================"
