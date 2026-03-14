package servicemanager

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errTestAdditionalContext = errors.New("additional context")
	errTestDifferent         = errors.New("different error")
)

func TestErrorDefinitions(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrServiceNotFound",
			err:         ErrServiceNotFound,
			expectedMsg: "service not found",
		},
		{
			name:        "ErrNoEnabledServices",
			err:         ErrNoEnabledServices,
			expectedMsg: "no enabled services",
		},
		{
			name:        "ErrCyclicDependency",
			err:         ErrCyclicDependency,
			expectedMsg: "cyclic dependency detected",
		},
		{
			name:        "ErrDependencyNotFound",
			err:         ErrDependencyNotFound,
			expectedMsg: "dependency not found",
		},
		{
			name:        "ErrMaxRetriesReached",
			err:         ErrMaxRetriesReached,
			expectedMsg: "max retries reached",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, tc.err)
			assert.Equal(t, tc.expectedMsg, tc.err.Error())
			assert.True(t, errors.Is(tc.err, tc.err))
		})
	}
}

func TestErrorMatching(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "ErrServiceNotFound matches itself",
			err:      ErrServiceNotFound,
			target:   ErrServiceNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrServiceNotFound matches",
			err:      errors.Join(ErrServiceNotFound, errTestAdditionalContext),
			target:   ErrServiceNotFound,
			expected: true,
		},
		{
			name:     "different error does not match",
			err:      errTestDifferent,
			target:   ErrServiceNotFound,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := errors.Is(tc.err, tc.target)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ErrServiceNotFound is proper error type", func(t *testing.T) {
		err := ErrServiceNotFound
		assert.NotNil(t, err)
		assert.Implements(t, (*error)(nil), ErrServiceNotFound)
	})
}
