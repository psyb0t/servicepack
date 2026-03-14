package servicemanager

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/spf13/cobra"
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

// ReadyNotifier is optionally implemented by services that
// need to signal when they're actually ready to serve.
// The service manager waits for the Ready channel to close
// before starting dependent services. Services that don't
// implement this are considered ready immediately after
// their goroutine is launched.
type ReadyNotifier interface {
	Ready() <-chan struct{}
}

// Commander is optionally implemented by services that expose
// CLI subcommands. The returned commands are added under the
// service name: ./app <servicename> <subcommand>.
type Commander interface {
	Commands() []*cobra.Command
}

// serviceGroup is a set of services that can start concurrently.
// Groups are ordered: group 0 starts first, then group 1, etc.
type serviceGroup []Service

const defaultStopTimeout = 30 * time.Second

type ServiceManager struct {
	services      map[string]Service
	servicesMutex sync.RWMutex
	startGroups   []serviceGroup
	startGroupsMu sync.RWMutex
	wg            sync.WaitGroup
	cancel        context.CancelFunc
	cancelMu      sync.Mutex
	stopOnce      sync.Once
	stopTimeout   time.Duration
}

func GetInstance() *ServiceManager {
	serviceManagerOnce.Do(func() {
		serviceManagerInstance = &ServiceManager{
			services:    make(map[string]Service),
			stopTimeout: defaultStopTimeout,
		}
	})

	return serviceManagerInstance
}

func ResetInstance() {
	serviceManagerOnce = sync.Once{}
	serviceManagerInstance = nil
}

func (s *ServiceManager) Commands() []*cobra.Command {
	s.servicesMutex.RLock()
	defer s.servicesMutex.RUnlock()

	var cmds []*cobra.Command

	for _, svc := range s.services {
		cmdr, ok := svc.(Commander)
		if !ok {
			continue
		}

		parent := &cobra.Command{
			Use:   svc.Name(),
			Short: svc.Name() + " commands",
		}

		parent.AddCommand(cmdr.Commands()...)
		cmds = append(cmds, parent)
	}

	return cmds
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
	s.startGroupsMu.Lock()
	defer s.startGroupsMu.Unlock()

	for i, group := range groups {
		names := make([]string, 0, len(group))
		for _, svc := range group {
			names = append(names, svc.Name())
		}

		slog.Debug("starting service group",
			"group", i,
			"services", names,
		)

		launchedCh := make(chan struct{}, len(group))

		for _, service := range group {
			s.wg.Add(1)

			go func(svc Service) {
				defer s.wg.Done()

				launchedCh <- struct{}{}

				s.runService(ctx, svc, errCh)
			}(service)
		}

		for range len(group) {
			<-launchedCh
		}

		s.waitGroupReady(ctx, group)
		s.startGroups = append(s.startGroups, group)
	}
}

func (s *ServiceManager) waitGroupReady(
	ctx context.Context,
	group serviceGroup,
) {
	for _, svc := range group {
		rn, ok := svc.(ReadyNotifier)
		if !ok {
			continue
		}

		slog.Debug("waiting for service ready",
			"service", svc.Name(),
		)

		select {
		case <-rn.Ready():
			slog.Debug("service ready",
				"service", svc.Name(),
			)
		case <-ctx.Done():
			return
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

		lastErr = s.safeRun(ctx, service)
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

func (s *ServiceManager) safeRun(
	ctx context.Context,
	service Service,
) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		slog.Error("service panicked",
			"service", service.Name(),
			"panic", r,
		)

		err = ctxerrors.Wrapf(
			ErrServicePanic, "%v", r,
		)
	}()

	return service.Run(ctx) //nolint:wrapcheck
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

		s.startGroupsMu.RLock()
		defer s.startGroupsMu.RUnlock()

		for i := len(s.startGroups) - 1; i >= 0; i-- {
			s.stopGroup(ctx, s.startGroups[i])
		}
	})
}

func (s *ServiceManager) stopGroup(
	ctx context.Context,
	group serviceGroup,
) {
	var wg sync.WaitGroup

	for _, service := range group {
		wg.Add(1)

		go func(svc Service) {
			defer wg.Done()

			slog.Debug("stopping service",
				"service", svc.Name(),
			)

			s.stopServiceWithTimeout(ctx, svc)
		}(service)
	}

	wg.Wait()
}

func (s *ServiceManager) stopServiceWithTimeout(
	ctx context.Context,
	service Service,
) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		ctx, cancel := context.WithTimeout(
			ctx, s.stopTimeout,
		)
		defer cancel()

		if err := service.Stop(ctx); err != nil {
			slog.Error(
				"failed to stop service",
				"service", service.Name(),
				"error", err,
			)
		}
	}()

	timer := time.NewTimer(s.stopTimeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		slog.Error("service stop timed out",
			"service", service.Name(),
			"timeout", s.stopTimeout,
		)
	}
}

func resolveOrder(
	services map[string]Service,
) ([]serviceGroup, error) {
	inDegree, dependents := buildDepGraph(services)

	return topoSort(services, inDegree, dependents)
}

func buildDepGraph(
	services map[string]Service,
) (map[string]int, map[string][]string) {
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
				slog.Warn(
					"dependency not in process, skipping",
					"service", name,
					"dependency", depName,
				)

				continue
			}

			inDegree[name]++

			dependents[depName] = append(
				dependents[depName], name,
			)
		}
	}

	return inDegree, dependents
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
