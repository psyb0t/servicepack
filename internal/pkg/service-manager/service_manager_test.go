package servicemanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTestService     = errors.New("test error")
	errTestServiceStop = errors.New("stop error")
)

func TestGetInstance(t *testing.T) {
	// Reset singleton before test
	ResetInstance()

	sm1 := GetInstance()
	assert.NotNil(t, sm1)
	assert.NotNil(t, sm1.services)
	assert.NotNil(t, sm1.services)
	assert.Equal(t, 0, len(sm1.services))

	// Test that subsequent calls return the same instance
	sm2 := GetInstance()
	assert.Same(t, sm1, sm2)

	// Test that NewServiceManager also returns the same instance
	sm3 := GetInstance()
	assert.Same(t, sm1, sm3)
}

func TestServiceManager_Add(t *testing.T) {
	testCases := []struct {
		name     string
		services []Service
		expected int
	}{
		{
			name:     "add single service",
			services: []Service{NewMockService("test1")},
			expected: 1,
		},
		{
			name:     "add multiple services",
			services: []Service{NewMockService("test1"), NewMockService("test2")},
			expected: 2,
		},
		{
			name:     "add services with duplicate names overwrites",
			services: []Service{NewMockService("test1"), NewMockService("test1")},
			expected: 1,
		},
		{
			name:     "add no services",
			services: []Service{},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			assert.Equal(t, tc.expected, len(sm.services))

			// Verify all services are accessible by name
			for _, svc := range tc.services {
				storedSvc, exists := sm.services[svc.Name()]
				if tc.expected > 0 {
					assert.True(t, exists)
					assert.Equal(t, svc.Name(), storedSvc.Name())
				}
			}
		})
	}
}

func TestServiceManager_Run(t *testing.T) {
	testCases := []struct {
		name         string
		services     []Service
		contextSetup func() (context.Context, context.CancelFunc)
		expectError  bool
		stopMethod   string // "context", "stop_method", "service_error"
	}{
		{
			name:     "run single service with context cancellation",
			services: []Service{NewMockService("test1")},
			contextSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()

				return ctx, cancel
			},
			expectError: false,
			stopMethod:  "context",
		},
		{
			name:     "run multiple services with context cancellation",
			services: []Service{NewMockService("test1"), NewMockService("test2")},
			contextSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()

				return ctx, cancel
			},
			expectError: false,
			stopMethod:  "context",
		},
		{
			name:     "run service with error",
			services: []Service{NewMockService("failing").WithRunError(errTestService)},
			contextSetup: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectError: true,
			stopMethod:  "service_error",
		},
		{
			name:     "run with stop method",
			services: []Service{NewMockService("test1")},
			contextSetup: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectError: false,
			stopMethod:  "stop_method",
		},
		{
			name:     "run with no services",
			services: []Service{},
			contextSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()

				return ctx, cancel
			},
			expectError: true, // No services means "no enabled services"
			stopMethod:  "context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			ctx, cancel := tc.contextSetup()
			defer cancel()

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			// Give services time to start
			time.Sleep(5 * time.Millisecond)

			// Verify services started (for non-error cases)
			if !tc.expectError && len(tc.services) > 0 {
				for _, svc := range tc.services {
					if mockSvc, ok := svc.(*MockService); ok {
						assert.True(t, mockSvc.WasRunCalled(),
							"Service %s should have Run called", mockSvc.name)
					}
				}
			}

			// Execute stop method
			switch tc.stopMethod {
			case "stop_method":
				sm.Stop(ctx)
			case "context":
				// Already set up in contextSetup
			case "service_error":
				// Service will error naturally
			}

			// Wait for Run to complete
			select {
			case err := <-runDone:
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(time.Second):
				t.Fatal("ServiceManager.Run() did not complete within timeout")
			}
		})
	}
}

