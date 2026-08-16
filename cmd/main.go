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

// go build -ldflags "-X main.appName=userservice -X main.buildCommit=abc123 -X main.buildVersion=v1.2.3".
//
//nolint:gochecknoglobals//need to be global bcuz ^.
var (
	appName      = "servicepack"
	buildCommit  string
	buildVersion = "dev"
)

const (
	scopeKeyBinary  = "binary"
	scopeKeyCommit  = "commit"
	scopeKeyVersion = "version"
)

func main() {
	setProcessScope(appName, buildCommit, buildVersion)
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

func setProcessScope(binary, commit, version string) {
	attrs := []ctxscope.Attribute{ctxscope.Attr(scopeKeyBinary, binary)}

	if commit == "" {
		ctxscope.RemoveGlobal(scopeKeyCommit)
	} else {
		attrs = append(attrs, ctxscope.Attr(scopeKeyCommit, commit))
	}

	if version == "" {
		ctxscope.RemoveGlobal(scopeKeyVersion)
	} else {
		attrs = append(attrs, ctxscope.Attr(scopeKeyVersion, version))
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
