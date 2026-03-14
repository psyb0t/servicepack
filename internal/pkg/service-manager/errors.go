package servicemanager

import "errors"

var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrNoEnabledServices  = errors.New("no enabled services")
	ErrCyclicDependency   = errors.New("cyclic dependency detected")
	ErrDependencyNotFound = errors.New("dependency not found")
	ErrMaxRetriesReached  = errors.New("max retries reached")
	ErrStopTimeout        = errors.New("service stop timed out")
	ErrServicePanic       = errors.New("service panicked")
)
