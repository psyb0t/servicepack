package services

import (
	"context"
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
	services      map[string]Service
	servicesMutex sync.RWMutex
	wg            sync.WaitGroup
	doneCh        chan struct{}
	stopOnce      sync.Once
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

func (s *ServiceManager) Run(ctx context.Context) error {
	logrus.Info("running registered services")

	s.servicesMutex.RLock()
	defer s.servicesMutex.RUnlock()

	errCh := make(chan error, 1)
	defer close(errCh)

	defer s.wg.Wait()
	defer s.Stop(ctx)

	for _, service := range s.services {
		s.wg.Add(1)

		go func(service Service) {
			defer s.wg.Done()

			if err := service.Run(ctx); err != nil {
				errCh <- err
			}
		}(service)
	}

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

func (s *ServiceManager) Stop(ctx context.Context) {
	s.stopOnce.Do(func() {
		logrus.Info("stopping services")
		defer logrus.Info("stopped services")

		close(s.doneCh)

		s.servicesMutex.RLock()
		defer s.servicesMutex.RUnlock()

		var wg sync.WaitGroup
		for _, service := range s.services {
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
