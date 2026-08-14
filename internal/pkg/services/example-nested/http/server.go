package server

import (
	"context"
	"time"

	"github.com/psyb0t/ctxscope"
)

const ServiceName = "example-nested-http"

// Server demonstrates a service in a nested directory.
// Both example-nested/http and example-nested/grpc use
// package name "server" - the codegen handles this by
// deriving unique import aliases from the directory path.
type Server struct{}

func New() (*Server, error) {
	return &Server{}, nil
}

func (s *Server) Name() string {
	return ServiceName
}

func (s *Server) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	ticker := time.NewTicker(10 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled, stopping service")

			return nil
		case <-ticker.C:
			logger.Info("heartbeat")
		}
	}
}

func (s *Server) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
