#!/bin/sh
set -e

# Start Docker daemon if not running and we have privileges
if ! docker info > /dev/null 2>&1; then
    if [ -f /var/run/docker.pid ]; then
        rm /var/run/docker.pid
    fi
    echo "Starting Docker daemon..."
    dockerd &
    # Wait for Docker to start
    echo "Waiting for Docker to start..."
    while ! docker info > /dev/null 2>&1; do
        sleep 1
    done
    echo "Docker started."
else
    echo "Docker daemon already running or socket mounted."
fi

echo "Starting Application..."

# Start Uvicorn
exec uvicorn main:app --host 0.0.0.0 --port 8080
