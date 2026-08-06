package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	servicemanager "github.com/psyb0t/servicepack/internal/pkg/service-manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Generous on purpose: this bounds only a FAILING test, and a consumer's
	// machine may be running a browser suite and image builds alongside it.
	servicesRunningTimeout = 5 * time.Second
	servicesRunningPoll    = time.Millisecond
)

// waitForRunningServices blocks until every mock reports IsRunning.
//
// This replaces a fixed `time.Sleep(20 * time.Millisecond)` that gated the
// assertion on start-up having happened. 20ms is ample on an idle machine and
// nowhere near enough on a loaded one, so the assertion failed in a downstream
// project's full-suite run and reported a service-startup bug that did not
// exist. Waiting on the condition itself cannot report that lie.
func waitForRunningServices(
	t *testing.T,
	services []*servicemanager.MockService,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		for _, svc := range services {
			if !svc.IsRunning() {
				return false
			}
		}

		return true
	}, servicesRunningTimeout, servicesRunningPoll,
		"not all services reported running")
}

// createTestApp creates an app with mock services instead of real ones.
func createTestApp() *App {
	// Reset and clear the service manager to avoid loading real services
	servicemanager.ResetInstance()
	servicemanager.GetInstance().ClearServices()

	app := &App{
		serviceManager: servicemanager.GetInstance(),
	}

	app.setupTestServices()

	return app
}

func (a *App) setupTestServices() {
	// Add minimal mock services for app testing
	a.serviceManager.Add(
		servicemanager.NewTestService("TestService1"),
		servicemanager.NewTestService("TestService2"),
	)
}

func TestApp_Run(t *testing.T) {
	// Reset singleton before each test
	testCases := []struct {
		name        string
		setupFunc   func() (context.Context, context.CancelFunc)
		runFunc     func(t *testing.T, app *App, ctx context.Context)
		expectError bool
	}{
		{
			name: "context cancellation",
			setupFunc: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately to test graceful shutdown

				return ctx, cancel
			},
			runFunc: func(t *testing.T, app *App, ctx context.Context) {
				t.Helper()

				err := app.Run(ctx)
				assert.NoError(t, err)
			},
			expectError: false,
		},
		{
			name: "stop via done channel",
			setupFunc: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			runFunc: func(t *testing.T, app *App, ctx context.Context) {
				t.Helper()
				// Start app in goroutine
				done := make(chan error, 1)

				go func() {
					done <- app.Run(ctx)
				}()

				// Give app time to start
				time.Sleep(10 * time.Millisecond)

				// Stop the app
				err := app.Stop(ctx)
				assert.NoError(t, err)

				// Wait for run to complete
				select {
				case err := <-done:
					assert.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("app.Run() did not complete within timeout")
				}
			},
			expectError: false,
		},
		{
			name: "timeout context",
			setupFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(
					context.Background(), 50*time.Millisecond,
				)
			},
			runFunc: func(t *testing.T, app *App, ctx context.Context) {
				t.Helper()

				err := app.Run(ctx)
				assert.NoError(t, err)
			},
			expectError: false,
		},
		{
			name: "service error propagation",
			setupFunc: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			runFunc: func(t *testing.T, _ *App, ctx context.Context) {
				t.Helper()
				// Create app with failing service
				servicemanager.ResetInstance()
				servicemanager.GetInstance().ClearServices()

				failingApp := &App{
					serviceManager: servicemanager.GetInstance(),
				}
				// Add a service that returns an error
				failingSvc := servicemanager.NewMockService("failing").
					WithRunError(assert.AnError)
				failingApp.serviceManager.Add(failingSvc)

				err := failingApp.Run(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to run app")
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetInstance()

			app := createTestApp()

			ctx, cancel := tc.setupFunc()
			defer cancel()

			tc.runFunc(t, app, ctx)
		})
	}
}

func TestApp_Stop(t *testing.T) {
	t.Run("stop once", func(t *testing.T) {
		resetInstance()

		app := createTestApp()

		ctx := context.Background()

		// First stop should work
		err := app.Stop(ctx)
		assert.NoError(t, err)

		// Second stop should also work (sync.Once behavior)
		err = app.Stop(ctx)
		assert.NoError(t, err)
	})
}

