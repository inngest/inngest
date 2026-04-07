# Upstream Compatibility Check

## Trigger

The workflow ([upstream_compat.yml](../workflows/upstream_compat.yml)) fires on any PR to `main` that touches `**.go`, `go.mod`, or `go.sum`. If a PR only changes markdown, YAML, or TypeScript, it doesn't run.

## What happens on the runner

The workflow has 4 steps:

**Step 1 — Checkout both repos side-by-side.**
The runner checks out the inngest/ PR branch (with full history via `fetch-depth: 0`) and the monorepo/ `develop` branch. The monorepo checkout uses the existing `AUTOMATED_UPSTREAM_TOKEN` secret — the same one already used by `dispatch_upstream.yml`.

```
$GITHUB_WORKSPACE/
├── inngest/     ← PR branch
└── monorepo/    ← develop branch (private, checked out via token)
```

**Step 2 — Set up Go.** Uses the Go version from inngest/go.mod.

**Step 3 — Run `check.sh`.** This is the orchestrator. It runs three independent checks sequentially, captures all output to temp files (never stdout), and assembles a sanitized markdown report.

**Step 4 — Post the report as a PR comment.** Uses `marocchino/sticky-pull-request-comment` with a `header: upstream-compat` key, so subsequent pushes to the same PR *update* the existing comment instead of creating new ones.

The step has `continue-on-error: true` and the script always exits 0, so **the action never blocks the build**.

## The three checks

### 1. Classify (`go run ./classify --ci`)

Compares every exported Go symbol between two versions of inngest/:
- **Old**: whatever inngest/ version is currently vendored in monorepo/ (`monorepo/vendor/github.com/inngest/inngest/`)
- **New**: the PR branch

How it works:
1. `astdiff.ExtractExports()` walks every `.go` file (skipping vendor/, testdata/, tests) and parses the AST. It extracts every exported function, type, interface, struct, const, var, and method into a map keyed by `"pkg.Name"` (e.g. `"queue.Queue.Enqueue"`).
2. `astdiff.DiffSymbols()` compares old vs new maps. Anything in old but not new = **removed**. Anything in both but with different signatures = **modified**. Anything in new but not old = **added**.
3. `iface.ClassifyChanges()` cross-references each change against a **registry of 12 watched interfaces** — the interfaces that monorepo/ actually implements (Executor, Queue, QueueShard, RunService, etc.). This is the key step:
   - Removed or modified symbol on a watched interface → **BREAKING**
   - New method on an interface that monorepo *implements* → **BREAKING** (monorepo's concrete types won't satisfy the interface anymore)
   - New method on an interface monorepo only *calls* → **ADDITIVE**
   - New standalone function/type/struct → **ADDITIVE**
   - Everything else → **SAFE**

Exit codes: 0=SAFE, 1=ADDITIVE, 2=BREAKING.

In `--ci` mode, only breaking changes are listed individually. Additive and safe changes show just a count (e.g. `ADDITIVE CHANGES: 1770 (not listed individually)`).

### 2. Interface check (`go run ./ifacecheck --ci`)

A focused check on the 12 watched interfaces specifically. While classify looks at *all* symbols, ifacecheck zooms in on the interfaces monorepo/ is known to implement and reports method-level diffs:

```
  BREAKING execution/queue.Queue [critical]:
    + NewMethod
      func(context.Context, string) error
    ~ Enqueue
      was: func(context.Context, QueueItem) error
      now: func(context.Context, QueueItem, EnqueueOpts) error
    3 implementations affected
```

It iterates the registry, extracts symbols from just that interface's package in both old and new, compares the interface signature, and if different, drills into individual method additions (`+`), removals (`-`), and modifications (`~`).

In `--ci` mode, the `printImplementors` block (which lists monorepo/ file paths) is replaced with just `"N implementations affected"`.

### 3. Compile check

The most direct test: can monorepo/ actually compile against this PR's inngest/?

1. Creates a temporary `go.work` file that tells Go to resolve `github.com/inngest/inngest` from the local PR checkout instead of the vendored copy:
   ```
   go 1.24
   use (
       .           // monorepo/
       ../inngest  // PR branch
   )
   ```
2. Lists all monorepo/ packages (excluding `test/integration/`)
3. Runs `GOWORK=go.work go build ./...` on monorepo/, capturing stderr
4. Processes build errors: strips file paths (which would reveal monorepo structure), keeps only error descriptions that reference `github.com/inngest/inngest` types

Output example:
```
42 compile errors total, 12 referencing inngest/ types.

Compile error: cannot use x (variable of type *queue.QueueItem) as *queue.QueueItem value
```

## Overall classification

`check.sh` takes the worst result across all three checks. If classify says ADDITIVE but the compile check fails, the overall classification escalates to BREAKING.

## The report

The three sections are assembled into `upstream-report.md`:

```markdown
## Upstream Compatibility: BREAKING 🔴

### Symbol Changes
(classify output — only breaking listed individually)

### Interface Changes
(ifacecheck output — method-level diffs)

### Compile Check
(PASS/FAIL + filtered error descriptions)

---
> Checks whether this PR's changes are compatible with downstream consumers.
> Breaking changes will need corresponding downstream updates before the next vendor cycle.
```

## Redaction (defense-in-depth)

The monorepo is private but checked out on a public repo's CI runner. Five layers prevent leaking its structure:

| Layer | Where | What it does |
|-------|-------|-------------|
| 1. `--ci` flag | Go tools | Suppresses monorepo file paths, replaces implementor lists with counts |
| 2. Temp file capture | check.sh | All raw tool output goes to `/tmp` files, never stdout |
| 3. Build error filter | check.sh `sanitize_build_errors()` | Strips file paths from `go build` stderr, keeps only error descriptions referencing inngest/ types |
| 4. Final sed guard | check.sh step 6 | Scans the finished report for any remaining `$MONOREPO_DIR` or `monorepo/...` references and replaces with `[redacted]` |
| 5. Panic recovery | Go tools `defer recover()` | Catches panics and prints generic "internal error" instead of stack traces that might contain monorepo paths |

## Size guardrails

GitHub comments have a 65,536 character limit. Three layers prevent hitting it:

1. **CI-mode output suppression** — only breaking changes listed individually; additive/safe are counts
2. **Section truncation** — each tool's output is capped at 150 lines
3. **Final size guard** — if the report exceeds 60,000 chars, it's hard-truncated with a note pointing to workflow logs

## File inventory

```
.github/upstream-compat/
├── check.sh                        # Orchestrator shell script
├── go.mod                          # inngest.com/upstream-compat (stdlib only)
├── classify/main.go                # Symbol diff + classification
├── ifacecheck/main.go              # Interface method-set comparison
└── internal/
    ├── astdiff/astdiff.go          # Go AST symbol extraction and diffing
    └── iface/iface.go              # Interface registry (12 watched interfaces)

.github/workflows/
└── upstream_compat.yml             # GitHub Action workflow
```

The Go tools are a standalone module with zero external dependencies — only stdlib and `go/ast`. They're adapted from the local-only tools in `tools/` (same repo root), with the `--ci` redaction mode added.
