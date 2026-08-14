#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

trap 'error "command failed line=$LINENO exit=$?"' ERR

if [[ "$#" -ne 1 || -z "$1" ]]; then
	error "Service name is required"
	printf 'Usage: %s <service-name>\n' "$0" >&2
	printf 'Example: %s myservice\n' "$0" >&2
	exit 1
fi

readonly SERVICE_NAME="$1"

if [[ ! "$SERVICE_NAME" =~ ^[a-z][a-z0-9-]*$ ||
	"$SERVICE_NAME" == *--* ||
	"$SERVICE_NAME" == *- ]]; then
	error "Service name must use lowercase letters, digits, and single hyphens"
	exit 1
fi

readonly SERVICE_DIR="internal/pkg/services/$SERVICE_NAME"

PACKAGE_NAME="${SERVICE_NAME//-/}"
PACKAGE_NAME="${PACKAGE_NAME,,}"
readonly PACKAGE_NAME
readonly SERVICE_FILE="$SERVICE_DIR/${PACKAGE_NAME}.go"

# Check if service already exists
if [[ -d "$SERVICE_DIR" ]]; then
	error "Service '$SERVICE_NAME' already exists at $SERVICE_DIR"
	exit 1
fi

section "Creating Service"
info "Service name: '$SERVICE_NAME'"
mkdir -p "$SERVICE_DIR"

struct_name=""
IFS='-' read -r -a name_parts <<<"$SERVICE_NAME"
for name_part in "${name_parts[@]}"; do
	struct_name+="${name_part^}"
done
readonly STRUCT_NAME="$struct_name"

# Generate the service file
cat >"$SERVICE_FILE" <<EOF
package $PACKAGE_NAME

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gonfiguration"
)

const ServiceName = "$SERVICE_NAME"

type Config struct {
	Value string \`env:"${PACKAGE_NAME^^}_VALUE" default:"default-value"\`
}

type $STRUCT_NAME struct {
	config Config
}

func New() (*$STRUCT_NAME, error) {
	cfg := Config{}

	if err := gonfiguration.Parse(&cfg); err != nil {
		return nil, ctxerrors.Wrap(err, "parse $SERVICE_NAME config")
	}

	return &$STRUCT_NAME{
		config: cfg,
	}, nil
}

func (s *$STRUCT_NAME) Name() string {
	return ServiceName
}

func (s *$STRUCT_NAME) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	<-ctx.Done()
	logger.Info("service context cancelled")

	return nil
}

func (s *$STRUCT_NAME) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
EOF

success "Service '$SERVICE_NAME' created at $SERVICE_FILE"

# Regenerate service registration
info "Regenerating service registration..."
bash "$SCRIPT_DIR/service_registration.sh"

section "Next Steps"
printf '1. Implement the service logic in the Run() method\n' >&2
printf '2. Your service will automatically start when the app runs.\n' >&2
