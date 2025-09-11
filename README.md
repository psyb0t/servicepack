# ServicePack

A Go framework for building concurrent service applications without the usual boilerplate bullshit.

Write once, deploy everywhere - run your entire stack locally for debugging or distribute services across machines with a single env var. No Docker Compose nightmares or managing 47 separate repos.

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

This generates a service skeleton in `internal/pkg/services/myservice/` with configuration support and automatically registers it. No manual wiring required.

### 2. Edit the Generated Service

Implement your business logic in the generated `Run()` method and any cleanup in `Stop()`. The generated service includes configuration parsing using environment variables.

### 3. Build and Run

```bash
make build
./servicepack
```

All services start automatically. If one dies, they all die. Simple and predictable (for now - see TODOs for planned retry and failure tolerance features).

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
- **Fail-Fast**: One service shits the bed, everything dies - no fucking zombie processes (for now - see TODOs for retry mechanisms)
- **Goroutines**: Services run in parallel because that's what the fuck Go was made for

## What You Fucking Get

- **Zero Bullshit**: Implement 3 methods, you're done
- **Auto-Discovery**: Framework finds your shit automatically without you lifting a finger

## Service Example

```go
type Config struct {
    Value string `env:"MYSERVICE_VALUE"`
}

type MyService struct{
    config   Config
    stopOnce sync.Once
}

func New() (*MyService, error) {
    cfg := Config{}
    
    gonfiguration.SetDefaults(map[string]any{
        "MYSERVICE_VALUE": "default-value",
    })
    
    if err := gonfiguration.Parse(&cfg); err != nil {
        return nil, ctxerrors.Wrap(err, "failed to parse myservice config")
    }
    
    return &MyService{
        config: cfg,
    }, nil
}

func (s *MyService) Name() string {
    return "myservice"
}

func (s *MyService) Run(ctx context.Context) error {
    defer s.Stop(ctx)

    errCh := make(chan error, 1)

    // Simulate some background work that might fail
    go func() {
        time.Sleep(10 * time.Second)
        errCh <- errors.New("service got fucked")
    }()

    select {
    case <-ctx.Done():
        return nil  // Graceful shutdown
    case err := <-errCh:
        return err  // Service failure
    }
}

func (s *MyService) Stop(ctx context.Context) error {
    s.stopOnce.Do(func() {
        // Cleanup logic runs only once
    })

    return nil
}
```

## Configuration

Currently minimal - uses environment variables:

**Environment (psyb0t/common-go/env):**

- `ENV=dev`: Environment setting (defaults to `prod` if not set)

**Service Control:**

- `SERVICEPACK_ENABLEDSERVICES=service1,service2`: Comma-separated list of services to run (if empty, runs all services)

This is where the magic happens - one codebase that works everywhere:

**Development**: Run the entire fucking stack locally with all services. Debug across services in a single session without Docker Compose hell.

**Production**: Deploy the same binary, control what runs per machine:

- `SERVICEPACK_ENABLEDSERVICES=user-service` on machine 1
- `SERVICEPACK_ENABLEDSERVICES=payment-service` on machine 2
- `SERVICEPACK_ENABLEDSERVICES=notification-service` on machine 3

**Testing**: Mix and match services as needed:

- `SERVICEPACK_ENABLEDSERVICES=user-service,payment-service` for integration tests

No separate repos, no Docker file madness, no Kubernetes clusterfuck - just env vars.

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
