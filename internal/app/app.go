package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/goenv"
	servicemanager "github.com/psyb0t/servicepack/internal/pkg/service-manager"
	"github.com/psyb0t/servicepack/internal/pkg/services"
)

var (
	instance *App      //nolint:gochecknoglobals
	once     sync.Once //nolint:gochecknoglobals
)

type App struct {
	wg             sync.WaitGroup
	cancel         context.CancelFunc
	cancelMu       sync.Mutex
	stopOnce       sync.Once
	serviceManager *servicemanager.ServiceManager
}

func GetInstance() *App {
	once.Do(func() {
		instance = newApp()
	})

	return instance
}

func newApp() *App {
	slog.Debug("initializing app")

	services.Init()

	return &App{
		serviceManager: servicemanager.GetInstance(),
	}
}

// resetInstance resets the singleton instance for testing purposes.
func resetInstance() {
	once = sync.Once{}
	instance = nil
	// Also reset ServiceManager singleton
	servicemanager.ResetInstance()
}

func (a *App) Run(ctx context.Context) error {
	slog.Info("running app", "env", goenv.Get())

	ctx, cancel := context.WithCancel(ctx)

	a.cancelMu.Lock()
	a.cancel = cancel
	a.cancelMu.Unlock()

	defer func() {
		if err := a.Stop(ctx); err != nil {
			slog.Error("failed to stop app", "error", err)
		}
	}()
	defer a.wg.Wait()

	errCh := make(chan error, 1)
	defer close(errCh)

	a.wg.Go(func() {
		if err := a.serviceManager.Run(ctx); err != nil {
			errCh <- err
		}
	})

	select {
	case <-ctx.Done():
		slog.Debug("app context done")

		return nil
	case err := <-errCh:
		slog.Error("app run error", "error", err)

		return ctxerrors.Wrap(err, "failed to run app")
	}
}

func (a *App) Stop(ctx context.Context) error {
	a.cancelMu.Lock()

	if a.cancel != nil {
		a.cancel()
	}

	a.cancelMu.Unlock()

	a.stopOnce.Do(func() {
		slog.Info("stopping app")
		defer slog.Info("stopped app")

		a.serviceManager.Stop(ctx)
		a.wg.Wait()
	})

	return nil
}
