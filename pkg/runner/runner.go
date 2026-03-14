package runner

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
)

var ErrShutdownTimeout = errors.New("shutdown timeout")

const (
	envVarNameShutdownTimeout = "APPRUNNER_SHUTDOWNTIMEOUT"
	defaultShutdownTimeout    = 10 * time.Second
)

type Runnable interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

type config struct {
	ShutdownTimeout time.Duration `env:"APPRUNNER_SHUTDOWNTIMEOUT"`
}

func Run(runnable Runnable) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &appRunner{
		runnable:        runnable,
		shutdownTimeout: cfg.ShutdownTimeout,
	}

	return r.run(ctx)
}

type appRunner struct {
	runnable        Runnable
	shutdownTimeout time.Duration
}

func getConfig() (*config, error) {
	cfg := &config{}

	gonfiguration.SetDefaults(map[string]any{
		envVarNameShutdownTimeout: defaultShutdownTimeout,
	})

	if err := gonfiguration.Parse(cfg); err != nil {
		return nil, ctxerrors.Wrap(
			err, "failed to parse app runner config",
		)
	}

	return cfg, nil
}

func (r *appRunner) run(ctx context.Context) error {
	sigCh := r.setupSignalHandling()
	errCh := make(chan error, 1)

	var wg sync.WaitGroup

	wg.Add(1)

	go r.runApp(ctx, &wg, errCh)

	shutdownErr := r.waitForShutdown(sigCh, errCh)

	return r.gracefulShutdown(
		ctx, &wg, shutdownErr,
	)
}

func (r *appRunner) setupSignalHandling() chan os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(
		sigCh,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	return sigCh
}

func (r *appRunner) runApp(
	ctx context.Context,
	wg *sync.WaitGroup,
	errCh chan error,
) {
	defer wg.Done()
	defer close(errCh)

	slog.Info("starting application")

	errCh <- r.runnable.Run(ctx)
}

func (r *appRunner) waitForShutdown(
	sigCh chan os.Signal,
	errCh chan error,
) error {
	select {
	case sig := <-sigCh:
		slog.Info("received signal",
			"signal", sig.String(),
		)

		return nil
	case err := <-errCh:
		if err != nil {
			slog.Error("application error",
				"error", err,
			)
		}

		return err
	}
}

func (r *appRunner) gracefulShutdown(
	ctx context.Context,
	wg *sync.WaitGroup,
	shutdownErr error,
) error {
	slog.Info("initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(
		ctx, r.shutdownTimeout,
	)
	defer cancel()

	stopErrCh := make(chan error, 1)

	wg.Go(func() {
		if err := r.runnable.Stop(ctx); err != nil {
			stopErrCh <- err
		}
	})

	doneCh := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case err := <-stopErrCh:
		return ctxerrors.Wrap(
			shutdownErr, err.Error(),
		)
	case <-shutdownCtx.Done():
		return r.handleShutdownTimeout(
			shutdownCtx, shutdownErr,
		)
	case <-doneCh:
		slog.Info("shutdown completed")
	}

	return shutdownErr
}

func (r *appRunner) handleShutdownTimeout(
	shutdownCtx context.Context,
	shutdownErr error,
) error {
	err := shutdownCtx.Err()
	if err == nil {
		return shutdownErr
	}

	if errors.Is(err, context.DeadlineExceeded) {
		slog.Error("shutdown timeout exceeded")

		return ErrShutdownTimeout
	}

	if shutdownErr != nil {
		return ctxerrors.Wrap(
			shutdownErr, err.Error(),
		)
	}

	return ctxerrors.Wrap(err, "shutdown error")
}
