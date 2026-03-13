package slogconfigurator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
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
			return fmt.Errorf("stderr handler failed: %w", err)
		}

		return nil
	}

	if err := h.stdoutHandler.Handle(ctx, r); err != nil {
		return fmt.Errorf("stdout handler failed: %w", err)
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

// Enabled reports whether any of the underlying handlers handle records at the given level.
func (h *FanOutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle dispatches the record to all underlying handlers.
func (h *FanOutHandler) Handle(ctx context.Context, r slog.Record) error {
	for i, handler := range h.handlers {
		if !handler.Enabled(ctx, r.Level) {
			continue
		}

		if err := handler.Handle(ctx, r); err != nil {
			return fmt.Errorf("handler %d failed: %w", i, err)
		}
	}

	return nil
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

// AddHandler adds an extra handler to the default FanOutHandler.
func AddHandler(handler slog.Handler) {
	current := slog.Default().Handler()

	fanOut, ok := current.(*FanOutHandler)
	if ok {
		handlers := make([]slog.Handler, len(fanOut.handlers)+1)
		copy(handlers, fanOut.handlers)
		handlers[len(fanOut.handlers)] = handler

		slog.SetDefault(slog.New(NewFanOutHandler(handlers...)))

		return
	}

	slog.SetDefault(slog.New(NewFanOutHandler(current, handler)))
}
