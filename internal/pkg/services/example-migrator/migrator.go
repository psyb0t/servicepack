package examplemigrator

import (
	"context"
	"log/slog"

	exampledatabase "github.com/psyb0t/servicepack/internal/pkg/services/example-database"
	"github.com/spf13/cobra"
)

const ServiceName = "example-migrator"

// ExampleMigrator demonstrates a one-shot service with
// dependency behavior and CLI commands. It depends on
// "example-database", runs its migrations, and exits
// cleanly. It also implements Commander to expose
// ./app example-migrator up|down|status commands.
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

// Commands makes ExampleMigrator implement the Commander
// interface. These are available as:
//
//	./app example-migrator up
//	./app example-migrator down
//	./app example-migrator status
func (m *ExampleMigrator) Commands() []*cobra.Command {
	return []*cobra.Command{
		{
			Use:   "up",
			Short: "Run all pending migrations",
			Run: func(_ *cobra.Command, _ []string) {
				slog.Info("running migrations up",
					"service", ServiceName,
				)

				slog.Info("migrations applied",
					"service", ServiceName,
				)
			},
		},
		{
			Use:   "down",
			Short: "Rollback the last migration",
			Run: func(_ *cobra.Command, _ []string) {
				slog.Info("rolling back migration",
					"service", ServiceName,
				)

				slog.Info("migration rolled back",
					"service", ServiceName,
				)
			},
		},
		{
			Use:   "status",
			Short: "Show migration status",
			Run: func(_ *cobra.Command, _ []string) {
				slog.Info("migration status: all up to date",
					"service", ServiceName,
				)
			},
		},
	}
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
