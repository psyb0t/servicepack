package slogconf

import (
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/slogging/handlers"
)

// parseFormat validates a configured format name into the handlers type.
//
// The name arrives from the environment as free text, so it is validated here
// rather than handed straight to handlers.Format: a typo in LOG_FORMAT should
// fail startup with ErrInvalidLogFormat, not build a handler that rejects every
// record later.
func parseFormat(name string) (handlers.Format, error) {
	switch handlers.Format(strings.ToLower(name)) {
	case handlers.FormatJSON:
		return handlers.FormatJSON, nil
	case handlers.FormatText:
		return handlers.FormatText, nil
	default:
		return "", ctxerrors.Wrapf(ErrInvalidLogFormat, "%s", name)
	}
}
