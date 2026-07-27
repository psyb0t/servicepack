package ctxerrors

import (
	"errors"
	"sync"
)

//nolint:gochecknoglobals
var (
	errorMapMu sync.RWMutex
	errorMap   = map[error]error{}
)

// SetErrorMap replaces the global translation map. Intended to be called
// once at startup. Entries with a nil key or nil value are skipped.
func SetErrorMap(m map[error]error) {
	errorMapMu.Lock()
	defer errorMapMu.Unlock()

	errorMap = make(map[error]error, len(m))

	for from, to := range m {
		if from == nil || to == nil {
			continue
		}

		errorMap[from] = to
	}
}

// MapError registers a single translation from -> to. Useful for incremental
// registration across packages in init(). No-op if either err is nil.
func MapError(from, to error) {
	if from == nil || to == nil {
		return
	}

	errorMapMu.Lock()
	defer errorMapMu.Unlock()

	errorMap[from] = to
}

// ClearErrorMap removes all mappings.
func ClearErrorMap() {
	errorMapMu.Lock()
	defer errorMapMu.Unlock()

	errorMap = map[error]error{}
}

// translate returns the mapped error if err matches a registered key via
// errors.Is, else returns err unchanged. Single-pass — no recursive mapping.
func translate(err error) error {
	if err == nil {
		return nil
	}

	errorMapMu.RLock()
	defer errorMapMu.RUnlock()

	for from, to := range errorMap {
		if errors.Is(err, from) {
			return to
		}
	}

	return err
}