func TestApp_GetInstance(t *testing.T) {
	// Reset singleton before test
	resetInstance()

	t.Run("successful app instance creation", func(t *testing.T) {
		// Use createTestApp to avoid calling services.Init()
		app := createTestApp()
		assert.NotNil(t, app)
		assert.NotNil(t, app.serviceManager)
		assert.NotNil(t, app.serviceManager)

		// Manually set the singleton to test singleton behavior
		instance = app

		once.Do(func() {}) // Mark as initialized

		// Test that subsequent calls return the same instance
		app2 := GetInstance()
		assert.Same(t, app, app2)
	})
}

func TestApp_RunAndStop_Integration(t *testing.T) {
	t.Run("full lifecycle", func(t *testing.T) {
		resetInstance()

		app := createTestApp()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start app in goroutine
		runDone := make(chan error, 1)

		go func() {
			runDone <- app.Run(ctx)
		}()

		// Let app start
		time.Sleep(50 * time.Millisecond)

		// Stop app
		stopErr := app.Stop(ctx)
		assert.NoError(t, stopErr)

		// Wait for run to complete
		select {
		case runErr := <-runDone:
			assert.NoError(t, runErr)
		case <-ctx.Done():
			t.Fatal("test timed out")
		}
	})
}

// TestApp_RunShutdownRacingServiceError proves Run does not panic when a
// service error arrives while Run is already unwinding on context cancellation.
//
// Run's deferred close(errCh) used to be registered LAST, so LIFO ran it FIRST
// — closing the channel while the goroutine that sends on it was still live.
// A shutdown that raced a non-nil error from ServiceManager.Run therefore hit
// "panic: send on closed channel", which no recover in the tree catches: it
// takes the whole process down during what should be a graceful stop.
//
// The service below returns an error only AFTER its context is cancelled, so
// cancelling reliably produces an in-flight send during teardown.
func TestApp_RunShutdownRacingServiceError(t *testing.T) {
	const shutdownRaceAttempts = 25

	for attempt := range shutdownRaceAttempts {
		servicemanager.ResetInstance()
		servicemanager.GetInstance().ClearServices()

		// One service fails on its own, so ServiceManager.Run is heading for a
		// NON-nil return (a cancelled context alone is treated as a clean
		// shutdown and returns nil, which never puts a send in flight).
		failer := servicemanager.NewMockService(
			fmt.Sprintf("failer-%d", attempt),
		).WithRunError(errShutdownRace)

		// The second holds ServiceManager.Run inside its own teardown, which is
		// what widens the window: Run's deferred Stop waits on this Stop, so
		// App.Run has time to observe ctx.Done, return, and fire its defers
		// before the goroutine finally sends the error.
		slow := &slowStopService{
			name: fmt.Sprintf("slow-stop-%d", attempt),
		}

		app := &App{serviceManager: servicemanager.GetInstance()}
		app.serviceManager.Add(failer, slow)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)

		go func() {
			done <- app.Run(ctx)
		}()

		// Cancel once the slow service is inside Run, so App.Run's select takes
		// the ctx.Done branch while the error is still working its way out.
		require.Eventually(t, slow.started.Load,
			servicesRunningTimeout, servicesRunningPoll,
			"service never started")

		cancel()

		select {
		case <-done:
			// Run returned rather than panicking. Either error value is fine —
			// this test is about surviving the race, not about which wins.
		case <-time.After(servicesRunningTimeout):
			t.Fatalf("attempt %d: Run did not return after cancel", attempt)
		}
	}
}

// slowStopService holds ServiceManager.Run inside its teardown for a beat, so a
// non-nil error is still travelling out of Run while App.Run has already
// returned on ctx.Done and run its defers.
type slowStopService struct {
	name    string
	started atomic.Bool
}

func (s *slowStopService) Name() string {
	return s.name
}

func (s *slowStopService) Run(ctx context.Context) error {
	s.started.Store(true)

	<-ctx.Done()

	return nil
}

func (s *slowStopService) Stop(_ context.Context) error {
	time.Sleep(stopUnwindDelay)

	return nil
}

// stopUnwindDelay is how long the manager is held in teardown. Long enough that
// App.Run reliably reaches its defers first, short enough not to pad the suite.
const stopUnwindDelay = 100 * time.Millisecond

