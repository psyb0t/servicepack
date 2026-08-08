package slogconf

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/slogging/handlers"
)

// The environment variables read when Options does not name others.
const (
	EnvVarNameLevel     = "LOG_LEVEL"
	EnvVarNameFormat    = "LOG_FORMAT"
	EnvVarNameAddSource = "LOG_ADD_SOURCE"
)

// Unexported aliases so existing call sites and tests keep reading the same.
const (
	envVarNameLevel     = EnvVarNameLevel
	envVarNameFormat    = EnvVarNameFormat
	envVarNameAddSource = EnvVarNameAddSource
)

const (
	defaultAddSource = false
	defaultLevel     = levelInfo
	defaultFormat    = handlers.DefaultFormat
)

// Options selects which environment variables configure logging and what they
// fall back to. The zero value reproduces this package's historical behaviour
// exactly — LOG_LEVEL / LOG_FORMAT / LOG_ADD_SOURCE, defaulting to info + text
// + no source — so the blank import is unaffected by this type existing.
//
// The names are FIELDS rather than struct tags because a struct tag is fixed at
// compile time: nothing can hand one a value at runtime. That is why a caller
// who needed different variable names previously had to copy this whole package
// rather than configure it.
type Options struct {
	// LevelEnvVar overrides the variable read for the log level.
	LevelEnvVar string

	// FormatEnvVar overrides the variable read for the output format.
	FormatEnvVar string

	// AddSourceEnvVar overrides the variable read for source reporting.
	AddSourceEnvVar string

	// DefaultLevel applies when the level variable is unset or empty.
	// Empty means "info".
	DefaultLevel string

	// DefaultFormat applies when the format variable is unset or empty.
	// Empty means "text".
	DefaultFormat string

	// DefaultAddSource applies when the source variable is unset or empty.
	DefaultAddSource bool
}

// withDefaults fills every unset field, so nothing downstream has to ask
// whether the caller supplied one.
func (o Options) withDefaults() Options {
	if o.LevelEnvVar == "" {
		o.LevelEnvVar = EnvVarNameLevel
	}

	if o.FormatEnvVar == "" {
		o.FormatEnvVar = EnvVarNameFormat
	}

	if o.AddSourceEnvVar == "" {
		o.AddSourceEnvVar = EnvVarNameAddSource
	}

	if o.DefaultLevel == "" {
		o.DefaultLevel = string(defaultLevel)
	}

	if o.DefaultFormat == "" {
		o.DefaultFormat = string(defaultFormat)
	}

	return o
}

type config struct {
	// Level is the configured NAME, kept for the startup log line.
	Level level

	// SlogLevel is that name resolved. Resolving during readConfig rather than
	// after it keeps every validation in one place, so the order failures
	// surface in is the order the settings are read.
	SlogLevel slog.Level

	Format    handlers.Format
	AddSource bool
}

func (c config) log() {
	slog.Debug(
		"slogconf: configured",
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

// Init reconfigures the default logger, reading the variables named by opts.
//
// The blank import already configures logging with the default names, so this
// is only needed to change them or their fallbacks:
//
//	slogconf.Init(slogconf.Options{
//	    LevelEnvVar:  "MYAPP_LOG_LEVEL",
//	    FormatEnvVar: "MYAPP_LOG_FORMAT",
//	})
//
// Call it as early in main as possible. It replaces slog's default logger, so
// any logger already derived through slog.Logger.With keeps pointing at the
// previous handler chain — the same ordering caveat AddHandler carries.
func Init(opts Options) error {
	return configureWith(opts)
}

func configure() error {
	return configureWith(Options{})
}

func configureWith(opts Options) error {
	opts = opts.withDefaults()

	c, err := readConfig(opts)
	if err != nil {
		return err
	}

	handler, err := handlers.NewStd(handlers.Options{
		Format:    c.Format,
		Level:     c.SlogLevel,
		AddSource: c.AddSource,
	})
	if err != nil {
		return ctxerrors.Wrap(err, "failed to create log handler")
	}

	// Installed as the fan-out's slot 0 rather than bare, so SetOutput has a
	// slot to swap and AddSink has somewhere to append.
	SetHandlers(handler)

	c.log()

	return nil
}

// readConfig resolves the three settings from the environment.
//
// Deliberately reads the environment directly instead of going through a
// struct-tag config loader. The tag is what made the variable names
// unchangeable without forking this package, and it is the entire reason Options
// exists. Nothing is lost by reading here: these are three plain variables with
// no validation beyond what getSlogLevel and parseFormat already do, and no
// config file has ever been involved.
func readConfig(opts Options) (config, error) {
	// Level before format, deliberately: with both misconfigured the level
	// error is the one that surfaces, and a test pins that ordering so it
	// cannot drift silently.
	levelName := level(envOrDefault(opts.LevelEnvVar, opts.DefaultLevel))

	slogLevel, err := getSlogLevel(levelName)
	if err != nil {
		return config{}, ctxerrors.Wrap(err, "failed to get log level")
	}

	logFormat, err := parseFormat(envOrDefault(opts.FormatEnvVar, opts.DefaultFormat))
	if err != nil {
		return config{}, err
	}

	c := config{
		Level:     levelName,
		SlogLevel: slogLevel,
		Format:    logFormat,
		AddSource: opts.DefaultAddSource,
	}

	raw, ok := os.LookupEnv(opts.AddSourceEnvVar)
	if !ok || raw == "" {
		return c, nil
	}

	addSource, err := strconv.ParseBool(raw)
	if err != nil {
		return config{}, ctxerrors.Wrapf(
			ErrInvalidLogAddSource, "%s=%q", opts.AddSourceEnvVar, raw,
		)
	}

	c.AddSource = addSource

	return c, nil
}

// envOrDefault treats an empty value as unset. An exported but empty
// LOG_LEVEL= in a shell profile means "I did not set this", not "configure me
// with the empty string" — which would fail validation and panic the process at
// import time.
func envOrDefault(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}

	return fallback
}
