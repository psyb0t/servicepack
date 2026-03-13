package slogconfigurator

import (
	"fmt"
	"log/slog"
	"strings"
)

type level string

const (
	levelDebug level = "debug"
	levelInfo  level = "info"
	levelWarn  level = "warn"
	levelError level = "error"
)

func getSlogLevel(lvl level) (slog.Level, error) {
	switch strings.ToLower(string(lvl)) {
	case string(levelDebug):
		return slog.LevelDebug, nil
	case string(levelInfo):
		return slog.LevelInfo, nil
	case string(levelWarn):
		return slog.LevelWarn, nil
	case string(levelError):
		return slog.LevelError, nil
	}

	return 0, fmt.Errorf("%s: %w", lvl, ErrInvalidLogLevel)
}
