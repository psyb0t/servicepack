#!/bin/bash

set -e

# Check if service name is provided
if [ -z "$1" ]; then
    echo "Error: Service name is required"
    echo "Usage: $0 <service-name>"
    echo "Example: $0 myservice"
    exit 1
fi

SERVICE_NAME="$1"
SERVICE_DIR="internal/pkg/services/$SERVICE_NAME"

# Check if service exists
if [ ! -d "$SERVICE_DIR" ]; then
    echo "Error: Service '$SERVICE_NAME' does not exist at $SERVICE_DIR"
    exit 1
fi

# Remove service directory
echo "Removing service '$SERVICE_NAME'..."
rm -rf "$SERVICE_DIR"

echo "Service '$SERVICE_NAME' removed from $SERVICE_DIR"