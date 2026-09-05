# Lessons Learned

## Debugging Approach

### 1. Practical repro beats static analysis
Build, run, query GQL, look at actual data. The span pipeline has too many layers (tracer → exporter → CQRS tree builder → GQL) for static analysis to be reliable. A 15-minute repro with targeted debug logging beats hours of code reading.

**Best debug points:** `ExportSpans` in `tracer_sqlc.go` (what gets written to SQLite) and `mapRootSpansFromRows` in `cqrs.go` (how the trace tree is built). Log `name`, `spanID`, `parentSpanID`, `dynamicSpanID`. Remove after diagnosis.

## Span & Trace Architecture

### 2. Orphaned spans corrupt the root
`mapRootSpansFromRows` sets `root` to the last span with no parent. A span with `parentSpanID == "0000000000000000"` (zero value) becomes a root candidate, **replacing** the real Run span and collapsing the hierarchy.

### 3. Span parent propagation requires queue metadata
Parent span references flow through `queue.Item.Metadata` via `meta.PropagationKey`. The `Carriers` field in `CreateSpanOptions` handles injection. When creating queue items outside the executor (e.g., checkpoint code), you must manually inject the parent span reference or execution spans will be orphaned.

### 4. Sync checkpoint must replicate async span hierarchy
In async mode, the executor creates all spans. In sync/checkpoint mode, `checkpoint.go` must create the same hierarchy:
- **Step span**: parent = Run span
- **Execution span**: parent = Step span (one per attempt)
- **Retry queue metadata**: must carry the step span reference

### 5. Two data stores: `trace_runs` (legacy) vs `spans` (preview)
- **`trace_runs`**: Populated by OTLP lifecycle hooks. Sync functions that complete without going async never fire `OnFunctionFinished`, leaving status stuck at QUEUED.
- **`spans`**: Populated by the internal tracer's `dbExporter`. Always up-to-date.

GQL `runs` list: `preview=false` → `trace_runs`, `preview=true` → `spans`. Always prefer spans-based data. The UI's `updateDynamicRunData` can overwrite correct span status with stale `trace_runs` status — use `runData.trace.status` not `runData.status`.

## SDK (inngest-js)

### 6. JS Error serialization for checkpoints
`JSON.stringify(new Error("x"))` produces `{}` because `name`/`message`/`stack` are non-enumerable. Always use `serializeError()` from `helpers/errors.ts`. The canonical pattern for transforming step results (success or error) is `stepRanHandler`, which calls `transformOutput` → runs middleware → serializes errors. Use it in any handler that produces step results before checkpointing.

### 7. Checkpoint request bodies can silently drop fields
If a field exists in the TypeScript type but isn't in the `JSON.stringify`/fetch body, the server receives zero/empty. Verify the actual `body:` object in the `fetch` call in `api.ts`.

### 8. `retries` = retry count, not total attempts
SDK sends `retries: N` (default 3) meaning N retries after the initial attempt = N+1 total. Checkpoint retry queue items start at `Attempt: 1`.

### 9. React `useState` + `useEffect` race
```typescript
const { value: flag } = booleanFlag('feature', true);  // async, defaults true
const [preview, setPreview] = useState(false);          // starts false
useEffect(() => setPreview(flag), [flag]);              // sets true after render
const query = useQuery({ enabled: flagReady, ... });    // fires with preview=false
```
Fix: initialize state to the default value, or compute directly without intermediate state.

## Go Server

### 10. httpv2 sync driver drops error info on non-opcode responses
When `httpv2.sync()` gets a non-opcode response (e.g., `function-rejected` at 400), `parseOpcodes` fails and returns a `UserError`. Returning `nil` for `DriverResponse` causes the executor to see `StatusCode: 0` and no error, finalizing as **Completed** instead of **Failed**. Fix: construct a proper `DriverResponse` with HTTP response data when `parseOpcodes` fails.

### 11. Sync runs should start as Running, not Queued
`executor.go`'s `Schedule()` defaults `DynamicStatus` to `StepStatusQueued`. For `RunModeSync`, the function is already executing — set `StepStatusRunning` instead.

## Dev Workflow