var errShutdownRace = errors.New("service failed during shutdown")

func TestApp_ServiceActivity(t *testing.T) {
	testCases := []struct {
		name         string
		serviceNames []string
		stopMethod   string // "app_stop" or "context_cancel"
		expectError  bool
	}{
		{
			name:         "single service run and stop correctly",
			serviceNames: []string{"TestService1"},
			stopMethod:   "app_stop",
			expectError:  false,
		},
		{
			name:         "multiple services run and stop correctly",
			serviceNames: []string{"TestService1", "TestService2"},
			stopMethod:   "app_stop",
			expectError:  false,
		},
		{
			name:         "services stop via context cancellation",
			serviceNames: []string{"TestService1", "TestService2"},
			stopMethod:   "context_cancel",
			expectError:  false,
		},
		{
			name: "many services run and stop correctly",
			serviceNames: []string{
				"Service1", "Service2", "Service3", "Service4",
			},
			stopMethod:  "app_stop",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset and clear services before test
			servicemanager.ResetInstance()
			servicemanager.GetInstance().ClearServices()

			// Create mock services that track their activity
			mockServices := make(
				[]*servicemanager.MockService, 0, len(tc.serviceNames),
			)
			for _, name := range tc.serviceNames {
				mockServices = append(
					mockServices, servicemanager.NewMockService(name),
				)
			}

			// Create app manually and add mock services
			app := &App{
				serviceManager: servicemanager.GetInstance(),
			}
			// Convert to Service interface for Add method
			servicesForAdd := make(
				[]servicemanager.Service, 0, len(mockServices),
			)
			for _, svc := range mockServices {
				servicesForAdd = append(servicesForAdd, svc)
			}

			app.serviceManager.Add(servicesForAdd...)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start app in goroutine
			done := make(chan error, 1)

			go func() {
				done <- app.Run(ctx)
			}()

			waitForRunningServices(t, mockServices)

			// Stop using specified method
			var stopErr error

			switch tc.stopMethod {
			case "app_stop":
				stopErr = app.Stop(ctx)
			case "context_cancel":
				cancel()
			}

			if !tc.expectError {
				assert.NoError(t, stopErr)
			}

			// Wait for run to complete
			select {
			case err := <-done:
				if !tc.expectError {
					assert.NoError(t, err)
				}
			case <-time.After(time.Second):
				t.Fatal("app.Run() did not complete within timeout")
			}

			// Verify all services have stopped
			for i, mockSvc := range mockServices {
				assert.False(t, mockSvc.IsRunning(),
					"Service %d (%s) should be stopped", i, mockSvc.Name())
			}
		})
	}
}

// runAppBriefly starts the app, waits briefly, cancels, and
// waits for Run to return.
func runAppBriefly(t *testing.T, a *App) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() { done <- a.Run(ctx) }()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestApp_PreRunHooks(t *testing.T) {
	t.Run("hooks run in order before services", func(t *testing.T) {
		resetInstance()

		a := createTestApp()

		var order []string

		a.OnPreRun(func(_ context.Context) {
			order = append(order, "hook1")
		})
		a.OnPreRun(func(_ context.Context) {
			order = append(order, "hook2")
		})

		runAppBriefly(t, a)

		assert.Equal(t, []string{"hook1", "hook2"}, order)
	})

	t.Run("no hooks is fine", func(t *testing.T) {
		resetInstance()

		a := createTestApp()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := a.Run(ctx)
		assert.NoError(t, err)
	})
}

func TestApp_PostStopHooks(t *testing.T) {
	t.Run("single hook runs after stop", func(t *testing.T) {
		resetInstance()

		a := createTestApp()

		var called bool

		a.OnPostStop(func(_ context.Context) {
			called = true
		})

		runAppBriefly(t, a)

		assert.True(t, called)
	})

	t.Run("multiple hooks run in order after stop", func(t *testing.T) {
		resetInstance()

		a := createTestApp()

		var order []string

		a.OnPostStop(func(_ context.Context) {
			order = append(order, "cleanup1")
		})
		a.OnPostStop(func(_ context.Context) {
			order = append(order, "cleanup2")
		})

		runAppBriefly(t, a)

		assert.Equal(t, []string{"cleanup1", "cleanup2"}, order)
	})
}
