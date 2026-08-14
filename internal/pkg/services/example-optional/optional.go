package exampleoptional

import (
	"context"
	"errors"

	"github.com/psyb0t/ctxscope"
)

const ServiceName = "example-optional"

var errOptional = errors.New("optional service failed")

// ExampleOptional demonstrates allowed-failure behavior.
// It fails immediately but is marked as an allowed
// failure, so it doesn't take down the rest of the app.
// Useful pattern for non-critical services like metrics
// exporters, cache warmers, or analytics collectors.
type ExampleOptional struct{}

func New() (*ExampleOptional, error) {
	return &ExampleOptional{}, nil
}

func (o *ExampleOptional) Name() string {
	return ServiceName
}

// IsAllowedFailure makes ExampleOptional implement the
// AllowedFailure interface. Its failure won't bring down
// the service manager.
func (o *ExampleOptional) IsAllowedFailure() bool {
	return true
}

func (o *ExampleOptional) Run(
	ctx context.Context,
) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	logger.Error("failing on purpose")

	return errOptional
}

func (o *ExampleOptional) Stop(
	ctx context.Context,
) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