### 12. E2E test recipe
```bash
# Build (CGO_ENABLED=0 required without gcc)
CGO_ENABLED=0 go build -o /tmp/inngest-dev ./cmd/*.go

# Start dev server
/tmp/inngest-dev dev --no-poll --retry-interval 1

# Install SDK changes in bun-sync example
cd inngest-js/examples/bun-sync
bun remove inngest && (cd ../../packages/inngest && pnpm local:pack) && bun add inngest@../../packages/inngest/inngest.tgz

# Start example
INNGEST_DEV=1 bun run index.ts

# Query runs
curl -s 'http://localhost:8288/v0/gql' -H 'Content-Type: application/json' \
  -d '{"query":"{ runs(orderBy: [{field: QUEUED_AT, direction: DESC}], filter: {from: \"2025-01-01T00:00:00Z\"}) { edges { node { id status } } } }"}'

# Query trace tree (substitute RUN_ID)
curl -s 'http://localhost:8288/v0/gql' -H 'Content-Type: application/json' \
  -d '{"query":"{ runTrace(runID: \"<RUN_ID>\") { spanID name status attempts outputID childrenSpans { spanID name status attempts outputID childrenSpans { spanID name status outputID } } } }"}'
```

### 13. EXTEND spans with zero-value status overwrite real status
`UpdateSpan` always wrote `DynamicStatus: opts.Status` to EXTEND spans, even when Status was the zero value (`StepStatusUnknown`). In `GetSpanRuns`, spans are grouped by `run_id + dynamic_span_id` and processed in start_time order — the EXTEND span (later) overwrites the run span's real status (e.g., Running → Unknown). Then `ToFunctionRunStatus(RunStatusUnknown)` returns an error, and the resolver silently skips the run with `if err != nil { continue }`.

**Fix**: (1) Only write `DynamicStatus` in `UpdateSpan` when status is explicitly set (non-zero). (2) Skip `StepStatusUnknown` when merging statuses in `GetSpanRuns` as defense-in-depth.

**Detection signal**: Run appears in `preview: false` (trace_runs) but not `preview: true` (spans). Direct SQLite query shows all spans exist with correct parents.

### 14. Checkpoint retry enqueue must use the configured backoff
The checkpoint path (`checkpoint.go`) was enqueuing step retries at `time.Now()` (immediate), while the normal async path (`queue/process.go`) uses `q.backoffFunc(attempt)` which respects the configured backoff (table-based by default, linear if `--retry-interval` is set). Fix: plumb the backoff function from devserver → `CheckpointAPIOpts` → `checkpoint.Opts.BackoffFunc`, defaulting to `backoff.DefaultBackoff` when nil.

### 15. Stop dev server before running SDK tests
SDK unit tests use nock mocks. A running dev server on :8288 causes the SDK to auto-discover it and bypass mocks, failing all framework registration tests.

### 16. Vite can serve a stale module to one importer and a fresh one to another
Symptom: the fixture gallery's facts row reported `1 group, aggregated=true` from
`planCollapse`, while the `Canvas` *rendered on the same page* drew all 502
uncollapsed nodes. Same input, same function, two answers — which is impossible
in the model, and the vitest suite agreed with the gallery.

Cause: HMR had updated `Canvas.tsx` and the route module, but `collapse.ts` was
being served from a stale transform to one of them. `pkill -f "vite dev"`,
`rm -rf apps/dev-server-ui/node_modules/.vite`, restart — and the render matched
the model immediately.

**Detection signal**: the UI contradicts a passing unit test on the same input,
or React Flow warns `Node type "X" not found` for a type you just added.
**Prevention**: after adding a new *module* (not just editing one) that is
imported from more than one place, restart Vite before debugging the logic.
Screenshots are fresh browser instances, so they do not rule this out — the
staleness is server-side, not in the page.

### 17. A passing assertion about a model is not evidence about a view
Symptom: segment tooltips were written, populated by every generator, asserted
by a test over all 44 fixtures, and **invisible to every user**. They shipped as
a native `title=` attribute, and the row's Radix HoverCard opens on the same
pointer move — immediately, and styled — so the browser tooltip never won.

The test asserted `segment.tooltip` was truthy. That was true. The strings were
correct. The reviewer found it by hovering.

**Failure mode**: writing the test at the layer the change was easiest to test
at, rather than at the layer the change was supposed to have an effect. The
model was never the risky part; the delivery was, and the delivery had no test
and no manual check either.

**Detection signal**: a change whose whole purpose is "the user can now see X",
verified only by a unit test. Ask "what would I look at to know a user sees
this?" — if the answer is not a screenshot or a DOM read, the change is
unverified however green the suite is.

**Prevention**: for anything user-visible, read it back out of the DOM before
committing. `page.hover()` + reading the popper's `innerText` took under a
minute and would have caught it immediately. The same check later caught a
second real bug on the first try — a fan-out whose members had all collapsed to
0% because the segment positions were measured from `queuedAt` while the
envelope used execution start.
