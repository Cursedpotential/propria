#!/bin/bash

echo "🚀 Starting system setup..."

# Install Docker
if ! command -v docker &> /dev/null
then
    curl -fsSL https://get.docker.com | sh
fi

# Install Docker Compose
if ! command -v docker-compose &> /dev/null
then
    apt-get install docker-compose -y
fi

# Install Tailscale
if ! command -v tailscale &> /dev/null
then
    curl -fsSL https://tailscale.com/install.sh | sh
fi

# Connect to Tailscale
echo "🔌 Connecting to Tailscale..."
tailscale up --authkey ${TAILSCALE_AUTHKEY}

# Start Docker services
docker-compose -f docker/docker-compose.backend.yml up -d

echo "✅ All services started."