func TestServiceManager_Stop(t *testing.T) {
	testCases := []struct {
		name             string
		services         []Service
		stopTwice        bool
		expectStopErrors bool
	}{
		{
			name:      "stop single service",
			services:  []Service{NewMockService("test1")},
			stopTwice: false,
		},
		{
			name:      "stop multiple services",
			services:  []Service{NewMockService("test1"), NewMockService("test2")},
			stopTwice: false,
		},
		{
			name:      "stop same manager twice (sync.Once behavior)",
			services:  []Service{NewMockService("test1")},
			stopTwice: true,
		},
		{
			name: "stop service with error",
			services: []Service{
				NewMockService("failing").WithStopError(errTestServiceStop),
			},
			expectStopErrors: true,
		},
		{
			name:     "stop with no services",
			services: []Service{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			ctx := context.Background()

			// First run the services if there are any
			// (so they get added to runningServices)
			if len(tc.services) > 0 {
				runCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
				defer cancel()

				// Run services in background so they can be stopped
				runDone := make(chan error, 1)

				go func() {
					runDone <- sm.Run(runCtx)
				}()

				// Give services time to start
				time.Sleep(5 * time.Millisecond)

				// Now stop them
				sm.Stop(ctx)

				// Wait for run to complete
				<-runDone
			} else {
				// For empty services test, just call stop
				sm.Stop(ctx)
			}

			// Verify all services had Stop called (only for non-empty services)
			if len(tc.services) > 0 {
				for _, svc := range tc.services {
					if mockSvc, ok := svc.(*MockService); ok {
						assert.True(t, mockSvc.WasStopCalled(),
							"Service %s should have Stop called", mockSvc.name)
					}
				}
			}

			// Second stop (if testing sync.Once)
			if tc.stopTwice {
				// Reset the stop called flag for testing
				for _, svc := range tc.services {
					if mockSvc, ok := svc.(*MockService); ok {
						atomic.StoreInt32(&mockSvc.stopCalled, 0)
					}
				}

				// Call stop again - this should be a no-op due to sync.Once
				sm.Stop(ctx)

				// Services should NOT be stopped again due to sync.Once
				for _, svc := range tc.services {
					if mockSvc, ok := svc.(*MockService); ok {
						assert.False(t, mockSvc.WasStopCalled(),
							"Service %s should NOT have Stop called again", mockSvc.name)
					}
				}
			}
		})
	}
}

func TestServiceManager_Concurrency(t *testing.T) {
	t.Run("concurrent add and run", func(t *testing.T) {
		ResetInstance()

		sm := GetInstance()

		// Add services concurrently
		done := make(chan bool, 10)

		for i := range 10 {
			go func(id int) {
				svc := NewMockService(fmt.Sprintf("service%d", id))
				sm.Add(svc)

				done <- true
			}(i)
		}

		// Wait for all adds to complete
		for range 10 {
			<-done
		}

		// Verify all services were added
		assert.Equal(t, 10, len(sm.services))

		// Run the manager
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := sm.Run(ctx)
		assert.NoError(t, err)
	})
}

func TestResolveOrder(t *testing.T) {
	testCases := []struct {
		name        string
		services    map[string]Service
		expectError error
		groupCount  int
	}{
		{
			name: "no dependencies - single group",
			services: map[string]Service{
				"a": NewTestService("a"),
				"b": NewTestService("b"),
			},
			groupCount: 1,
		},
		{
			name: "linear chain a->b->c",
			services: map[string]Service{
				"c": NewTestService("c"),
				"b": NewDependentMockService("b", "c"),
				"a": NewDependentMockService("a", "b"),
			},
			groupCount: 3,
		},
		{
			name: "diamond a->b,c b->d c->d",
			services: map[string]Service{
				"d": NewTestService("d"),
				"b": NewDependentMockService("b", "d"),
				"c": NewDependentMockService("c", "d"),
				"a": NewFullMockService("a").
					WithDependencies("b", "c"),
			},
			groupCount: 3,
		},
		{
			name: "cycle a->b b->a",
			services: map[string]Service{
				"a": NewDependentMockService("a", "b"),
				"b": NewDependentMockService("b", "a"),
			},
			expectError: ErrCyclicDependency,
		},
		{
			name: "self dependency",
			services: map[string]Service{
				"a": NewDependentMockService("a", "a"),
			},
			expectError: ErrCyclicDependency,
		},
		{
			name: "missing dep skipped (external)",
			services: map[string]Service{
				"a": NewDependentMockService(
					"a", "nonexistent",
				),
			},
			groupCount: 1,
		},
		{
			name: "mixed deps and no deps",
			services: map[string]Service{
				"db":  NewTestService("db"),
				"api": NewDependentMockService("api", "db"),
				"log": NewTestService("log"),
			},
			groupCount: 2,
		},
		{
			name: "single service no deps",
			services: map[string]Service{
				"solo": NewTestService("solo"),
			},
			groupCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groups, err := resolveOrder(tc.services)

			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)

				return
			}

			assert.NoError(t, err)
			assert.Len(t, groups, tc.groupCount)

			// Verify all services are present
			total := 0
			for _, g := range groups {
				total += len(g)
			}

			assert.Equal(t, len(tc.services), total)
		})
	}
}

