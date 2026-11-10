# Testing

- **T1:** Tests run through the Make targets with Docker access, because they may use Testcontainers. `make test-coverage` enforces the 90% floor.
- **T2:** Service-manager tests reset singleton state and add controlled fake services. They do not call the generated `services.Init()` unless testing generation itself. Use controlled contexts and channels to test ordering, readiness, retry, failure and shutdown deterministically.
- **T3:** `pkg/runner` tests drive parent-context cancellation and fake runnables, not host signals. The runner's default whole-application shutdown deadline is `RUNNER_SHUTDOWNTIMEOUT=10s`.
