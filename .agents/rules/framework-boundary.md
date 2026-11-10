# Framework boundary

- **FB1:** Project extension points: `internal/pkg/services/` (business services; registration output is generated), `cmd/init.go` (lifecycle hooks and log-handler additions), `cmd/commands.go` (application Cobra commands), `Makefile`, `scripts/make/`, `Dockerfile` and `Dockerfile.dev` (project build and deploy), `docs/` and `tests/`.
- **FB2:** Framework files, replaced by a normal `make servicepack-update`: `internal/app/`, `internal/pkg/service-manager/`, `pkg/runner/`, `cmd/main.go`, `Makefile.servicepack`, `scripts/make/servicepack/` and `Dockerfile.servicepack*`. Add an extension point rather than expecting downstreams to patch these.
- **FB3:** The update's mechanical floor is `EXCLUDE_ARGS` in `scripts/make/servicepack/do_update.sh`. The shipped `.servicepackupdateignore` holds the default opt-outs, but an existing downstream never receives new lines in it, so protection every project needs goes in the floor.
- **FB4:** A change to the floor or the shipped ignore file also updates the "What is preserved" list in `docs/framework-updates.md`.
