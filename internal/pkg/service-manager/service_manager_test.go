package servicemanager

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tt.services...)

			assert.Equal(t, tt.expected, len(sm.services))

			// Verify all services are accessible by name
			for _, svc := range tt.services {
				storedSvc, exists := sm.services[svc.Name()]
				if tt.expected > 0 {
					assert.True(t, exists)
					assert.Equal(t, svc.Name(), storedSvc.Name())
				}
			}
		})
	}
}

func TestServiceManager_Run(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tt.services...)

			ctx, cancel := tt.contextSetup()
			defer cancel()

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			// Give services time to start
			time.Sleep(5 * time.Millisecond)

			// Verify services started (for non-error cases)
			if !tt.expectError && len(tt.services) > 0 {
				for _, svc := range tt.services {
					if mockSvc, ok := svc.(*MockService); ok {
						assert.True(t, mockSvc.WasRunCalled(),
							"Service %s should have Run called", mockSvc.name)
					}
				}
			}

			// Execute stop method
			switch tt.stopMethod {
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
				if tt.expectError {
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
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tt.services...)

			ctx := context.Background()

			// First run the services if there are any
			// (so they get added to runningServices)
			if len(tt.services) > 0 {
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
			if len(tt.services) > 0 {
				for _, svc := range tt.services {
					if mockSvc, ok := svc.(*MockService); ok {
						assert.True(t, mockSvc.WasStopCalled(),
							"Service %s should have Stop called", mockSvc.name)
					}
				}
			}

			// Second stop (if testing sync.Once)
			if tt.stopTwice {
				// Reset the stop called flag for testing
				for _, svc := range tt.services {
					if mockSvc, ok := svc.(*MockService); ok {
						atomic.StoreInt32(&mockSvc.stopCalled, 0)
					}
				}

				// Call stop again - this should be a no-op due to sync.Once
				sm.Stop(ctx)

				// Services should NOT be stopped again due to sync.Once
				for _, svc := range tt.services {
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
	tests := []struct {
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
			name: "missing dependency",
			services: map[string]Service{
				"a": NewDependentMockService(
					"a", "nonexistent",
				),
			},
			expectError: ErrDependencyNotFound,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := resolveOrder(tt.services)

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)

				return
			}

			assert.NoError(t, err)
			assert.Len(t, groups, tt.groupCount)

			// Verify all services are present
			total := 0
			for _, g := range groups {
				total += len(g)
			}

			assert.Equal(t, len(tt.services), total)
		})
	}
}

func TestServiceManager_RunWithRetries(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			svc := tt.setupService()
			sm.Add(svc)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tt.cancelAfter > 0 {
				go func() {
					time.Sleep(tt.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timed out")
			}

			if tt.expectedRuns < 0 {
				return
			}

			switch s := svc.(type) {
			case *RetryableMockService:
				assert.Equal(
					t, tt.expectedRuns, s.RunCount(),
				)
			case *MockService:
				assert.Equal(
					t, tt.expectedRuns, s.RunCount(),
				)
			}
		})
	}
}

func TestServiceManager_RunWithAllowedFailure(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tt.services...)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tt.cancelAfter > 0 {
				go func() {
					time.Sleep(tt.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tt.expectError {
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
	tests := []struct {
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
			name: "missing dependency error",
			services: []Service{
				NewDependentMockService(
					"api", "nonexistent",
				),
			},
			expectError: true,
			errorIs:     ErrDependencyNotFound,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetInstance()

			sm := GetInstance()
			sm.Add(tt.services...)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			defer cancel()

			if tt.cancelAfter > 0 {
				go func() {
					time.Sleep(tt.cancelAfter)
					cancel()
				}()
			}

			runDone := make(chan error, 1)

			go func() {
				runDone <- sm.Run(ctx)
			}()

			select {
			case err := <-runDone:
				if tt.expectError {
					assert.Error(t, err)

					if tt.errorIs != nil {
						assert.ErrorIs(
							t, err, tt.errorIs,
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
