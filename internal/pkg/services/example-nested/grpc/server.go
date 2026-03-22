package server

import (
	"context"
	"log/slog"
	"time"
)

const ServiceName = "example-nested-grpc"

// Server demonstrates a service in a nested directory
// that shares its package name with a sibling. Both
// example-nested/http and example-nested/grpc are
// package "server" - the codegen derives unique import
// aliases (examplenestedhttpserver, examplenestedgrpcserver)
// from the directory path.
type Server struct{}

func New() (*Server, error) {
	return &Server{}, nil
}

func (s *Server) Name() string {
	return ServiceName
}

func (s *Server) Run(ctx context.Context) error {
	slog.Info("starting service", "service", ServiceName)

	ticker := time.NewTicker(10 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"context cancelled, stopping service",
				"service", ServiceName,
			)

			return nil
		case <-ticker.C:
			slog.Info("heartbeat",
				"service", ServiceName,
			)
		}
	}
}

func (s *Server) Stop(_ context.Context) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
