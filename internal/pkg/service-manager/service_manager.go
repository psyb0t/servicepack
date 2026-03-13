package servicemanager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/psyb0t/ctxerrors"
)

var (
	//nolint:gochecknoglobals
	serviceManagerInstance *ServiceManager
	//nolint:gochecknoglobals
	serviceManagerOnce sync.Once
)

type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Retryable is optionally implemented by services that want
// automatic restart on failure. MaxRetries returns the maximum
// number of retry attempts (0 means no retries).
// RetryDelay returns the delay between retries. Use
// human-readable durations like 1s, 2m, 1h30m via
// time.Duration.
type Retryable interface {
	MaxRetries() int
	RetryDelay() time.Duration
}

// AllowedFailure is optionally implemented by services whose
// failure should not bring down the entire service manager.
type AllowedFailure interface {
	IsAllowedFailure() bool
}

// Dependent is optionally implemented by services that must
// start after other services. Dependencies returns the names
// of services this service depends on.
type Dependent interface {
	Dependencies() []string
}

// serviceGroup is a set of services that can start concurrently.
// Groups are ordered: group 0 starts first, then group 1, etc.
type serviceGroup []Service

type ServiceManager struct {
	services             map[string]Service
	servicesMutex        sync.RWMutex
	runningServices      []Service
	runningServicesMutex sync.RWMutex
	wg                   sync.WaitGroup
	cancel               context.CancelFunc
	cancelMu             sync.Mutex
	stopOnce             sync.Once
}

func GetInstance() *ServiceManager {
	serviceManagerOnce.Do(func() {
		serviceManagerInstance = &ServiceManager{
			services: make(map[string]Service),
		}
	})

	return serviceManagerInstance
}

func ResetInstance() {
	serviceManagerOnce = sync.Once{}
	serviceManagerInstance = nil
}

func (s *ServiceManager) ClearServices() {
	s.servicesMutex.Lock()
	defer s.servicesMutex.Unlock()

	s.services = make(map[string]Service)
}

func (s *ServiceManager) Add(services ...Service) {
	s.servicesMutex.Lock()
	defer s.servicesMutex.Unlock()

	for _, service := range services {
		slog.Debug("registering service",
			"service", service.Name(),
		)

		s.services[service.Name()] = service
	}
}

func (s *ServiceManager) Run(ctx context.Context) error {
	slog.Info("running services")

	ctx, cancel := context.WithCancel(ctx)

	s.cancelMu.Lock()
	s.cancel = cancel
	s.cancelMu.Unlock()

	s.servicesMutex.RLock()
	defer s.servicesMutex.RUnlock()

	errCh := make(chan error, 1)
	defer close(errCh)

	defer s.wg.Wait()
	defer s.Stop(ctx)

	if len(s.services) == 0 {
		return ErrNoEnabledServices
	}

	groups, err := resolveOrder(s.services)
	if err != nil {
		return ctxerrors.Wrap(
			err, "failed to resolve service order",
		)
	}

	slog.Debug("resolved service order",
		"groups", len(groups),
		"services", len(s.services),
	)

	s.runServiceGroups(ctx, groups, errCh)

	select {
	case <-ctx.Done():
		slog.Info("services run context done")

		return nil
	case err := <-errCh:
		return ctxerrors.Wrap(err, "service failed")
	}
}

func (s *ServiceManager) runServiceGroups(
	ctx context.Context,
	groups []serviceGroup,
	errCh chan<- error,
) {
	s.runningServicesMutex.Lock()
	defer s.runningServicesMutex.Unlock()

	for i, group := range groups {
		names := make([]string, 0, len(group))
		for _, svc := range group {
			names = append(names, svc.Name())
		}

		slog.Debug("starting service group",
			"group", i,
			"services", names,
		)

		readyCh := make(chan struct{}, len(group))

		for _, service := range group {
			s.wg.Add(1)

			go func(svc Service) {
				defer s.wg.Done()

				readyCh <- struct{}{}

				s.runService(ctx, svc, errCh)
			}(service)

			s.runningServices = append(
				s.runningServices, service,
			)
		}

		for range len(group) {
			<-readyCh
		}
	}
}

