package main

import (
	"os"

	"github.com/joho/godotenv"
	apprunner "github.com/psyb0t/common-go/app-runner"
	_ "github.com/psyb0t/logrus-configurator"
	"github.com/psyb0t/servicepack/internal/app"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// go build -ldflags "-X main.appName=userservice".
//
//nolint:gochecknoglobals//need to be global bcuz ^.
var appName = "servicepack"

func main() {
	if err := godotenv.Load(); err != nil {
		logrus.Warnf("godotenv.Load error: %v", err)
	}

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
				logrus.Fatalf("apprunner.Run error: %v", err)
			}
		},
	}
}
