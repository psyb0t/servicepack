package services

import (
	"context"
	"slices"
	"sync"

	"github.com/psyb0t/ctxerrors"
	"github.com/sirupsen/logrus"
)

var (
	serviceManagerInstance *ServiceManager //nolint:gochecknoglobals
	serviceManagerOnce     sync.Once       //nolint:gochecknoglobals
)

type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

type ServiceManager struct {
	services             map[string]Service
	servicesMutex        sync.RWMutex
	runningServices      []Service
	runningServicesMutex sync.RWMutex
	wg                   sync.WaitGroup
	doneCh               chan struct{}
	stopOnce             sync.Once
}

func GetServiceManagerInstance() *ServiceManager {
	serviceManagerOnce.Do(func() {
		serviceManagerInstance = newServiceManager()
	})

	return serviceManagerInstance
}

func NewServiceManager() *ServiceManager {
	return GetServiceManagerInstance()
}

func newServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]Service),
		doneCh:   make(chan struct{}),
	}
}

// ResetServiceManagerInstance resets the singleton instance for testing purposes.
func ResetServiceManagerInstance() {
	serviceManagerOnce = sync.Once{}
	serviceManagerInstance = nil
}

// ClearServices clears all services from the service manager for testing purposes.
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

func (s *ServiceManager) Run(
	ctx context.Context,
	services []string,
) error {
	logrus.Info("running services")

	s.servicesMutex.RLock()
	defer s.servicesMutex.RUnlock()

	errCh := make(chan error, 1)
	defer close(errCh)

	defer s.wg.Wait()
	defer s.Stop(ctx)

	enabledServices, err := s.filterServices(services)
	if err != nil {
		return err
	}

	if len(enabledServices) == 0 {
		return ErrNoEnabledServices
	}

	s.runServices(ctx, enabledServices, errCh)

	select {
	case <-ctx.Done():
		logrus.Info("services run context done")

		return nil
	case err := <-errCh:
		return ctxerrors.Wrap(err, "service failed")
	case <-s.doneCh:
		return nil
	}
}

func (s *ServiceManager) filterServices(services []string) ([]Service, error) {
	enabledServices := []Service{}

	if len(services) == 0 {
		for _, service := range s.services {
			enabledServices = append(enabledServices, service)
		}

		return enabledServices, nil
	}

	for _, serviceName := range services {
		if service, ok := s.services[serviceName]; ok {
			enabledServices = append(enabledServices, service)

			continue
		}

		return nil, ctxerrors.Wrap(ErrServiceNotFound, serviceName)
	}

	// Log which services are not enabled
	for serviceName := range s.services {
		isEnabled := slices.Contains(services, serviceName)

		if !isEnabled {
			logrus.Infof("service %s is not enabled", serviceName)
		}
	}

	return enabledServices, nil
}

func (s *ServiceManager) runServices(
	ctx context.Context,
	services []Service,
	errCh chan<- error,
) {
	s.runningServicesMutex.Lock()
	defer s.runningServicesMutex.Unlock()

	for _, service := range services {
		s.wg.Add(1)

		go func(service Service) {
			defer s.wg.Done()

			if err := service.Run(ctx); err != nil {
				errCh <- err
			}
		}(service)

		s.runningServices = append(s.runningServices, service)
	}
}

func (s *ServiceManager) Stop(ctx context.Context) {
	s.stopOnce.Do(func() {
		logrus.Info("stopping services")
		defer logrus.Info("stopped services")

		close(s.doneCh)

		s.runningServicesMutex.RLock()
		defer s.runningServicesMutex.RUnlock()

		var wg sync.WaitGroup
		for _, service := range s.runningServices {
			wg.Add(1)

			go func(service Service) {
				defer wg.Done()

				if err := service.Stop(ctx); err != nil {
					logrus.Errorf("failed to stop service %s: %v",
						service.Name(), err)
				}
			}(service)
		}

		wg.Wait()
	})
}
