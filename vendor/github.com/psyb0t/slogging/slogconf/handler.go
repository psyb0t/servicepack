package slogconf

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/psyb0t/ctxerrors"
)

// MultiWriterHandler routes log records to different writers based on level.
// Error and Warn levels go to stderr, Info and Debug go to stdout.
type MultiWriterHandler struct {
	stdoutHandler slog.Handler
	stderrHandler slog.Handler
	level         slog.Level
}

// NewMultiWriterHandler creates a handler that routes logs to stdout/stderr based on level.
func NewMultiWriterHandler(f format, opts *slog.HandlerOptions, stdout, stderr io.Writer) (*MultiWriterHandler, error) {
	if stdout == nil {
		stdout = os.Stdout
	}

	if stderr == nil {
		stderr = os.Stderr
	}

	stdoutHandler, err := getSlogHandler(f, stdout, opts)
	if err != nil {
		return nil, err
	}

	stderrHandler, err := getSlogHandler(f, stderr, opts)
	if err != nil {
		return nil, err
	}

	lvl := slog.LevelInfo
	if opts != nil && opts.Level != nil {
		lvl = opts.Level.Level()
	}

	return &MultiWriterHandler{
		stdoutHandler: stdoutHandler,
		stderrHandler: stderrHandler,
		level:         lvl,
	}, nil
}

// Enabled reports whether the handler handles records at the given level.
func (h *MultiWriterHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle handles the Record by routing it to the appropriate writer.
func (h *MultiWriterHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		if err := h.stderrHandler.Handle(ctx, r); err != nil {
			return ctxerrors.Wrap(err, "stderr handler failed")
		}

		return nil
	}

	if err := h.stdoutHandler.Handle(ctx, r); err != nil {
		return ctxerrors.Wrap(err, "stdout handler failed")
	}

	return nil
}

// WithAttrs returns a new Handler with the given attributes added.
func (h *MultiWriterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &MultiWriterHandler{
		stdoutHandler: h.stdoutHandler.WithAttrs(attrs),
		stderrHandler: h.stderrHandler.WithAttrs(attrs),
		level:         h.level,
	}
}

// WithGroup returns a new Handler with the given group name.
func (h *MultiWriterHandler) WithGroup(name string) slog.Handler {
	return &MultiWriterHandler{
		stdoutHandler: h.stdoutHandler.WithGroup(name),
		stderrHandler: h.stderrHandler.WithGroup(name),
		level:         h.level,
	}
}

// FanOutHandler dispatches log records to multiple handlers.
// The default slog handler is always a FanOutHandler. Use SetHandlers to replace
// all handlers and AddHandler to stack extra ones on top.
type FanOutHandler struct {
	handlers []slog.Handler
}

// NewFanOutHandler creates a handler that dispatches to all provided handlers.
func NewFanOutHandler(handlers ...slog.Handler) *FanOutHandler {
	return &FanOutHandler{handlers: handlers}
}

// Len reports how many handlers the fan-out dispatches to. Exported so callers
// can assert that AddHandler stacked onto the existing set rather than
// replacing it — the difference is invisible from the outside until logs go
// missing.
func (h *FanOutHandler) Len() int {
	return len(h.handlers)
}

// Enabled reports whether any of the underlying handlers handle records at the given level.
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

// WithAttrs returns a new FanOutHandler with the given attributes added to all handlers.
func (h *FanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}

	return &FanOutHandler{handlers: handlers}
}

// WithGroup returns a new FanOutHandler with the given group name applied to all handlers.
func (h *FanOutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}

	return &FanOutHandler{handlers: handlers}
}

// SetHandlers replaces all handlers in the default FanOutHandler with the provided ones.
func SetHandlers(handlers ...slog.Handler) {
	slog.SetDefault(slog.New(NewFanOutHandler(handlers...)))
}

// AddHandler stacks an extra handler onto the default FanOutHandler, keeping
// the existing ones. Use it to tee logs somewhere additional — a ring buffer, a
// shipper, a test capture — without replacing what init configured.
//
// Call it EARLY. slog.Logger.With and WithGroup snapshot the handler chain at
// the moment they are called, so a logger derived BEFORE the Add never carries
// the new handler and silently drops those records. Handlers added before any
// derivation reach everything.
//
// Reports whether the default logger was a FanOutHandler, i.e. whether this
// package configured it. False means something else replaced slog's default and
// the handler was stacked onto that instead — still added, but the caller has
// lost the stdout/stderr split this package sets up, which is worth noticing
// rather than discovering through absent logs.
func AddHandler(handler slog.Handler) bool {
	current := slog.Default().Handler()

	fanOut, ok := current.(*FanOutHandler)
	if !ok {
		slog.SetDefault(slog.New(NewFanOutHandler(current, handler)))

		return false
	}

	handlers := make([]slog.Handler, len(fanOut.handlers)+1)
	copy(handlers, fanOut.handlers)
	handlers[len(fanOut.handlers)] = handler

	slog.SetDefault(slog.New(NewFanOutHandler(handlers...)))

	return true
}
