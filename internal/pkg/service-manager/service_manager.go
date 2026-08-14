package servicemanager

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gonfiguration"
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

// ServiceFactory is a function that creates a service instance.
type ServiceFactory func() (Service, error)

type servicesConfig struct {
	Enabled []string `env:"SERVICES_ENABLED"`
}

// serviceGroup is a set of services that can start concurrently.
// Groups are ordered: group 0 starts first, then group 1, etc.
type serviceGroup []Service

const (
	defaultStopTimeout        = 30 * time.Second
	envVarNameServicesEnabled = "SERVICES_ENABLED"
	scopeKeyService           = "service"
)

type ServiceManager struct {
	factories     map[string]ServiceFactory
	factoriesMu   sync.RWMutex
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
			factories:   make(map[string]ServiceFactory),
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

// Register stores a service factory for lazy instantiation.
// The factory is only called when the service is actually
// needed (during Run or when a Commander command is invoked).
func (s *ServiceManager) Register(
	name string,
	factory ServiceFactory,
) {
	s.factoriesMu.Lock()
	defer s.factoriesMu.Unlock()

	ctxscope.GetLogger(context.Background()).Debug(
		"registering service factory",
		"service", name,
	)

	s.factories[name] = factory
}

// RegisteredNames returns the names of all registered
// service factories.
func (s *ServiceManager) RegisteredNames() []string {
	s.factoriesMu.RLock()
	defer s.factoriesMu.RUnlock()

	names := make([]string, 0, len(s.factories))
	for name := range s.factories {
		names = append(names, name)
	}

	return names
}

// Instantiate creates a single service by calling its factory.
// Used for Commander commands that need only one service.
//
//nolint:ireturn
func (s *ServiceManager) Instantiate(
	name string,
) (Service, error) {
	s.factoriesMu.RLock()
	factory, ok := s.factories[name]
	s.factoriesMu.RUnlock()

	if !ok {
		return nil, ctxerrors.Wrapf(
			ErrServiceNotFound, "%s", name,
		)
	}

	return factory()
}

// instantiateAll calls all factories (filtered by
// SERVICES_ENABLED) and adds them to the services map.
func (s *ServiceManager) instantiateAll() error {
	return s.instantiateAllContext(context.Background())
}

func (s *ServiceManager) instantiateAllContext(ctx context.Context) error {
	s.factoriesMu.RLock()
	defer s.factoriesMu.RUnlock()

	enabledServices, allEnabled, err := parseEnabledServicesContext(ctx)
	if err != nil {
		return ctxerrors.Wrap(err, "parse enabled services")
	}

	for name, factory := range s.factories {
		if !allEnabled &&
			!slices.Contains(enabledServices, name) {
			ctxscope.GetLogger(
				withServiceScope(ctx, name),
			).Debug("service disabled, skipping")

			continue
		}

		svc, err := factory()
		if err != nil {
			return ctxerrors.Wrapf(
				err, "failed to create service %s", name,
			)
		}

		s.AddContext(ctx, svc)
	}

	return nil
}

func parseEnabledServices() ([]string, bool) {
	enabledServices, allEnabled, err := parseEnabledServicesContext(
		context.Background(),
	)
	if err != nil {
		ctxscope.GetLogger(context.Background()).Error(
			"failed to parse service filter; running all services",
			"err", err,
		)

		return nil, true
	}

	return enabledServices, allEnabled
}

func parseEnabledServicesContext(
	ctx context.Context,
) ([]string, bool, error) {
	cfg := servicesConfig{}
	if err := gonfiguration.Parse(&cfg); err != nil {
		return nil, false, ctxerrors.Wrap(err, "parse service config")
	}

	if len(cfg.Enabled) == 0 {
		return nil, true, nil
	}

	ctxscope.GetLogger(ctx).Debug("service filter active",
		"enabled", cfg.Enabled,
	)

	return cfg.Enabled, false, nil
}

// Commands returns lazy cobra commands for each registered
// service factory. Each parent command instantiates only its
// own service when invoked, so ./app <service> <subcommand>
// doesn't trigger initialization of all services.
func (s *ServiceManager) Commands() []*cobra.Command {
	s.factoriesMu.RLock()
	defer s.factoriesMu.RUnlock()

	cmds := make([]*cobra.Command, 0, len(s.factories))

	for name := range s.factories {
		cmds = append(cmds, s.buildServiceCommand(name))
	}

	return cmds
}

func (s *ServiceManager) buildServiceCommand(
	name string,
) *cobra.Command {
	return &cobra.Command{
		Use:                name,
		Short:              name + " service commands",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := withServiceScope(cmd.Context(), name)

			svc, err := s.Instantiate(name)
			if err != nil {
				ctxscope.GetLogger(ctx).Error("failed to instantiate service",
					"err", err,
				)

				return ctxerrors.Wrapf(
					err, "instantiate service %s", name,
				)
			}

			cmdr, ok := svc.(Commander)
			if !ok {
				ctxscope.GetLogger(ctx).Error("service has no commands")

				return ctxerrors.Wrapf(
					ErrNoCommands, "%s", name,
				)
			}

			sub := &cobra.Command{Use: name}
			sub.AddCommand(cmdr.Commands()...)
			sub.SetArgs(args)
			sub.SetContext(ctx)

			return sub.Execute()
		},
	}
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
		s.services[service.Name()] = service
	}
}

