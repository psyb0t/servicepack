// Package handlers holds the structural slog handlers this module composes
// chains from: Handler, which writes the process's own output, and
// FanOutHandler, which tees one record to many.
//
// Destinations live in their own packages beside this one — handlers/logring
// for a searchable in-memory ring, handlers/loki for a Loki server. Those carry
// real API surface of their own; these two carry a type and a constructor each,
// which is why they sit here rather than in a subpackage apiece.
package handlers

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/psyb0t/ctxerrors"
)

// DefaultSplitLevel is where records start going to stderr instead of stdout.
//
// This is a deliberate departure from stdlib slog, which sends EVERY level to
// stderr. Splitting lets a container log collector keep error noise separate
// from happy-path noise without parsing anything.
const DefaultSplitLevel = slog.LevelWarn

// Options configures a Handler at construction.
//
// Level is the only setting that stays live afterwards: it is a slog.Leveler,
// so passing a *slog.LevelVar and calling Set on it later changes what the
// handler emits from that moment on. Everything else is fixed once built.
type Options struct {
	// Format is the wire shape. Empty means DefaultFormat.
	Format Format

	// Level is the minimum level emitted. Nil means slog.LevelInfo.
	//
	// Hand it a *slog.LevelVar to be able to change the level at runtime; the
	// handler resolves it on every record rather than reading it once.
	Level slog.Leveler

	// AddSource attaches file/line/function to every record.
	AddSource bool

	// SplitAt is the level at and above which records go to stderr rather than
	// stdout. Nil means DefaultSplitLevel. Point both writer sets at the same
	// place and this stops mattering.
	SplitAt slog.Leveler
}

// Option supplies writers to New. Stdout and Stderr are the two that exist;
// each is variadic, which is how a single call can name several writers per
// stream — Go allows only the FINAL parameter of a function to be variadic, so
// one constructor taking two variadic lists is not expressible.
type Option func(*writerSet)

type writerSet struct {
	stdout []io.Writer
	stderr []io.Writer
}

// Stdout appends writers to the stdout side.
func Stdout(w ...io.Writer) Option {
	return func(s *writerSet) {
		s.stdout = append(s.stdout, w...)
	}
}

// Stderr appends writers to the stderr side.
func Stderr(w ...io.Writer) Option {
	return func(s *writerSet) {
		s.stderr = append(s.stderr, w...)
	}
}

// Handler is the process's output: every record goes to one of two writer sets,
// chosen by level.
//
// Point both sets at the same writer and everything lands together, which is
// what stdlib slog does. Point them at different writers — the default — and
// warnings and errors are separable from the rest without parsing.
type Handler struct {
	stdout  slog.Handler
	stderr  slog.Handler
	level   slog.Leveler
	splitAt slog.Leveler
}

// NewStd builds the conventional handler: stdout for informational records,
// stderr for warnings and above.
func NewStd(opts Options) (*Handler, error) {
	return New(opts, Stdout(os.Stdout), Stderr(os.Stderr))
}

// New builds a Handler over the writers the options name, defaulting either
// side to the matching os stream when nothing named it.
//
// Several writers on one side are joined with io.MultiWriter, so they all
// receive the same bytes. Different RENDERINGS per destination is a different
// job — build a Handler each and tee them with a FanOutHandler.
func New(opts Options, writers ...Option) (*Handler, error) {
	set := &writerSet{}
	for _, apply := range writers {
		apply(set)
	}

	if len(set.stdout) == 0 {
		set.stdout = []io.Writer{os.Stdout}
	}

	if len(set.stderr) == 0 {
		set.stderr = []io.Writer{os.Stderr}
	}

	format := opts.Format
	if format == "" {
		format = DefaultFormat
	}

	var level slog.Leveler = slog.LevelInfo
	if opts.Level != nil {
		level = opts.Level
	}

	var splitAt slog.Leveler = DefaultSplitLevel
	if opts.SplitAt != nil {
		splitAt = opts.SplitAt
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: opts.AddSource,
		Level:     level,
	}

	stdout, err := format.newHandler(join(set.stdout), handlerOpts)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build stdout handler")
	}

	stderr, err := format.newHandler(join(set.stderr), handlerOpts)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build stderr handler")
	}

	return &Handler{
		stdout:  stdout,
		stderr:  stderr,
		level:   level,
		splitAt: splitAt,
	}, nil
}

// join collapses a writer set, skipping io.MultiWriter for the one-writer case
// that almost every process actually has.
func join(writers []io.Writer) io.Writer {
	if len(writers) == 1 {
		return writers[0]
	}

	return io.MultiWriter(writers...)
}

// Enabled reports whether this handler emits records at the given level.
//
// The Leveler is resolved HERE rather than captured at construction, so a
// *slog.LevelVar the caller bumps later actually takes effect. Storing the
// resolved slog.Level instead would silently ignore every later change.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle writes the record to whichever side its level selects.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= h.splitAt.Level() {
		if err := h.stderr.Handle(ctx, r); err != nil {
			return ctxerrors.Wrap(err, "stderr handler failed")
		}

		return nil
	}

	if err := h.stdout.Handle(ctx, r); err != nil {
		return ctxerrors.Wrap(err, "stdout handler failed")
	}

	return nil
}

// WithAttrs returns a Handler with the attrs bound to both sides.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		stdout:  h.stdout.WithAttrs(attrs),
		stderr:  h.stderr.WithAttrs(attrs),
		level:   h.level,
		splitAt: h.splitAt,
	}
}

// WithGroup returns a Handler with the group applied to both sides.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		stdout:  h.stdout.WithGroup(name),
		stderr:  h.stderr.WithGroup(name),
		level:   h.level,
		splitAt: h.splitAt,
	}
}
