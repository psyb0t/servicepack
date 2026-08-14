package ctxscope

import (
	"maps"
	"sync/atomic"
)

// An atomic pointer to an immutable map rather than a guarded one: GetLogger
// reads this on every call, so reads stay lock-free and writers copy-and-swap.
var globalScope atomic.Pointer[Scope] //nolint:gochecknoglobals // the process-wide tier IS this package's point

// GetGlobal returns a copy of the process-wide scope, safe to mutate.
func GetGlobal() Scope {
	current := globalScope.Load()

	out := make(Scope, len(*orEmpty(current)))
	maps.Copy(out, *orEmpty(current))

	return out
}

// SetGlobal adds attributes to every line this process logs, under any context.
// Call it at startup, with the facts that describe the binary rather than the
// work — nothing set here is ever serialized by ToJSON.
//
//	scope.SetGlobal(
//	    scope.Attr("commit", commitSHA),
//	    scope.Attr("service", serviceName),
//	)
func SetGlobal(attrs ...Attribute) {
	if len(attrs) == 0 {
		return
	}

	for {
		old := globalScope.Load()

		next := make(Scope, len(*orEmpty(old))+len(attrs))
		maps.Copy(next, *orEmpty(old))

		for _, attr := range attrs {
			next[attr.key] = attr.value
		}

		if globalScope.CompareAndSwap(old, &next) {
			return
		}
	}
}

// RemoveGlobal drops the named keys from the process-wide scope. Unset keys are
// ignored.
func RemoveGlobal(keys ...string) {
	if len(keys) == 0 {
		return
	}

	for {
		old := globalScope.Load()

		if !containsAny(*orEmpty(old), keys) {
			return
		}

		next := make(Scope, len(*orEmpty(old)))
		maps.Copy(next, *orEmpty(old))

		for _, key := range keys {
			delete(next, key)
		}

		if globalScope.CompareAndSwap(old, &next) {
			return
		}
	}
}

// containsAny lets RemoveGlobal skip the copy-and-swap when nothing matches.
func containsAny(s Scope, keys []string) bool {
	for _, key := range keys {
		if _, ok := s[key]; ok {
			return true
		}
	}

	return false
}

// orEmpty absorbs the nil a never-written atomic returns, so callers can
// dereference without a nil check.
func orEmpty(s *Scope) *Scope {
	if s == nil {
		return &Scope{}
	}

	return s
}
