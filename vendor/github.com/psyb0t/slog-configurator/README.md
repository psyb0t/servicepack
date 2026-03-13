# slog-configurator

Welcome to `slog-configurator`, the badass sidekick for your logging adventures with Go's stdlib `log/slog`! This is the spiritual successor to [`logrus-configurator`](https://github.com/psyb0t/logrus-configurator), upgraded to use the standard library's structured logging package.

## What's This Shit About?

`slog-configurator` is a Go package that whips your `slog` logger into shape without you breaking a sweat. Want to set the log level? Bam! Prefer JSON over plain text? Wham! Want to know who called the logger? Boom! It's got you covered.

It also handles **stdout/stderr separation** automatically - info and debug go to stdout, warnings and errors go to stderr. Because that's how proper logging works.

## Features

- **No-nonsense log level setting** (debug, info, warn, error - the slog way).
- **Formatting logs like a boss** with JSON or text formats - keep it structured or keep it simple.
- **Source reporting** for when you need to backtrack who messed up. It's like `CSI` for your code.
- **Automated configuration** using environment variables, because who has time for manual setup?
- **stdout/stderr separation** - errors and warnings go to stderr, everything else goes to stdout.
- **Custom handler support** for when you need to take full control of your logging pipeline.
- **Handler stacking** via `AddHandler()` - add extra handlers without nuking the existing setup, just like logrus hooks.

## Usage Example

Ready to rock with `slog-configurator`? Check this out.

### main.go

```go
package main

import (
	"log/slog"

	_ "github.com/psyb0t/slog-configurator"
)

func main() {
	slog.Debug("this is a debug message", "key", "value", "number", 42)
	slog.Info("this is an info message", "user", "psyb0t", "action", "testing")
	slog.Warn("this is a warning message", "warning_code", "W001")
	slog.Error("this is an error message", "error_code", "E001", "details", "something went wrong")
}
```

### Crank It Up

Get your environment dialed in:

```bash
export LOG_LEVEL="debug"        # Choose the verbosity level.
export LOG_FORMAT="text"        # Pick your poison: json or text.
export LOG_ADD_SOURCE="true"    # Decide if you want to see source location in logs.
```

Unleash the beast with:

```bash
go run main.go
```

And let the good times roll with the output:

```plaintext
time=2026-03-13T20:34:53.184Z level=DEBUG source=main.go:10 msg="this is a debug message" key=value number=42
time=2026-03-13T20:34:53.184Z level=INFO source=main.go:11 msg="this is an info message" user=psyb0t action=testing
time=2026-03-13T20:34:53.184Z level=WARN source=main.go:12 msg="this is a warning message" warning_code=W001
time=2026-03-13T20:34:53.184Z level=ERROR source=main.go:13 msg="this is an error message" error_code=E001 details="something went wrong"
```

Wanna switch it up? Change the environment variables to mix the brew.

```bash
export LOG_LEVEL="warn"
export LOG_FORMAT="json"
export LOG_ADD_SOURCE="false"
```

Then let it simmer with:

```bash
go run main.go
```

And enjoy the sweet sound of (almost) silence:

```plaintext
{"time":"2026-03-13T20:34:53.296Z","level":"WARN","msg":"this is a warning message","warning_code":"W001"}
{"time":"2026-03-13T20:34:53.296Z","level":"ERROR","msg":"this is an error message","error_code":"E001","details":"something went wrong"}
```

Whether you're in for a riot or a silent disco, `slog-configurator` is your ticket. (check out all of the supported levels in [`level.go`](level.go))

## Advanced: Handler Management

The default handler is always a `FanOutHandler` that dispatches to all registered handlers. On init, it contains a single `MultiWriterHandler` (the stdout/stderr splitter). You can stack more handlers on top or replace them all.

```go
import slogconfigurator "github.com/psyb0t/slog-configurator"

// Add a handler without fucking up the existing setup (like logrus AddHook)
slogconfigurator.AddHandler(myDBHandler)

// Replace ALL handlers - go full nuclear (like logrus SetHooks)
slogconfigurator.SetHandlers(myHandler1, myHandler2)
```

### Adding Extra Handlers

`AddHandler` stacks a new handler on top of the existing ones. Every log record gets dispatched to all handlers. Call it multiple times and they all stack up:

```go
import slogconfigurator "github.com/psyb0t/slog-configurator"

// These all fire for every log record alongside the default stdout/stderr handler
slogconfigurator.AddHandler(myMetricsHandler)
slogconfigurator.AddHandler(mySlackAlertHandler)
slogconfigurator.AddHandler(myDBHandler)
```

### Replacing All Handlers

`SetHandlers` replaces everything in the fan-out, including the default stdout/stderr handler:

```go
import slogconfigurator "github.com/psyb0t/slog-configurator"

// Only these handlers will fire from now on
slogconfigurator.SetHandlers(myCustomHandler1, myCustomHandler2)
```

### Custom MultiWriterHandler

Need custom stdout/stderr writers? Create your own and add it or set it:

```go
import slogconfigurator "github.com/psyb0t/slog-configurator"

handler, err := slogconfigurator.NewMultiWriterHandler(
	"json",
	&slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true},
	myStdoutWriter,
	myStderrWriter,
)

slogconfigurator.SetHandlers(handler)
```

### FanOutHandler Directly

Build complex pipelines by composing handlers:

```go
import slogconfigurator "github.com/psyb0t/slog-configurator"

fanOut := slogconfigurator.NewFanOutHandler(handler1, handler2, handler3)
slogconfigurator.SetHandlers(fanOut)
```

Perfect for when you need to:
- Send errors to external monitoring systems
- Log to databases or message queues
- Route different log levels to different destinations
- Build complex logging pipelines

The `MultiWriterHandler` handles stderr/stdout separation automatically, so your custom setup plays nice with the standard streams.

## Testing & Quality

This package is tested harder than a Nokia 3310. This shit's got **90%+ test coverage** because nobody fucks around with quality here.

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage check (fails if below 90%)
make test-coverage

# Run linting
make lint

# Run everything (dependencies, linting, tests)
make all
```

### What Gets Tested

**Core Shit:**
- Environment variable parsing and configuration
- All log levels (debug, info, warn, error)
- JSON and text formatting
- MultiWriterHandler stdout/stderr routing
- Error handling for fucked up configurations

**Advanced Shit:**
- `SetHandlers()`, `AddHandler()`, and `NewMultiWriterHandler()` API functions
- `FanOutHandler` dispatching and handler stacking
- Handler `WithAttrs()` and `WithGroup()` propagation
- Custom writers and level filtering
- Configuration edge cases and error scenarios

**Testing Philosophy:**
- Table-driven tests for consistency and maintainability
- Proper `require` vs `assert` patterns for better debugging
- 90% minimum coverage enforced in CI/CD
- Comprehensive edge case coverage

And that's damn it. You've just pimped your logger with military-grade reliability!

## Contribute

Got an idea? Throw in a PR! Found a bug? Raise an issue! Let's make `slog-configurator` as tight as your favorite jeans.

## License

It's MIT. Free as in 'do whatever the hell you want with it', just don't blame me if shit hits the fan.
