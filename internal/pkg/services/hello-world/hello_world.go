package helloworld

import (
	"context"
	"time"

	"github.com/psyb0t/ctxscope"
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
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	ticker := time.NewTicker(5 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled, stopping service")

			return nil
		case <-ticker.C:
			logger.Info("hello world heartbeat")
		}
	}
}

func (h *HelloWorld) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
