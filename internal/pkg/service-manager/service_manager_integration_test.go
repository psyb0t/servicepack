package servicemanager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errIntegration = errors.New("integration test error")

type startTracker struct {
	mu    sync.Mutex
	order []string
}

func (t *startTracker) record(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.order = append(t.order, name)
}

func (t *startTracker) getOrder() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	cp := make([]string, len(t.order))
	copy(cp, t.order)

	return cp
}

func (t *startTracker) makeCallback(
	name string,
) func() {
	return func() {
		t.record(name)
	}
}

func TestIntegration_RetryWithDependencies(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()
	tracker := &startTracker{}

	// db: retryable, fails once then succeeds
	db := NewRetryableMockService("db", 2)
	db.WithRunErrors(errIntegration, nil)
	db.WithOnRun(tracker.makeCallback("db"))

	// api: depends on db
	api := NewDependentMockService("api", "db")
	api.WithOnRun(tracker.makeCallback("api"))

	sm.Add(db, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	assert.NoError(t, err)

	// db should have been retried
	assert.GreaterOrEqual(t, db.RunCount(), 2)

	// db should start before api
	order := tracker.getOrder()
	require.GreaterOrEqual(t, len(order), 2)

	dbIdx := -1
	apiIdx := -1

	for i, name := range order {
		if name == "db" && dbIdx == -1 {
			dbIdx = i
		}

		if name == "api" && apiIdx == -1 {
			apiIdx = i
		}
	}

	assert.Less(t, dbIdx, apiIdx,
		"db should start before api")
}

func TestIntegration_AllowedFailureWithDependencies(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()

	// db: healthy, no deps
	db := NewMockService("db")

	// migrator: depends on db, allowed failure, fails
	migrator := NewFullMockService("migrator")
	migrator.WithDependencies("db")
	migrator.WithAllowFailure(true)
	migrator.WithRunError(errIntegration)

	// api: depends on db, healthy
	api := NewDependentMockService("api", "db")

	sm.Add(db, migrator, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	// Manager should NOT die from migrator failure
	assert.NoError(t, err)

	// api should have been started
	assert.True(t, api.WasRunCalled())
}

func TestIntegration_RetryAndAllowedFailure(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()

	// svc: retryable + allowed failure, always fails
	svc := NewFullMockService("flaky")
	svc.WithMaxRetries(2)
	svc.WithAllowFailure(true)
	svc.WithRunError(errIntegration)

	// healthy service to keep manager alive
	healthy := NewMockService("healthy")

	sm.Add(svc, healthy)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	assert.NoError(t, err)

	// Should have retried 3 times (1 + 2 retries)
	assert.Equal(t, 3, svc.RunCount())
}

func TestIntegration_OneShotServiceExitsCleanly(
	t *testing.T,
) {
	ResetInstance()

	sm := GetInstance()

	// oneshot: returns nil immediately (like a migrator)
	oneshot := NewMockService("oneshot").
		WithRunErrors(nil)

	// longrunning: blocks on context
	longrunning := NewMockService("longrunning")

	sm.Add(oneshot, longrunning)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sm.Run(ctx)
	// Manager should NOT die when oneshot exits
	assert.NoError(t, err)

	// Both services should have run
	assert.True(t, oneshot.WasRunCalled())
	assert.True(t, longrunning.WasRunCalled())
}

func TestIntegration_FullStack(t *testing.T) {
	ResetInstance()

	sm := GetInstance()
	tracker := &startTracker{}

	// db: retryable, fails once then succeeds
	db := NewRetryableMockService("db", 1)
	db.WithRunErrors(errIntegration, nil)
	db.WithOnRun(tracker.makeCallback("db"))

	// cache: allowed failure, fails immediately
	cache := NewAllowedFailureMockService("cache")
	cache.WithRunError(errIntegration)
	cache.WithOnRun(tracker.makeCallback("cache"))

	// migrator: depends on db, allowed failure,
	// succeeds then exits (returns nil immediately)
	migrator := NewFullMockService("migrator")
	migrator.WithDependencies("db")
	migrator.WithAllowFailure(true)
	migrator.WithRunErrors(nil)
	migrator.WithOnRun(
		tracker.makeCallback("migrator"),
	)

	// api: depends on db and migrator
	api := NewFullMockService("api")
	api.WithDependencies("db", "migrator")
	api.WithOnRun(tracker.makeCallback("api"))

	sm.Add(db, cache, migrator, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	var managerDone int32

	go func() {
		time.Sleep(200 * time.Millisecond)

		if atomic.LoadInt32(&managerDone) == 0 {
			cancel()
		}
	}()

	err := sm.Run(ctx)

	atomic.StoreInt32(&managerDone, 1)

	// Manager should stay up despite cache failure
	assert.NoError(t, err)

	// Verify start ordering
	order := tracker.getOrder()

	// db and cache should appear before migrator/api
	dbFirst := false
	cacheFirst := false
	migratorIdx := -1
	apiIdx := -1

	for i, name := range order {
		switch name {
		case "db":
			if migratorIdx == -1 && apiIdx == -1 {
				dbFirst = true
			}
		case "cache":
			if migratorIdx == -1 && apiIdx == -1 {
				cacheFirst = true
			}
		case "migrator":
			if migratorIdx == -1 {
				migratorIdx = i
			}
		case "api":
			if apiIdx == -1 {
				apiIdx = i
			}
		}
	}

	assert.True(t, dbFirst,
		"db should start before migrator/api")
	assert.True(t, cacheFirst,
		"cache should start before migrator/api")
}

func TestIntegration_BackwardCompatibility(
	t *testing.T,
) {
	tests := []struct {
		name        string
		services    []Service
		expectError bool
		cancelAfter time.Duration
	}{
		{
			name: "plain services all start",
			services: []Service{
				NewTestService("a"),
				NewTestService("b"),
				NewTestService("c"),
			},
			expectError: false,
			cancelAfter: 20 * time.Millisecond,
		},
		{
			name: "one failure kills all",
			services: []Service{
				NewMockService("ok"),
				NewMockService("bad").
					WithRunError(errIntegration),
			},
			expectError: true,
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

			done := make(chan error, 1)

			go func() {
				done <- sm.Run(ctx)
			}()

			select {
			case err := <-done:
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