func TestServiceManager_RunWithRetries(t *testing.T) {
	testCases := []struct {
		name         string
		setupService func() Service
		expectError  bool
		expectedRuns int
		cancelAfter  time.Duration
	}{
		{
			name: "service succeeds first try",
			setupService: func() Service {
				svc := NewRetryableMockService("ok", 2)
				// No errors - will block on ctx
				return svc
			},
			expectError:  false,
			expectedRuns: 1,
			cancelAfter:  20 * time.Millisecond,
		},
		{
			name: "service fails all retries",
			setupService: func() Service {
				svc := NewRetryableMockService("fail", 2)
				svc.WithRunError(errTestService)

				return svc
			},
			expectError:  true,
			expectedRuns: 3, // 1 + 2 retries
		},
		{
			name: "service fails then succeeds",
			setupService: func() Service {
				svc := NewRetryableMockService("retry", 2)
				svc.WithRunErrors(
					errTestService,
					errTestService,
					nil, // third attempt succeeds
				)

				return svc
			},
			expectError:  false,
			expectedRuns: 3,
			cancelAfter:  50 * time.Millisecond,
		},
		{
			name: "context cancelled during retry",
			setupService: func() Service {
				svc := NewRetryableMockService(
					"ctxcancel", 10,
				)
				svc.WithRunError(errTestService)
				svc.WithRunDelay(
					50 * time.Millisecond,
				)

				return svc
			},
			expectError:  false,
			expectedRuns: -1,
			cancelAfter:  30 * time.Millisecond,
		},
		{
			name: "retry with delay",
			setupService: func() Service {
				svc := NewRetryableMockService(
					"delayed", 1,
				)
				svc.WithRetryDelay(
					10 * time.Millisecond,
				)
				svc.WithRunErrors(
					errTestService, nil,
				)

				return svc
			},
			expectError:  false,
			expectedRuns: 2,
			cancelAfter:  200 * time.Millisecond,
		},
		{
			name: "ctx cancelled during retry delay",
			setupService: func() Service {
				svc := NewRetryableMockService(
					"delaycancel", 10,
				)
				svc.WithRetryDelay(time.Second)
				svc.WithRunError(errTestService)

				return svc
			},
			expectError:  false,
			expectedRuns: -1,
			cancelAfter:  50 * time.Millisecond,
		},
		{
			name: "non-retryable service fails",
			setupService: func() Service {
				return NewMockService("norety").
					WithRunError(errTestService)
			},
			expectError:  true,
			expectedRuns: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			svc := tc.setupService()
			sm.Add(svc)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tc.cancelAfter > 0 {
				go func() {
					time.Sleep(tc.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timed out")
			}

			if tc.expectedRuns < 0 {
				return
			}

			switch s := svc.(type) {
			case *RetryableMockService:
				assert.Equal(
					t, tc.expectedRuns, s.RunCount(),
				)
			case *MockService:
				assert.Equal(
					t, tc.expectedRuns, s.RunCount(),
				)
			}
		})
	}
}

func TestServiceManager_RunWithAllowedFailure(t *testing.T) {
	testCases := []struct {
		name        string
		services    []Service
		expectError bool
		cancelAfter time.Duration
	}{
		{
			name: "allowed failure does not kill manager",
			services: []Service{
				func() Service {
					s := NewAllowedFailureMockService(
						"fail-ok",
					)
					s.WithRunError(errTestService)

					return s
				}(),
				NewMockService("healthy"),
			},
			expectError: false,
			cancelAfter: 50 * time.Millisecond,
		},
		{
			name: "non-allowed failure kills manager",
			services: []Service{
				NewMockService("fail-bad").
					WithRunError(errTestService),
				NewMockService("healthy2"),
			},
			expectError: true,
		},
		{
			name: "allowed failure with retries exhausted",
			services: []Service{
				func() Service {
					s := NewFullMockService("retry-fail")
					s.WithMaxRetries(1)
					s.WithAllowFailure(true)
					s.WithRunError(
						errTestService,
					)

					return s
				}(),
				NewMockService("healthy3"),
			},
			expectError: false,
			cancelAfter: 50 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tc.cancelAfter > 0 {
				go func() {
					time.Sleep(tc.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tc.expectError {
					assert.Error(t, err)

					return
				}

				assert.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out")
			}
		})
	}
}

func TestServiceManager_RunWithDependencies(t *testing.T) {
	testCases := []struct {
		name        string
		services    []Service
		expectError bool
		errorIs     error
		cancelAfter time.Duration
	}{
		{
			name: "dependent starts after dependency",
			services: []Service{
				NewTestService("db"),
				NewDependentMockService("api", "db"),
			},
			expectError: false,
			cancelAfter: 50 * time.Millisecond,
		},
		{
			name: "cyclic dependency error",
			services: []Service{
				NewDependentMockService("a", "b"),
				NewDependentMockService("b", "a"),
			},
			expectError: true,
			errorIs:     ErrCyclicDependency,
		},
		{
			name: "missing dep runs anyway (external)",
			services: []Service{
				NewDependentMockService(
					"api", "nonexistent",
				),
			},
			expectError: false,
			cancelAfter: 20 * time.Millisecond,
		},
		{
			name: "no deps backward compat",
			services: []Service{
				NewTestService("a"),
				NewTestService("b"),
				NewTestService("c"),
			},
			expectError: false,
			cancelAfter: 20 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tc.cancelAfter > 0 {
				go func() {
					time.Sleep(tc.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tc.expectError {
					assert.Error(t, err)

					if tc.errorIs != nil {
						assert.ErrorIs(
							t, err, tc.errorIs,
						)
					}

					return
				}

				assert.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out")
			}
		})
	}
}

type panicService struct {
	name  string
	value any
}

func (p *panicService) Name() string { return p.name }

func (p *panicService) Run(
	_ context.Context,
) error {
	panic(p.value)
}

func (p *panicService) Stop(
	_ context.Context,
) error {
	return nil
}

type stopTrackingService struct {
	Service
	onStop func()
}

func (s *stopTrackingService) Stop(
	ctx context.Context,
) error {
	s.onStop()

	return s.Service.Stop(ctx)
}

type dependentStopTrackingService struct {
	stopTrackingService
	deps []string
}

func (d *dependentStopTrackingService) Dependencies() []string {
	return d.deps
}

func TestServiceManager_PanicRecovery(t *testing.T) {
	testCases := []struct {
		name       string
		panicValue any
	}{
		{"string panic", "oh no"},
		{"error panic", errTestService},
		{"int panic", 42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(&panicService{
				name:  "panicker",
				value: tc.panicValue,
			})

			ctx := t.Context()

			done := make(chan error, 1)

			go func() {
				done <- sm.Run(ctx)
			}()

			select {
			case err := <-done:
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrServicePanic)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out")
			}
		})
	}
}

func TestServiceManager_ReverseOrderShutdown(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()

	var (
		stopOrder []string
		mu        sync.Mutex
	)

	recordStop := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		stopOrder = append(stopOrder, name)
	}

	db := &stopTrackingService{
		Service: NewTestService("db"),
		onStop:  func() { recordStop("db") },
	}

	api := &dependentStopTrackingService{
		stopTrackingService: stopTrackingService{
			Service: NewTestService("api"),
			onStop:  func() { recordStop("api") },
		},
		deps: []string{"db"},
	}

	sm.Add(db, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, stopOrder, 2)
	assert.Equal(t, "api", stopOrder[0],
		"api should stop before db")
	assert.Equal(t, "db", stopOrder[1],
		"db should stop after api")
}

