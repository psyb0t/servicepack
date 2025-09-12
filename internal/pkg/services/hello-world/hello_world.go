package helloworld

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
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
	logrus.Infof("Starting %s service", ServiceName)

	ticker := time.NewTicker(5 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Context cancelled, stopping %s service", ServiceName)

			return nil
		case <-ticker.C:
			logrus.Info("Hello, World!")
		}
	}
}

func (h *HelloWorld) Stop(_ context.Context) error {
	logrus.Infof("Stopping %s service", ServiceName)

	return nil
}
