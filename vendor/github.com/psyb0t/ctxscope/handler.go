package ctxscope

import (
	"context"
	"log/slog"
	"maps"
	"slices"

	"github.com/psyb0t/ctxerrors"
)

// Handler stamps the scope carried by a context onto every record passing
// through it, then delegates to an inner handler.
//
// Installing one at startup is what makes plain slog.InfoContext(ctx, ...)
// carry request_id — including from code that has never heard of this package,
// which GetLogger by definition cannot reach:
//
//	slog.SetDefault(slog.New(ctxscope.NewHandler(base)))
//
// Handler and GetLogger are ALTERNATIVES, not layers. GetLogger applies the
// scope itself, so calling it under an installed Handler emits every attribute
// twice. Install the Handler and log with the Context-suffixed slog calls, or
// install nothing and go through GetLogger — one or the other, per project.
//
// Only the Context-suffixed calls (InfoContext, ErrorContext, ...) carry a
// context to the handler. Plain slog.Info hands it a background context, so the
// line still gets the global tier — which never came from a context — but none
// of the per-context tier. That is slog's contract, not this package's choice.
type Handler struct {
	root slog.Handler

	// ops replays the WithAttrs / WithGroup calls made on this handler AFTER
	// the scope goes on, so scope attributes always land at the record's top
	// level. Adding them to the record instead would nest them inside any open
	// group, and a request_id buried under a group is not the request_id that
	// log queries match on.
	ops []func(slog.Handler) slog.Handler
}

// NewHandler wraps inner so that records handled through it carry the scope of
// the context they were logged with.
func NewHandler(inner slog.Handler) *Handler {
	return &Handler{root: inner, ops: nil}
}

// Enabled reports whether inner handles this level. Scope never changes the
// answer — it adds attributes, it does not gate them.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.root.Enabled(ctx, level)
}

// Handle merges both tiers onto the record's handler, the context tier winning
// collisions, and delegates. The merge happens per record because the scope is
// read from the context at THIS moment — a handler built earlier still sees
// attributes set later.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	merged := GetGlobal()
	maps.Copy(merged, Get(ctx))

	next := h.root
	if len(merged) > 0 {
		next = next.WithAttrs(slogAttrs(merged))
	}

	for _, op := range h.ops {
		next = op(next)
	}

	if err := next.Handle(ctx, record); err != nil {
		return ctxerrors.Wrap(err, "handle record")
	}

	return nil
}

// WithAttrs records the call rather than applying it, so Handle can replay it
// after the scope has gone on. See the ops field.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return h.withOp(func(inner slog.Handler) slog.Handler {
		return inner.WithAttrs(attrs)
	})
}

// WithGroup records the call rather than applying it. This is the reason ops
// exists at all: a group opened here must not swallow the scope attributes.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return h.withOp(func(inner slog.Handler) slog.Handler {
		return inner.WithGroup(name)
	})
}

// withOp returns a copy carrying one more replayed operation. Clip forces the
// append to allocate, so two handlers derived from the same parent cannot write
// over each other's ops through a shared backing array.
func (h *Handler) withOp(op func(slog.Handler) slog.Handler) *Handler {
	return &Handler{
		root: h.root,
		ops:  append(slices.Clip(h.ops), op),
	}
}

// slogAttrs returns s as slog attributes, sorted so field order is stable
// across lines and diffable.
func slogAttrs(s Scope) []slog.Attr {
	out := make([]slog.Attr, 0, len(s))
	for _, key := range slices.Sorted(maps.Keys(s)) {
		out = append(out, slog.Any(key, s[key]))
	}

	return out
}
