package servicemanager

import (
	"context"
	"sync/atomic"
	"time"
)

type TestService struct{ name string }

func NewTestService(name string) *TestService {
	return &TestService{name: name}
}

func (s *TestService) Name() string { return s.name }

func (s *TestService) Run(ctx context.Context) error {
	<-ctx.Done()

	return nil
}

func (s *TestService) Stop(_ context.Context) error {
	return nil
}

type MockService struct {
	name       string
	running    int32
	runCalled  int32
	stopCalled int32
	runCount   int32
	runError   error
	runErrors  []error
	stopError  error
	stopCh     chan struct{}
	runDelay   time.Duration
	onRun      func()
}

func NewMockService(name string) *MockService {
	return &MockService{
		name:   name,
		stopCh: make(chan struct{}),
	}
}

func (m *MockService) WithRunError(
	err error,
) *MockService {
	m.runError = err

	return m
}

func (m *MockService) WithRunErrors(
	errs ...error,
) *MockService {
	m.runErrors = errs

	return m
}

func (m *MockService) WithStopError(
	err error,
) *MockService {
	m.stopError = err

	return m
}

func (m *MockService) WithRunDelay(
	delay time.Duration,
) *MockService {
	m.runDelay = delay

	return m
}

func (m *MockService) Name() string {
	return m.name
}

func (m *MockService) WithOnRun(
	fn func(),
) *MockService {
	m.onRun = fn

	return m
}

func (m *MockService) Run(ctx context.Context) error {
	count := atomic.AddInt32(&m.runCount, 1)
	atomic.StoreInt32(&m.runCalled, 1)
	atomic.StoreInt32(&m.running, 1)

	if m.onRun != nil {
		m.onRun()
	}

	if m.runDelay > 0 {
		time.Sleep(m.runDelay)
	}

	if len(m.runErrors) > 0 {
		idx := int(count) - 1
		if idx < len(m.runErrors) &&
			m.runErrors[idx] != nil {
			return m.runErrors[idx]
		}
	}

	if m.runError != nil {
		return m.runError
	}

	select {
	case <-ctx.Done():
		return nil
	case <-m.stopCh:
		return nil
	}
}

func (m *MockService) Stop(_ context.Context) error {
	atomic.StoreInt32(&m.stopCalled, 1)
	atomic.StoreInt32(&m.running, 0)

	select {
	case <-m.stopCh:
		// already closed
	default:
		close(m.stopCh)
	}

	return m.stopError
}

func (m *MockService) WasRunCalled() bool {
	return atomic.LoadInt32(&m.runCalled) == 1
}

func (m *MockService) WasStopCalled() bool {
	return atomic.LoadInt32(&m.stopCalled) == 1
}

func (m *MockService) IsRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

func (m *MockService) RunCount() int {
	return int(atomic.LoadInt32(&m.runCount))
}

type RetryableMockService struct {
	*MockService
	maxRetries int
	retryDelay time.Duration
}

func NewRetryableMockService(
	name string,
	maxRetries int,
) *RetryableMockService {
	return &RetryableMockService{
		MockService: NewMockService(name),
		maxRetries:  maxRetries,
	}
}

func (r *RetryableMockService) WithRetryDelay(
	d time.Duration,
) *RetryableMockService {
	r.retryDelay = d

	return r
}

func (r *RetryableMockService) MaxRetries() int {
	return r.maxRetries
}

func (r *RetryableMockService) RetryDelay() time.Duration {
	return r.retryDelay
}

type AllowedFailureMockService struct {
	*MockService
}

func NewAllowedFailureMockService(
	name string,
) *AllowedFailureMockService {
	return &AllowedFailureMockService{
		MockService: NewMockService(name),
	}
}

func (a *AllowedFailureMockService) IsAllowedFailure() bool {
	return true
}

type DependentMockService struct {
	*MockService
	dependencies []string
}

func NewDependentMockService(
	name string,
	deps ...string,
) *DependentMockService {
	return &DependentMockService{
		MockService:  NewMockService(name),
		dependencies: deps,
	}
}

func (d *DependentMockService) Dependencies() []string {
	return d.dependencies
}

type FullMockService struct {
	*MockService
	maxRetries   int
	retryDelay   time.Duration
	allowFailure bool
	dependencies []string
}

func NewFullMockService(name string) *FullMockService {
	return &FullMockService{
		MockService: NewMockService(name),
	}
}

func (f *FullMockService) WithMaxRetries(
	n int,
) *FullMockService {
	f.maxRetries = n

	return f
}

func (f *FullMockService) WithAllowFailure(
	v bool,
) *FullMockService {
	f.allowFailure = v

	return f
}

func (f *FullMockService) WithDependencies(
	deps ...string,
) *FullMockService {
	f.dependencies = deps

	return f
}

func (f *FullMockService) WithRetryDelay(
	d time.Duration,
) *FullMockService {
	f.retryDelay = d

	return f
}

func (f *FullMockService) MaxRetries() int {
	return f.maxRetries
}

func (f *FullMockService) RetryDelay() time.Duration {
	return f.retryDelay
}

func (f *FullMockService) IsAllowedFailure() bool {
	return f.allowFailure
}

func (f *FullMockService) Dependencies() []string {
	return f.dependencies
}
