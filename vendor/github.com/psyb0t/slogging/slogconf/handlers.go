package slogconf

import (
	"log/slog"

	"github.com/psyb0t/slogging/handlers"
)

// outputSlot is where the process's own output handler lives in the fan-out.
//
// Reserving a slot is what makes "swap where my logs print" expressible without
// also discarding the sinks. The alternative — one flat list and a single Add
// verb — is why stacking a second console handler used to double-print every
// line, and why replacing it used to take the ring and the shipper with it.
const outputSlot = 0

// SetOutput replaces the process's output handler, leaving every sink added
// with AddSink in place.
//
// This is the call for "send my logs somewhere else". Adding a second handler
// that also writes to the terminal does NOT replace the first one — both would
// receive every record and print it twice.
//
// Call it early. slog.Logger.With and WithGroup snapshot the handler chain at
// the moment they are called, so a logger derived BEFORE this never sees the
// new output.
//
// Reports whether the default logger was one this package configured. False
// means something else replaced slog's default, so there was no slot to swap
// and the whole chain was replaced instead — worth noticing rather than
// discovering through absent logs.
func SetOutput(handler slog.Handler) bool {
	fanOut, ok := slog.Default().Handler().(*handlers.FanOutHandler)
	if !ok {
		SetHandlers(handler)

		return false
	}

	chain := fanOut.Handlers()
	if len(chain) == 0 {
		SetHandlers(handler)

		return true
	}

	chain[outputSlot] = handler
	slog.SetDefault(slog.New(handlers.NewFanOut(chain...)))

	return true
}

// AddSink stacks an extra handler alongside the existing ones — a ring buffer,
// a shipper, a test capture — without touching where the process already
// prints.
//
// Call it early, for the same snapshotting reason as SetOutput.
//
// Reports whether the default logger was one this package configured. False
// means something else replaced slog's default and the sink was stacked onto
// that instead: still added, but the stdout/stderr split this package sets up
// is gone.
func AddSink(handler slog.Handler) bool {
	current := slog.Default().Handler()

	fanOut, ok := current.(*handlers.FanOutHandler)
	if !ok {
		slog.SetDefault(slog.New(handlers.NewFanOut(current, handler)))

		return false
	}

	slog.SetDefault(slog.New(handlers.NewFanOut(append(fanOut.Handlers(), handler)...)))

	return true
}

// SetHandlers replaces the ENTIRE chain — output slot and every sink.
//
// This is the escape hatch, not the everyday call: it discards anything
// AddSink put there. Reach for it when you genuinely want to start over, which
// in practice means tests pointing everything at a buffer or io.Discard. To
// change where logs print while keeping your sinks, use SetOutput.
func SetHandlers(hs ...slog.Handler) {
	slog.SetDefault(slog.New(handlers.NewFanOut(hs...)))
}
