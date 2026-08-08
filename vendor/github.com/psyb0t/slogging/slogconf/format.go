package slogconf

import (
	"io"
	"log/slog"

	"github.com/psyb0t/ctxerrors"
)

type format string

const (
	formatJSON format = "json"
	formatText format = "text"
)

func getSlogHandler(f format, w io.Writer, opts *slog.HandlerOptions) (slog.Handler, error) {
	switch f {
	case formatJSON:
		return slog.NewJSONHandler(w, opts), nil
	case formatText:
		return slog.NewTextHandler(w, opts), nil
	}

	return nil, ctxerrors.Wrapf(ErrInvalidLogFormat, "%s", f)
}
