package main

import (
	"log/slog"
	"os"

	apprunner "github.com/psyb0t/common-go/app-runner"
	"github.com/psyb0t/servicepack/internal/app"
	_ "github.com/psyb0t/servicepack/internal/pkg/services" // Trigger service registration
	_ "github.com/psyb0t/slog-configurator"
	"github.com/spf13/cobra"
)

// go build -ldflags "-X main.appName=userservice".
//
//nolint:gochecknoglobals//need to be global bcuz ^.
var appName = "servicepack"

func main() {
	a := app.GetInstance()

	rootCmd := buildRootCommand(a)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRootCommand(a *app.App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: appName,
	}

	rootCmd.AddCommand(
		buildRunCommand(a),
	)

	return rootCmd
}

func buildRunCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the app",
		Run: func(_ *cobra.Command, _ []string) {
			if err := apprunner.Run(a); err != nil {
				slog.Error("apprunner.Run error", "error", err)
				os.Exit(1)
			}
		},
	}
}
