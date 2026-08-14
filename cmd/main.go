package main

import (
	"context"
	"os"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/servicepack/internal/app"
	servicemanager "github.com/psyb0t/servicepack/internal/pkg/service-manager"
	"github.com/psyb0t/servicepack/internal/pkg/services"
	"github.com/psyb0t/servicepack/pkg/runner"
	_ "github.com/psyb0t/slogging/slogconf"
	"github.com/spf13/cobra"
)

// go build -ldflags "-X main.appName=userservice -X main.buildCommit=abc123".
//
//nolint:gochecknoglobals//need to be global bcuz ^.
var (
	appName     = "servicepack"
	buildCommit string
)

const (
	scopeKeyBinary = "binary"
	scopeKeyCommit = "commit"
)

func main() {
	setProcessScope(appName, buildCommit)
	services.Init()

	rootCmd := buildRootCommand()
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		ctxscope.GetLogger(context.Background()).Error(
			"command failed",
			"err", err,
		)

		os.Exit(1)
	}
}

func setProcessScope(binary, commit string) {
	attrs := []ctxscope.Attribute{ctxscope.Attr(scopeKeyBinary, binary)}

	if commit == "" {
		ctxscope.RemoveGlobal(scopeKeyCommit)
	} else {
		attrs = append(attrs, ctxscope.Attr(scopeKeyCommit, commit))
	}

	ctxscope.SetGlobal(attrs...)
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app.GetInstance()
			if err := runner.RunContext(cmd.Context(), a); err != nil {
				return ctxerrors.Wrap(err, "run application")
			}

			return nil
		},
	}
}
