package slogconfigurator

import (
	"fmt"
	"io"
	"log/slog"
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

	return nil, fmt.Errorf("%s: %w", f, ErrInvalidLogFormat)
}
