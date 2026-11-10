# Services

- **SV1:** `Run(ctx)` must respect cancellation; `Stop(ctx)` must do bounded cleanup.
- **SV2:** `Dependent` only orders launch. Use `ReadyNotifier` when a dependency must actually be ready before its dependent begins.
- **SV3:** The first non-allowed terminal service failure starts shutdown. `Retryable` retries before that; `AllowedFailure` is non-fatal only after retries fail.
- **SV4:** Keep the `ctxscope` context the manager passes in, so logs keep the `binary`, `commit`, `version` and `service` fields. Wrap operational errors with `ctxerrors`.
- **SV5:** Use `gonfiguration` for typed environment config; do not scatter direct `os.Getenv` calls.