func TestServiceManager_ReadyNotifier(t *testing.T) {
	ResetInstance()

	sm := GetInstance()

	var (
		startOrder []string
		mu         sync.Mutex
	)

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		startOrder = append(startOrder, name)
	}

	// db: signals ready after 50ms
	db := NewReadyMockService("db")
	db.WithOnRun(func() {
		record("db")

		// Simulate startup delay then signal ready
		go func() {
			time.Sleep(50 * time.Millisecond)
			db.SignalReady()
		}()
	})

	// api: depends on db, should not start
	// until db signals ready
	api := NewReadyMockService("api", "db")
	api.WithOnRun(func() {
		record("api")
		api.SignalReady()
	})

	sm.Add(db, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, startOrder, 2)
	assert.Equal(t, "db", startOrder[0])
	assert.Equal(t, "api", startOrder[1])
}

func TestServiceManager_ReadyNotifierNotImplemented(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()

	// Plain services without ReadyNotifier
	// should start immediately
	a := NewTestService("a")
	b := NewTestService("b")

	sm.Add(a, b)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	assert.NoError(t, err)
}

type commanderService struct {
	*MockService
}

func (c *commanderService) Commands() []*cobra.Command {
	return []*cobra.Command{
		{
			Use:   "do-stuff",
			Short: "does stuff",
		},
	}
}

func TestServiceManager_Commands(t *testing.T) {
	testCases := []struct {
		name     string
		services []Service
		expected int
	}{
		{
			name: "service with commands",
			services: []Service{
				&commanderService{
					MockService: NewMockService("svc"),
				},
			},
			expected: 1,
		},
		{
			name: "service without commands",
			services: []Service{
				NewTestService("plain"),
			},
			expected: 0,
		},
		{
			name: "mixed",
			services: []Service{
				&commanderService{
					MockService: NewMockService("cmd1"),
				},
				NewTestService("plain"),
				&commanderService{
					MockService: NewMockService("cmd2"),
				},
			},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tc.services...)

			cmds := sm.Commands()
			assert.Len(t, cmds, tc.expected)

			for _, cmd := range cmds {
				assert.NotEmpty(t, cmd.Use)
				assert.True(t, cmd.HasSubCommands())
			}
		})
	}
}
