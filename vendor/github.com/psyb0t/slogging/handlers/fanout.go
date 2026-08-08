package handlers

import (
	"context"
	"log/slog"

	"github.com/psyb0t/ctxerrors"
)

// FanOutHandler tees one record to every handler it holds.
//
// It holds the slog.Handler INTERFACE, not this package's Handler: that is what
// lets a searchable ring, a Loki client and anything a caller wrote themselves
// sit in the same chain. Narrowing it to a concrete type would make the fan-out
// useless for the sinks it exists to carry.
type FanOutHandler struct {
	handlers []slog.Handler
}

// NewFanOut creates a handler that dispatches to all the given handlers.
func NewFanOut(handlers ...slog.Handler) *FanOutHandler {
	return &FanOutHandler{handlers: handlers}
}

// Len reports how many handlers the fan-out dispatches to. Exported so callers
// can assert that adding a sink stacked onto the existing set rather than
// replacing it — the difference is invisible from the outside until logs go
// missing.
func (h *FanOutHandler) Len() int {
	return len(h.handlers)
}

// Handlers returns a copy of the dispatch list.
//
// A copy, not the slice: handing out the backing array would let a caller
// reorder or nil an entry and silently change where the process logs. The
// copy is also what makes replace-slot-0 expressible from another package
// without exposing the field.
func (h *FanOutHandler) Handlers() []slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	copy(out, h.handlers)

	return out
}

// Enabled reports whether ANY underlying handler wants records at this level.
//
// Any, not all: a ring retaining DEBUG while the console sits at INFO is a
// normal setup, and the fan-out must not veto a record one of its children
// asked for. Each child re-checks its own level in Handle, so the noisier one
// does not force output on the quieter one.
func (h *FanOutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle dispatches the record to every underlying handler, and keeps going
// when one of them fails.
//
// Returning at the first error would let a single broken sink silence all the
// ones after it: slog discards whatever Handle returns, so a Loki handler that
// cannot reach its server would take stdout down with it and nothing would say
// so. Every handler sees the record regardless of what the earlier ones did,
// and the failures are joined so none is lost.
func (h *FanOutHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error

	for i, handler := range h.handlers {
		if !handler.Enabled(ctx, r.Level) {
			continue
		}

		if err := handler.Handle(ctx, r); err != nil {
			errs = append(errs, ctxerrors.Wrapf(err, "handler %d failed", i))
		}
	}

	return ctxerrors.Join(errs...)
}

// WithAttrs returns a FanOutHandler with the attrs added to every handler.
func (h *FanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}

	return &FanOutHandler{handlers: handlers}
}

// WithGroup returns a FanOutHandler with the group applied to every handler.
func (h *FanOutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}

	return &FanOutHandler{handlers: handlers}
}
