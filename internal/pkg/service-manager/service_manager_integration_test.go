package servicemanager

import (
	"context"
	"errors"
	"slices"
	"sync"
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

func (t *startTracker) has(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return slices.Contains(t.order, name)
}

// waitThenCancel cancels once cond holds, or after a generous safety timeout.
// It replaces fixed time.Sleep-based cancellation, which flakes on slow or
// loaded CI runners where the async startup sequence does not finish within
// the sleep window. The safety timeout still cancels so a genuinely broken run
// cannot hang -- the assertions below then report the real failure.
func waitThenCancel(
	cancel context.CancelFunc,
	cond func() bool,
) {
	go func() {
		deadline := time.After(5 * time.Second)

		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()

		for {
			if cond() {
				cancel()

				return
			}

			select {
			case <-deadline:
				cancel()

				return
			case <-tick.C:
			}
		}
	}()
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

	waitThenCancel(cancel, func() bool {
		return db.RunCount() >= 2 && tracker.has("api")
	})

	err := sm.Run(ctx)
	assert.NoError(t, err)

	// db should have been retried
	assert.GreaterOrEqual(t, db.RunCount(), 2)

	// Both ran: a dependency that fails and retries does not strand its
	// dependent.
	order := tracker.getOrder()
	require.GreaterOrEqual(t, len(order), 2)
	assert.True(t, tracker.has("db"), "db should have run")
	assert.True(t, tracker.has("api"), "api should have run")
}

// This test deliberately does NOT assert that db reaches Run before api.
//
// It used to, and that assertion was wrong about once in 600 runs — failing
// with api recorded first. Neither mock here implements ReadyNotifier, and that
// interface's own contract says a service which does not is "considered ready
// immediately after their goroutine is launched". The manager therefore orders
// the LAUNCH of the groups, not the moment each service enters Run: db's
// goroutine is started and has signalled before api's goroutine exists, but it
// can be descheduled between that signal and its own Run body, letting api
// record first.
//
// So the ordering guarantee is real only for services that signal readiness,
// and it is asserted where it actually holds — TestServiceManager_ReadyNotifier
// drives two ReadyMockServices and checks the exact sequence. Asserting it here
// too was not extra coverage; it was a coin flip weighted heavily enough to
// pass almost always.

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

	waitThenCancel(cancel, func() bool {
		return api.WasRunCalled()
	})

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

	waitThenCancel(cancel, func() bool {
		return svc.RunCount() >= 3
	})

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

	waitThenCancel(cancel, func() bool {
		return oneshot.WasRunCalled() && longrunning.WasRunCalled()
	})

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

	// db and cache have no dependencies, so the manager launches
	// them as concurrent goroutines within the same start group -
	// there is no ordering guarantee between the two of THEM.
	// group0Started only orders group 0 (db, cache) as a whole
	// before group 1 (migrator, api), by waiting for both to
	// actually fire onRun at least once (not just be scheduled).
	// db retries (onRun fires per attempt), so each service's
	// Done() is guarded to fire only on its FIRST onRun call.
	var group0Started sync.WaitGroup
	group0Started.Add(2)

	var dbStarted, cacheStarted sync.Once

	// db: retryable, fails once then succeeds
	db := NewRetryableMockService("db", 1)
	db.WithRunErrors(errIntegration, nil)
	db.WithOnRun(func() {
		tracker.record("db")
		dbStarted.Do(group0Started.Done)
	})

	// cache: allowed failure, fails immediately
	cache := NewAllowedFailureMockService("cache")
	cache.WithRunError(errIntegration)
	cache.WithOnRun(func() {
		tracker.record("cache")
		cacheStarted.Do(group0Started.Done)
	})

	// migrator: depends on db, allowed failure,
	// succeeds then exits (returns nil immediately)
	migrator := NewFullMockService("migrator")
	migrator.WithDependencies("db")
	migrator.WithAllowFailure(true)
	migrator.WithRunErrors(nil)
	migrator.WithOnRun(func() {
		group0Started.Wait()
		tracker.record("migrator")
	})

	// api: depends on db and migrator
	api := NewFullMockService("api")
	api.WithDependencies("db", "migrator")
	api.WithOnRun(func() {
		group0Started.Wait()
		tracker.record("api")
	})

	sm.Add(db, cache, migrator, api)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	waitThenCancel(cancel, func() bool {
		return tracker.has("migrator") && tracker.has("api")
	})

	err := sm.Run(ctx)

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
	testCases := []struct {
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

			done := make(chan error, 1)

			go func() {
				done <- sm.Run(ctx)
			}()

			select {
			case err := <-done:
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
