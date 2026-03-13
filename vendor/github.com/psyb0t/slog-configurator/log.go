package slogconfigurator

import (
	"fmt"
	"log/slog"

	"github.com/psyb0t/gonfiguration"
)

const (
	envVarNameLevel     = "LOG_LEVEL"
	envVarNameFormat    = "LOG_FORMAT"
	envVarNameAddSource = "LOG_ADD_SOURCE"
)

const (
	defaultAddSource = false
	defaultLevel     = levelInfo
	defaultFormat    = formatText
)

type config struct {
	Level     level  `env:"LOG_LEVEL"`
	Format    format `env:"LOG_FORMAT"`
	AddSource bool   `env:"LOG_ADD_SOURCE"`
}

func (c config) log() {
	slog.Debug(
		"slog-configurator: configured",
		slog.String("level", string(c.Level)),
		slog.String("format", string(c.Format)),
		slog.Bool("addSource", c.AddSource),
	)
}

//nolint:gochecknoinits
func init() {
	if err := configure(); err != nil {
		panic(err)
	}
}

func configure() error {
	setDefaults()

	c := config{}
	if err := gonfiguration.Parse(&c); err != nil {
		return fmt.Errorf("failed to parse log config: %w", err)
	}

	slogLevel, err := getSlogLevel(c.Level)
	if err != nil {
		return fmt.Errorf("failed to get log level: %w", err)
	}

	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     slogLevel,
	}

	handler, err := NewMultiWriterHandler(c.Format, opts, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create log handler: %w", err)
	}

	slog.SetDefault(slog.New(NewFanOutHandler(handler)))

	c.log()

	return nil
}

func setDefaults() {
	gonfiguration.SetDefaults(map[string]any{
		envVarNameLevel:     defaultLevel,
		envVarNameFormat:    defaultFormat,
		envVarNameAddSource: defaultAddSource,
	})
}
