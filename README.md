# ServicePack

A Go framework for building concurrent service applications without the usual boilerplate bullshit.

## Why This Shit Exists

Tired of writing the same goddamn service boilerplate every time? ServicePack handles the boring lifecycle bullshit so you can focus on actual work instead of reinventing goroutine management for the millionth fuckin' time.

## The Service Interface

Every service implements this simple interface:

```go
type Service interface {
    Name() string
    Run(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

That's it. Three fucking methods and you're done.

## How To Use This Shit

### 1. Create a New Service

```bash
make service NAME=myservice
```

This generates a service skeleton in `internal/pkg/services/myservice/` and automatically registers it. No manual wiring required.

### 2. Edit the Generated Service

Implement your business logic in the generated `Run()` method and any cleanup in `Stop()`.

### 3. Build and Run

```bash
make build
./servicepack
```

All services start automatically. If one dies, they all die. Simple and predictable.

## Development Workflow

```bash
# Install dependencies
make dep

# Create a new service
make service NAME=notification

# Edit the generated service file
# internal/pkg/services/notification/notification.go

# Build the application
make build

# Run tests
make test

# Run with coverage
make test-coverage

# Lint the code
make lint

# Run in development mode (Docker)
make run-dev
```

## How This Shit Works

- **Singletons**: One fucking app instance, one service manager - no race condition bullshit
- **Code Generation**: Scans your code and wires up services so you don't have to do jack shit
- **Fail-Fast**: One service shits the bed, everything dies - no fucking zombie processes
- **Goroutines**: Services run in parallel because that's what the fuck Go was made for

## What You Fucking Get

- **Zero Bullshit**: Implement 3 methods, you're done
- **Auto-Discovery**: Framework finds your shit automatically without you lifting a finger

## Service Example

```go
type MyService struct{}

func New() *MyService {
    return &MyService{}
}

func (s *MyService) Name() string {
    return "myservice"
}

func (s *MyService) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil  // Graceful shutdown
        default:
            // Do your work here
            time.Sleep(time.Second)
        }
    }
}

func (s *MyService) Stop(ctx context.Context) error {
    // Cleanup logic
    return nil
}
```

## Configuration

Currently minimal - uses environment variables:

**Environment (psyb0t/common-go/env):**

- `ENV=dev`: Environment setting (defaults to `prod` if not set)

**Logging (psyb0t/logrus-configurator):**

- `LOG_LEVEL=debug`: Logging level (trace, debug, info, warn, error, fatal, panic)
- `LOG_FORMAT=text`: Log output format (`text` or `json`)
- `LOG_CALLER=true`: Include caller info in logs

## Requirements

- Go 1.24+
- Docker (for development environment)
- Make

## Shit That Needs Doing

- **Service Retry**: When a service shits itself, check retry count and restart the fucker if it hasn't hit the limit yet
- **Allowed Failures**: Let some services die without killing everything - useful for one-shot jobs like migrators that run once and fuck off
- **Service Dependencies**: Let services say "I need this other shit to start first" so database comes up before API and shit
- **Health Checks**: Built-in endpoints to check if services are alive or dead with timeouts and failure limits
- **Management API**: HTTP endpoint to see what's running and control the bastards (start/stop/restart individual services)
- **Metrics**: Track startup times, failure counts, restart counts and optionally export to Prometheus
- **Service Communication**: Built-in message passing so services can talk to each other instead of figuring that shit out themselves

## License

MIT - Do whatever the fuck you want with it.
