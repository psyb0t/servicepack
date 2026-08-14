package examplemigrator

import (
	"context"

	"github.com/psyb0t/ctxscope"
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
			Run: func(cmd *cobra.Command, _ []string) {
				ctx := ctxscope.Set(
					cmd.Context(),
					ctxscope.Attr("service", ServiceName),
				)
				logger := ctxscope.GetLogger(ctx)

				logger.Info("running migrations up")
				logger.Info("migrations applied")
			},
		},
		{
			Use:   "down",
			Short: "Rollback the last migration",
			Run: func(cmd *cobra.Command, _ []string) {
				ctx := ctxscope.Set(
					cmd.Context(),
					ctxscope.Attr("service", ServiceName),
				)
				logger := ctxscope.GetLogger(ctx)

				logger.Info("rolling back migration")
				logger.Info("migration rolled back")
			},
		},
		{
			Use:   "status",
			Short: "Show migration status",
			Run: func(cmd *cobra.Command, _ []string) {
				ctx := ctxscope.Set(
					cmd.Context(),
					ctxscope.Attr("service", ServiceName),
				)

				ctxscope.GetLogger(ctx).Info(
					"migration status: all up to date",
				)
			},
		},
	}
}

func (m *ExampleMigrator) Run(
	ctx context.Context,
) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("running migrations")

	logger.Info("migrations completed")

	return nil
}

func (m *ExampleMigrator) Stop(
	ctx context.Context,
) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
