package exampleoptional

import (
	"context"
	"errors"
	"log/slog"
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
	_ context.Context,
) error {
	slog.Info("starting service",
		"service", ServiceName,
	)

	slog.Error("failing on purpose",
		"service", ServiceName,
	)

	return errOptional
}

func (o *ExampleOptional) Stop(
	_ context.Context,
) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