func (s *ServiceManager) runService(
	ctx context.Context,
	service Service,
	errCh chan<- error,
) {
	maxRetries := 0

	retryable, ok := service.(Retryable)
	if ok {
		maxRetries = retryable.MaxRetries()
	}

	var lastErr error

	for attempt := range maxRetries + 1 {
		slog.Debug("running service",
			"service", service.Name(),
			"attempt", attempt+1,
		)

		lastErr = service.Run(ctx)
		if lastErr == nil {
			slog.Info("service exited cleanly",
				"service", service.Name(),
			)

			return
		}

		if ctx.Err() != nil {
			slog.Debug(
				"context cancelled during retry",
				"service", service.Name(),
				"attempt", attempt+1,
			)

			return
		}

		if attempt >= maxRetries {
			break
		}

		if !s.waitRetryDelay(
			ctx, service, retryable,
			attempt, maxRetries, lastErr,
		) {
			return
		}
	}

	slog.Error("service failed",
		"service", service.Name(),
		"attempts", maxRetries+1,
		"error", lastErr,
	)

	s.handleServiceError(service, lastErr, errCh)
}

// waitRetryDelay logs the retry and waits for the delay.
// Returns false if context was cancelled during the wait.
func (s *ServiceManager) waitRetryDelay(
	ctx context.Context,
	service Service,
	retryable Retryable,
	attempt int,
	maxRetries int,
	err error,
) bool {
	delay := retryable.RetryDelay()

	slog.Warn("service failed, retrying",
		"service", service.Name(),
		"attempt", attempt+1,
		"maxRetries", maxRetries,
		"retryDelay", delay,
		"error", err,
	)

	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)

	select {
	case <-ctx.Done():
		timer.Stop()

		return false
	case <-timer.C:
		return true
	}
}

func (s *ServiceManager) handleServiceError(
	service Service,
	err error,
	errCh chan<- error,
) {
	af, ok := service.(AllowedFailure)
	if ok && af.IsAllowedFailure() {
		slog.Warn("service failed (allowed failure)",
			"service", service.Name(),
			"error", err,
		)

		return
	}

	errCh <- err
}

func (s *ServiceManager) Stop(ctx context.Context) {
	s.cancelMu.Lock()

	if s.cancel != nil {
		s.cancel()
	}

	s.cancelMu.Unlock()

	s.stopOnce.Do(func() {
		slog.Info("stopping services")
		defer slog.Info("stopped services")

		s.runningServicesMutex.RLock()
		defer s.runningServicesMutex.RUnlock()

		var wg sync.WaitGroup
		for _, service := range s.runningServices {
			wg.Add(1)

			go func(service Service) {
				defer wg.Done()

				slog.Debug("stopping service",
					"service", service.Name(),
				)

				if err := service.Stop(ctx); err != nil {
					slog.Error(
						"failed to stop service",
						"service", service.Name(),
						"error", err,
					)
				}
			}(service)
		}

		wg.Wait()
	})
}

func resolveOrder(
	services map[string]Service,
) ([]serviceGroup, error) {
	inDegree, dependents, err := buildDepGraph(services)
	if err != nil {
		return nil, err
	}

	return topoSort(services, inDegree, dependents)
}

func buildDepGraph(
	services map[string]Service,
) (map[string]int, map[string][]string, error) {
	inDegree := make(map[string]int, len(services))
	dependents := make(
		map[string][]string, len(services),
	)

	for name := range services {
		inDegree[name] = 0
	}

	for name, svc := range services {
		dep, ok := svc.(Dependent)
		if !ok {
			continue
		}

		for _, depName := range dep.Dependencies() {
			if _, exists := services[depName]; !exists {
				return nil, nil, fmt.Errorf(
					"%w: service %q depends on %q",
					ErrDependencyNotFound,
					name,
					depName,
				)
			}

			inDegree[name]++

			dependents[depName] = append(
				dependents[depName], name,
			)
		}
	}

	return inDegree, dependents, nil
}

func topoSort(
	services map[string]Service,
	inDegree map[string]int,
	dependents map[string][]string,
) ([]serviceGroup, error) {
	var groups []serviceGroup

	processed := 0

	queue := make([]string, 0, len(services))

	for name, deg := range inDegree {
		if deg != 0 {
			continue
		}

		queue = append(queue, name)
	}

	for len(queue) > 0 {
		group := make(serviceGroup, 0, len(queue))
		for _, name := range queue {
			group = append(group, services[name])
		}

		groups = append(groups, group)
		processed += len(queue)

		nextQueue := make([]string, 0)

		for _, name := range queue {
			for _, dep := range dependents[name] {
				inDegree[dep]--

				if inDegree[dep] != 0 {
					continue
				}

				nextQueue = append(nextQueue, dep)
			}
		}

		queue = nextQueue
	}

	if processed != len(services) {
		return nil, ErrCyclicDependency
	}

	return groups, nil
}