// AddContext registers services and records their registration in ctx's scope.
func (s *ServiceManager) AddContext(ctx context.Context, services ...Service) {
	for _, service := range services {
		ctxscope.GetLogger(
			withServiceScope(ctx, service.Name()),
		).Debug("registering service")
	}

	s.Add(services...)
}

func withServiceScope(ctx context.Context, service string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return ctxscope.Set(ctx, ctxscope.Attr(scopeKeyService, service))
}

func (s *ServiceManager) Run(ctx context.Context) error {
	ctxscope.GetLogger(ctx).Info("running services")

	if err := s.instantiateAllContext(ctx); err != nil {
		return ctxerrors.Wrap(
			err, "failed to instantiate services",
		)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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

	groups, err := resolveOrderContext(ctx, s.services)
	if err != nil {
		return ctxerrors.Wrap(
			err, "failed to resolve service order",
		)
	}

	ctxscope.GetLogger(ctx).Debug("resolved service order",
		"groups", len(groups),
		"services", len(s.services),
	)

	s.runServiceGroups(ctx, groups, errCh)

	select {
	case <-ctx.Done():
		ctxscope.GetLogger(ctx).Info("services run context done")

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

		ctxscope.GetLogger(ctx).Debug("starting service group",
			"group", i,
			"services", names,
		)

		launchedCh := make(chan struct{}, len(group))

		for _, service := range group {
			s.wg.Add(1)

			go func(svc Service) {
				defer s.wg.Done()

				launchedCh <- struct{}{}

				serviceCtx := withServiceScope(ctx, svc.Name())

				s.runService(serviceCtx, svc, errCh)
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

		serviceCtx := withServiceScope(ctx, svc.Name())

		ctxscope.GetLogger(serviceCtx).Debug("waiting for service ready")

		select {
		case <-rn.Ready():
			ctxscope.GetLogger(serviceCtx).Debug("service ready")
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
		ctxscope.GetLogger(ctx).Debug("running service",
			"attempt", attempt+1,
		)

		lastErr = s.safeRun(ctx, service)
		if lastErr == nil {
			ctxscope.GetLogger(ctx).Info("service exited cleanly")

			return
		}

		if ctx.Err() != nil {
			ctxscope.GetLogger(ctx).Debug(
				"context cancelled during retry",
				"attempt", attempt+1,
			)

			return
		}

		if attempt >= maxRetries {
			break
		}

		if !s.waitRetryDelay(
			ctx, retryable,
			attempt, maxRetries, lastErr,
		) {
			return
		}
	}

	ctxscope.GetLogger(ctx).Error("service failed",
		"attempts", maxRetries+1,
		"err", lastErr,
	)

	s.handleServiceError(ctx, service, lastErr, errCh)
}

// waitRetryDelay logs the retry and waits for the delay.
// Returns false if context was cancelled during the wait.
func (s *ServiceManager) waitRetryDelay(
	ctx context.Context,
	retryable Retryable,
	attempt int,
	maxRetries int,
	err error,
) bool {
	delay := retryable.RetryDelay()

	ctxscope.GetLogger(ctx).Warn("service failed, retrying",
		"attempt", attempt+1,
		"max_retries", maxRetries,
		"retry_delay", delay,
		"err", err,
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
	ctx context.Context,
	service Service,
	err error,
	errCh chan<- error,
) {
	af, ok := service.(AllowedFailure)
	if ok && af.IsAllowedFailure() {
		ctxscope.GetLogger(ctx).Warn("service failed (allowed failure)",
			"err", err,
		)

		return
	}

	// Non-blocking on purpose. errCh has capacity 1 and Run receives from it at
	// most once, so a plain send parks every concurrent failure after the first
	// forever — and Run's `defer s.wg.Wait()` then never returns, turning a
	// reported error into a hung process. A bare send is not selectable, so
	// context cancellation cannot rescue it either.
	//
	// Dropping the later errors is the correct semantic rather than a
	// compromise: the documented contract is that the FIRST non-allowed
	// failure stops everything, and what follows is a consequence of that
	// same shutdown. They are logged here so nothing vanishes silently.
	select {
	case errCh <- err:
	default:
		ctxscope.GetLogger(ctx).Error(
			"service failed after an earlier failure stopped the app",
			"err", err,
		)
	}
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

		ctxscope.GetLogger(ctx).Error("service panicked",
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
		ctxscope.GetLogger(ctx).Info("stopping services")
		defer ctxscope.GetLogger(ctx).Info("stopped services")

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

			serviceCtx := withServiceScope(ctx, svc.Name())

			ctxscope.GetLogger(serviceCtx).Debug("stopping service")

			s.stopServiceWithTimeout(serviceCtx, svc)
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
			ctxscope.GetLogger(ctx).Error(
				"failed to stop service",
				"err", err,
			)
		}
	}()

	timer := time.NewTimer(s.stopTimeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		ctxscope.GetLogger(ctx).Error("service stop timed out",
			"timeout", s.stopTimeout,
		)
	}
}

func resolveOrder(
	services map[string]Service,
) ([]serviceGroup, error) {
	return resolveOrderContext(context.Background(), services)
}

func resolveOrderContext(
	ctx context.Context,
	services map[string]Service,
) ([]serviceGroup, error) {
	inDegree, dependents := buildDepGraphContext(ctx, services)

	return topoSort(services, inDegree, dependents)
}

func buildDepGraphContext(
	ctx context.Context,
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
				ctxscope.GetLogger(
					withServiceScope(ctx, name),
				).Warn(
					"dependency not in process, skipping",
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
