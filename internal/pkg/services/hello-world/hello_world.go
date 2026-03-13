package helloworld

import (
	"context"
	"log/slog"
	"time"
)

const ServiceName = "hello-world"

type HelloWorld struct{}

func New() (*HelloWorld, error) {
	return &HelloWorld{}, nil
}

func (h *HelloWorld) Name() string {
	return ServiceName
}

func (h *HelloWorld) Run(ctx context.Context) error {
	slog.Info("starting service", "service", ServiceName)

	ticker := time.NewTicker(5 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("context cancelled, stopping service", "service", ServiceName)

			return nil
		case <-ticker.C:
			slog.Info("Hello, World!")
		}
	}
}

func (h *HelloWorld) Stop(_ context.Context) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
