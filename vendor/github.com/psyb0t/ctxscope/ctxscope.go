// Package ctxscope carries attributes on a context.Context and stamps them onto
// every line logged under that context, so the values travel without anyone
// passing them around.
//
// Two tiers, split by whether an attribute describes the PROCESS or the WORK:
//
//	SetGlobal   commit, service, region   never travels
//	Set         request_id, user_id       travels, via ToJSON/FromJSON
//
// Putting a process fact in Set's tier is a bug rather than a style slip: it
// would cross a hop and overwrite the receiving service's own value, whose logs
// would then name the wrong deploy.
//
// There are two ways to get the attributes onto a line. Install NewHandler once
// at startup and plain slog.InfoContext(ctx, ...) carries them everywhere,
// including inside libraries that never heard of this package; or call
// GetLogger(ctx) at the log site. The handler is the one nobody can forget.
package ctxscope

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"

	"github.com/psyb0t/ctxerrors"
)

type scopeKey struct{}

// attrPairWidth is the key and value each attribute takes in a flattened slice.
const attrPairWidth = 2

// Value is what a scope attribute may hold. Values become slog attributes and
// JSON, and anything wider has no sane rendering in either.
type Value interface {
	~string | ~bool |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Scope is the attribute map carried on a context.
type Scope map[string]any

// Attribute is one key/value pair headed into a scope. The fields are
// unexported so Attr is the only way to build one, which is what keeps the
// Value constraint despite the field being an any — a variadic Set cannot hold
// mixed instantiations of a generic type, so the field has to be one.
type Attribute struct {
	key   string
	value any
}

// Attr builds a scope attribute, rejecting at compile time anything Value does
// not cover.
func Attr[T Value](key string, value T) Attribute {
	return Attribute{key: key, value: value}
}

// Get returns a copy of the scope carried by ctx, safe to mutate.
func Get(ctx context.Context) Scope {
	current, _ := ctx.Value(scopeKey{}).(Scope)

	out := make(Scope, len(current))
	maps.Copy(out, current)

	return out
}

// Set returns a new context carrying ctx's scope plus every attribute given.
// A key already set is replaced; no attributes returns ctx unchanged.
//
//	ctx = scope.Set(ctx,
//	    scope.Attr("request_id", requestID),
//	    scope.Attr("user_id", userID),
//	)
func Set(ctx context.Context, attrs ...Attribute) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	next := Get(ctx)
	for _, attr := range attrs {
		next[attr.key] = attr.value
	}

	return context.WithValue(ctx, scopeKey{}, next)
}

// Remove returns a new context with the named keys dropped. Unset keys are
// ignored; no keys returns ctx unchanged.
func Remove(ctx context.Context, keys ...string) context.Context {
	if len(keys) == 0 {
		return ctx
	}

	next := Get(ctx)
	for _, key := range keys {
		delete(next, key)
	}

	return context.WithValue(ctx, scopeKey{}, next)
}

// GetLogger returns slog.Default() with both tiers applied, sorted by key, the
// context tier winning collisions. The context carries attributes, never a
// logger — where output goes is slog's business, configured once at startup.
//
// Call it where you log rather than holding the result: a logger is a value, so
// one fetched before a Set or Remove keeps the attributes it was built with.
func GetLogger(ctx context.Context) *slog.Logger {
	merged := GetGlobal()
	maps.Copy(merged, Get(ctx))

	return slog.Default().With(flatten(merged)...)
}

// ToJSON marshals ctx's scope for an outbound header, queue message or
// subprocess env. FromJSON reads it back on the far side. Empty marshals to {}.
func ToJSON(ctx context.Context) ([]byte, error) {
	data, err := json.Marshal(Get(ctx))
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal scope")
	}

	return data, nil
}

// FromJSON returns a new context carrying ctx's scope plus what ToJSON wrote on
// the other side of a hop. Incoming keys win.
//
// Numbers come back as float64, JSON having one number type, so an int sent as
// 42 returns 42 but is no longer an int. Only matters if you type-assert.
func FromJSON(ctx context.Context, data []byte) (context.Context, error) {
	incoming := Scope{}
	if err := json.Unmarshal(data, &incoming); err != nil {
		return ctx, ctxerrors.Wrap(err, "unmarshal scope")
	}

	next := Get(ctx)
	maps.Copy(next, incoming)

	return context.WithValue(ctx, scopeKey{}, next), nil
}

// flatten returns s as slog's alternating key/value slice, sorted so log field
// order is stable across lines.
func flatten(s Scope) []any {
	if len(s) == 0 {
		return nil
	}

	out := make([]any, 0, len(s)*attrPairWidth)
	for _, key := range slices.Sorted(maps.Keys(s)) {
		out = append(out, key, s[key])
	}

	return out
}
