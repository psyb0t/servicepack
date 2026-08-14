package app

import (
	"context"
	"sync"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/goenv"
	servicemanager "github.com/psyb0t/servicepack/internal/pkg/service-manager"
)

var (
	instance *App      //nolint:gochecknoglobals
	once     sync.Once //nolint:gochecknoglobals
)

// HookFunc is a callback invoked during the app lifecycle.
// Hooks receive the app context and can launch goroutines from it.
type HookFunc func(ctx context.Context)

type App struct {
	wg             sync.WaitGroup
	cancel         context.CancelFunc
	cancelMu       sync.Mutex
	stopOnce       sync.Once
	serviceManager *servicemanager.ServiceManager
	preRunHooks    []HookFunc
	postStopHooks  []HookFunc
}

func GetInstance() *App {
	once.Do(func() {
		instance = newApp()
	})

	return instance
}

func newApp() *App {
	ctxscope.GetLogger(context.Background()).Debug("initializing app")

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

// OnPreRun registers a function that runs before services start.
// Hooks execute sequentially in registration order.
func (a *App) OnPreRun(fn HookFunc) {
	a.preRunHooks = append(a.preRunHooks, fn)
}

// OnPostStop registers a function that runs after all services stop.
// Hooks execute sequentially in registration order.
func (a *App) OnPostStop(fn HookFunc) {
	a.postStopHooks = append(a.postStopHooks, fn)
}

func (a *App) Run(ctx context.Context) error {
	ctxscope.GetLogger(ctx).Info("running app", "env", goenv.Get())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.cancelMu.Lock()
	a.cancel = cancel
	a.cancelMu.Unlock()

	// Defers run LIFO, so these three are registered in the REVERSE of the
	// order they must run in: Stop first (it cancels the context, which is
	// what lets the goroutine below finish), then wait for that goroutine to
	// actually exit, and only then close the channel it sends on.
	//
	// Registered the other way round, close(errCh) runs while its producer is
	// still live: a shutdown racing a non-nil error out of ServiceManager.Run
	// panics the process with "send on closed channel" during what should be a
	// graceful stop, and Stop lands last -- after the wait it was meant to
	// unblock. ServiceManager.Run already registers the same trio in this
	// order; this function used to disagree with it.
	errCh := make(chan error, 1)
	defer close(errCh)

	defer a.wg.Wait()

	defer func() {
		if err := a.Stop(ctx); err != nil {
			ctxscope.GetLogger(ctx).Error("failed to stop app", "err", err)
		}
	}()

	for _, hook := range a.preRunHooks {
		hook(ctx)
	}

	a.wg.Go(func() {
		if err := a.serviceManager.Run(ctx); err != nil {
			errCh <- err
		}
	})

	select {
	case <-ctx.Done():
		ctxscope.GetLogger(ctx).Debug("app context done")

		return nil
	case err := <-errCh:
		ctxscope.GetLogger(ctx).Error("app run error", "err", err)

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
		ctxscope.GetLogger(ctx).Info("stopping app")
		defer ctxscope.GetLogger(ctx).Info("stopped app")

		a.serviceManager.Stop(ctx)
		a.wg.Wait()

		for _, hook := range a.postStopHooks {
			hook(ctx)
		}
	})

	return nil
}
