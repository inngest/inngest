# `cmd/doctor` — agent guidance

Diagnostic subcommand surface under `inngest alpha doctor`. Each subcommand is one "check"; bare `doctor` runs them all and prints a summary. Checks fall into two rough buckets:

- **Probe checks** (e.g. `healthcheck`): cheap, fast, side-effect-free. Designed to be invoked repeatedly — often from a docker-compose `healthcheck:` directive.
- **Diagnostic checks** (e.g. a future `configcheck`, `depscheck`): may load files, hit databases, or do real work. Designed for human/CI invocation, not loop polling.

The first-class motivation for the surface was replacing `curl` in compose healthchecks so the published image can drop curl and eventually move to distroless. Don't let that history constrain checks whose purpose is diagnostic.

## Adding a new check

1. Create `cmd/doctor/<name>/cmd.go` exposing `func Command() *cli.Command`.
2. Append `<name>.Command()` to the `Commands:` slice in `cmd/doctor/cmd.go`.

That's it. Do **not** create a separate `[]Check{...}` registry. The `Commands:` slice **is** the registry — the parent's default `Action` (`runAllChecks`) iterates it.

## Universal contract — every check must

- Be **silent on success** (no stdout, no stderr).
- On failure, print one line per problem to **stderr**, then return `cli.Exit("", 1)`. Do **not** return a plain `error` — `cmd/root.go` prints non-cli errors to stdout, which pollutes consumers (compose healthcheck logs, CI output).
- Reuse existing `INNGEST_*` env vars (e.g. `INNGEST_HOST`, `INNGEST_PORT`, `INNGEST_CONNECT_GATEWAY_PORT`) via `Sources: cli.EnvVars(...)`. Do **not** introduce new env vars without explicit user approval.
- Be non-interactive: no TTY prompts, no stdin reads.

## Routing

When a user runs `inngest alpha doctor <name>`, urfave/cli routes directly to that subcommand's `Action`; `runAllChecks` is not invoked. This is what preserves silent-on-success for the probe-check use case. When a user runs bare `inngest alpha doctor`, `runAllChecks` calls every subcommand's `Action` in turn and prints a human summary.

## Tests

Co-locate `cmd_test.go` next to each check. **Required** in any test file that exercises a check end-to-end:

```go
func init() {
    cli.OsExiter = func(int) {} // cli.Exit otherwise calls os.Exit and kills the test process
}
```

Drive the command with `cmd.Run(ctx, []string{"<name>", "--flag=value", ...})` and assert on the returned error. For HTTP probes, use `httptest.Server`.
