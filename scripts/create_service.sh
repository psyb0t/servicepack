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
SERVICE_FILE="$SERVICE_DIR/${SERVICE_NAME}.go"

# Check if service already exists
if [ -d "$SERVICE_DIR" ]; then
    echo "Error: Service '$SERVICE_NAME' already exists at $SERVICE_DIR"
    exit 1
fi

# Create service directory
echo "Creating service '$SERVICE_NAME'..."
mkdir -p "$SERVICE_DIR"

# Convert service name to proper Go struct name
# my-service -> MyService
STRUCT_NAME=$(echo "$SERVICE_NAME" | sed 's/-/ /g' | sed 's/\b\w/\U&/g' | sed 's/ //g')

# Generate the service file
cat > "$SERVICE_FILE" << EOF
package $SERVICE_NAME

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
	"github.com/sirupsen/logrus"
)

const serviceName = "$SERVICE_NAME"

type Config struct {
	Value string \`env:"${SERVICE_NAME^^}_VALUE"\`
}

type $STRUCT_NAME struct{
	config Config
}

func New() (*$STRUCT_NAME, error) {
	cfg := Config{}
	
	gonfiguration.SetDefaults(map[string]any{
		"${SERVICE_NAME^^}_VALUE": "default-value",
	})
	
	if err := gonfiguration.Parse(&cfg); err != nil {
		return nil, ctxerrors.Wrap(err, "failed to parse $SERVICE_NAME config")
	}
	
	return &$STRUCT_NAME{
		config: cfg,
	}, nil
}

func (s *$STRUCT_NAME) Name() string {
	return serviceName
}

func (s *$STRUCT_NAME) Run(ctx context.Context) error {
	logrus.Infof("Starting %s service", serviceName)
	panic("TODO: Implement $SERVICE_NAME service logic")
}

func (s *$STRUCT_NAME) Stop(_ context.Context) error {
	logrus.Infof("Stopping %s service", serviceName)
	return nil
}
EOF

echo "Service '$SERVICE_NAME' created at $SERVICE_FILE"
echo ""
echo "Next steps:"
echo "1. Implement the service logic in the Run() method"
echo "2. Your service will automatically start when the app runs!"
