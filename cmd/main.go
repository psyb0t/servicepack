package main

import (
	"log/slog"
	"os"

	"github.com/psyb0t/servicepack/internal/app"
	servicemanager "github.com/psyb0t/servicepack/internal/pkg/service-manager"
	"github.com/psyb0t/servicepack/internal/pkg/services"
	"github.com/psyb0t/servicepack/pkg/runner"
	_ "github.com/psyb0t/slog-configurator"
	"github.com/spf13/cobra"
)

// go build -ldflags "-X main.appName=userservice".
//
//nolint:gochecknoglobals//need to be global bcuz ^.
var appName = "servicepack"

func main() {
	services.Init()

	rootCmd := buildRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: appName,
	}

	rootCmd.AddCommand(buildRunCommand())

	rootCmd.AddCommand(
		servicemanager.GetInstance().Commands()...,
	)

	rootCmd.AddCommand(commands()...)

	return rootCmd
}

func buildRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the app",
		Run: func(_ *cobra.Command, _ []string) {
			a := app.GetInstance()
			if err := runner.Run(a); err != nil {
				slog.Error("runner.Run error", "error", err)
				os.Exit(1)
			}
		},
	}
}
