# Stage 1: Build the Angular frontend
FROM node:20 as frontend-builder
WORKDIR /app/frontend
# Copy frontend dependency definitions
COPY deploy/onix-installer/frontend/package.json deploy/onix-installer/frontend/package-lock.json ./
# Install dependencies
RUN npm ci
# Copy the frontend source code
COPY deploy/onix-installer/frontend/ .
# Build the Angular application
RUN npm run build -- --configuration production

# Stage 2: Final Image with Python Backend and Tools
FROM python:3.12-slim

# Set environment variables for non-interactive installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies and tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    gnupg \
    lsb-release \
    git \
    unzip \
    iptables \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Google Cloud SDK, Terraform, Helm, Kubectl, and Docker
RUN curl -sSLo /usr/share/keyrings/google-cloud-key.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/google-cloud-key.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl -fsSL https://apt.releases.hashicorp.com/gpg | gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list && \
    install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    apt-get install -y \
        google-cloud-cli \
        terraform=1.5.7-1 \
        docker-ce \
        docker-ce-cli \
        containerd.io && \
    curl -LO "https://dl.k8s.io/release/v1.28.3/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \
    rm get_helm.sh && \
    rm -rf /var/lib/apt/lists/*

# Copy backend requirements and install Python dependencies
WORKDIR /app
COPY deploy/onix-installer/backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy backend code
COPY deploy/onix-installer/backend/ /app/

# Copy built frontend assets from the browser subdirectory
COPY --from=frontend-builder /app/frontend/dist/frontend-test/browser /app/static

# Copy and set permissions for the entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose port and define entrypoint
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
