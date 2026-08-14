package runner

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gonfiguration"
)

var ErrShutdownTimeout = errors.New("shutdown timeout")

type Runnable interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

type config struct {
	ShutdownTimeout time.Duration `default:"10s" env:"RUNNER_SHUTDOWNTIMEOUT"`
}

func Run(runnable Runnable) error {
	return RunContext(context.Background(), runnable)
}

// RunContext runs a lifecycle with ctx as the application context's parent.
func RunContext(ctx context.Context, runnable Runnable) error {
	cfg, err := getConfig()
	if err != nil {
		return ctxerrors.Wrap(err, "get runner config")
	}

	ctx, cancel := context.WithCancel(ctx)
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

	if err := gonfiguration.Parse(cfg); err != nil {
		return nil, ctxerrors.Wrap(
			err, "failed to parse runner config",
		)
	}

	return cfg, nil
}

func (r *appRunner) run(ctx context.Context) error {
	sigCh := r.setupSignalHandling()
	defer signal.Stop(sigCh)

	errCh := make(chan error, 1)

	var wg sync.WaitGroup

	wg.Add(1)

	go r.runApp(ctx, &wg, errCh)

	shutdownErr := r.waitForShutdown(ctx, sigCh, errCh)

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

	ctxscope.GetLogger(ctx).Info("starting application")

	errCh <- r.runnable.Run(ctx)
}

func (r *appRunner) waitForShutdown(
	ctx context.Context,
	sigCh chan os.Signal,
	errCh chan error,
) error {
	select {
	case <-ctx.Done():
		ctxscope.GetLogger(ctx).Debug("runner context cancelled")

		return nil
	case sig := <-sigCh:
		ctxscope.GetLogger(ctx).Info("received signal",
			"signal", sig.String(),
		)

		return nil
	case err := <-errCh:
		// The wrap stays inside the non-nil branch: ctxerrors logs an ERROR of
		// its own when handed a nil, so wrapping unconditionally would emit a
		// bogus "Trying to wrap a nil error" line on every clean shutdown that
		// arrives through this channel.
		if err != nil {
			ctxscope.GetLogger(ctx).Error("application error",
				"err", err,
			)

			return ctxerrors.Wrap(err, "run application")
		}

		return nil
	}
}

func (r *appRunner) gracefulShutdown(
	ctx context.Context,
	wg *sync.WaitGroup,
	shutdownErr error,
) error {
	ctxscope.GetLogger(ctx).Info("initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx), r.shutdownTimeout,
	)
	defer cancel()

	stopErrCh := make(chan error, 1)

	wg.Go(func() {
		if err := r.runnable.Stop(shutdownCtx); err != nil {
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
		return ctxerrors.Wrap(errors.Join(shutdownErr, err), "stop application")
	case <-shutdownCtx.Done():
		return r.handleShutdownTimeout(
			shutdownCtx, shutdownErr,
		)
	case <-doneCh:
		ctxscope.GetLogger(ctx).Info("shutdown completed")
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
		ctxscope.GetLogger(shutdownCtx).Error("shutdown timeout exceeded")

		return ErrShutdownTimeout
	}

	if shutdownErr != nil {
		return ctxerrors.Wrap(
			shutdownErr, err.Error(),
		)
	}

	return ctxerrors.Wrap(err, "shutdown error")
}
