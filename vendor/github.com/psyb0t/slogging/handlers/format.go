package handlers

import (
	"io"
	"log/slog"

	"github.com/psyb0t/ctxerrors"
)

// Format selects the wire shape of a record. It is a named type rather than a
// bare string so an unknown value is a compile-time mistake at every call site
// that uses the constants, and a single validated error everywhere else.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// DefaultFormat matches what slog itself does when you build a TextHandler by
// hand. JSON is the production choice; text is the one you want at a terminal.
const DefaultFormat = FormatText

// newHandler builds the stdlib handler for this format.
func (f Format) newHandler(
	w io.Writer,
	opts *slog.HandlerOptions,
) (slog.Handler, error) {
	switch f {
	case FormatJSON:
		return slog.NewJSONHandler(w, opts), nil
	case FormatText:
		return slog.NewTextHandler(w, opts), nil
	default:
		return nil, ctxerrors.Wrapf(ErrInvalidFormat, "%s", f)
	}
}
