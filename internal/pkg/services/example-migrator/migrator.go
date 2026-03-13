package examplemigrator

import (
	"context"
	"log/slog"

	exampledatabase "github.com/psyb0t/servicepack/internal/pkg/services/example-database"
)

const ServiceName = "example-migrator"

// ExampleMigrator demonstrates a one-shot service with
// dependency behavior. It depends on "example-database",
// runs its migrations, and exits cleanly.
// It's also an allowed failure because migration issues
// shouldn't take down the whole app.
type ExampleMigrator struct{}

func New() (*ExampleMigrator, error) {
	return &ExampleMigrator{}, nil
}

func (m *ExampleMigrator) Name() string {
	return ServiceName
}

// Dependencies makes ExampleMigrator implement the
// Dependent interface. Starts after example-database.
func (m *ExampleMigrator) Dependencies() []string {
	return []string{exampledatabase.ServiceName}
}

// IsAllowedFailure makes ExampleMigrator implement the
// AllowedFailure interface. Migration failures don't
// bring down the whole app.
func (m *ExampleMigrator) IsAllowedFailure() bool {
	return true
}

func (m *ExampleMigrator) Run(
	_ context.Context,
) error {
	slog.Info("running migrations",
		"service", ServiceName,
	)

	slog.Info("migrations completed",
		"service", ServiceName,
	)

	return nil
}

func (m *ExampleMigrator) Stop(
	_ context.Context,
) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
