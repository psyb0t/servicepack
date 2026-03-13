package gonfiguration

import "errors"

var (
	ErrNilDestination       = errors.New("destination is nil")
	ErrInvalidEnvVar        = errors.New("invalid environment variable")
	ErrTargetNotPointer     = errors.New("destination must be a pointer")
	ErrDestinationNotStruct = errors.New("destination must be a struct")
	ErrUnsupportedFieldType = errors.New("unsupported field type")
	ErrRequiredFieldNotSet  = errors.New("required field not set")
	ErrDefaultTypeMismatch  = errors.New("default value type mismatch")
)
