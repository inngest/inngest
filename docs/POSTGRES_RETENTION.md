# Postgres Retention & Truncation (Self-Hosted)

Self-hosted Inngest persists run history to Postgres and **never truncates it
automatically**. These tables grow without bound:

| Table | Grows with | Anchor for retention |
| --- | --- | --- |
| `trace_runs` | one row per run (upserted) | **self** (status + `ended_at`) |
| `traces` | OTEL spans per run | `trace_runs` |
| `spans` | tracing spans per run | `trace_runs` |
| `event_batches` | batched-run executions | `trace_runs` |
| `function_finishes` | one row per **finished** run | **self** (`created_at`) |
| `function_runs` | one row per run | `function_finishes` |
| `history` | many rows per run | `function_finishes` |
| `events` | one row per received event | **self** (`received_at` + guard) |

This document gives a safe, ready-to-run truncation strategy. The default
retention window is **30 days**.

> There is no retention config flag today (`cmd/start/cmd.go`) and no background
> GC. Run this manually or via cron until in-server GC ships (see
> [Future work](#future-work)).

## Why age alone is unsafe

A run can legitimately stay open for a long time without being stuck:

- **debounce** up to 7 days (`MaxDebouncePeriod`, `pkg/consts/consts.go`)
- **`waitForEvent`** — effectively unbounded
- **retries** up to 24h (`MaxRetryDuration`)

Deleting rows purely by age would destroy in-flight runs. **Deletion must be
gated on a terminal run status**, with age only deciding how much *finished*
history to keep.

## The model: status-gated, encoding-aligned cascade

A run's data is deletable only when **(a)** the run reached a terminal status
**and** **(b)** it finished more than the retention window ago.

Two authoritative "ended" signals exist, and each child table happens to share
the `run_id` encoding of one of them — so **no ULID decoding is ever needed**:

### Anchor 1 — `trace_runs` (`run_id` is `CHAR(26)` string ULID)

`status` is stored as the enum **code** (`run.Status.ToCode()`,
`pkg/cqrs/manager/cqrs.go`):

| Status | Code | Terminal? |
| --- | --- | --- |
| Scheduled | 100 | no |
| Running | 200 | no |
| Overflowed | 50 | yes |
| Completed | 300 | yes |
| Failed | 400 | yes |
| Cancelled | 500 | yes |
| Skipped | 600 | yes |

Terminal set = `(50, 300, 400, 500, 600)`. `ended_at` is a unix-millisecond
timestamp that is only meaningful once a run has ended, so it is **always
filtered after** the status gate, never on its own. This anchor gates `traces`,
`spans`, and `event_batches` (all carry `run_id` as text/`CHAR(26)`).

### Anchor 2 — `function_finishes` (`run_id` is `BYTEA` binary ULID)

A row exists **only when a run finishes** (`InsertFunctionFinish`); `created_at`
is the finish time. Its `run_id` encoding matches `function_runs.run_id` and
`history.run_id` exactly. This anchor gates `function_runs` and `history`. A
`function_runs` row with **no** `function_finishes` row is still executing and
is therefore **never deleted**.

### `events` are not 1:1 with runs

One event can fan out to many runs, and events can exist with zero runs. They
are pruned by their own `received_at` age, with a guard that keeps any event
still referenced by an **unfinished** `function_run`.

## Required indexes

The current schema (`pkg/db/postgres/migrations/000001_baseline.sql`) does not
index the gating predicates. Add these once, using `CONCURRENTLY` to avoid
locking writes (matching migrations `000002`/`000003`):

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_status_ended_at
  ON trace_runs (status, ended_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_function_finishes_created_at
  ON function_finishes (created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_traces_run_id
  ON traces (run_id);
```

`spans (run_id)` and `history (run_id, created_at)` already exist.
`CREATE INDEX CONCURRENTLY` cannot run inside a transaction block — run each on
its own.

## The truncation script

Run in this order. Children are deleted **before** their anchor — once anchor
rows are gone, the gating subquery returns nothing. Each statement is batched
(`LIMIT`) and should be looped until it reports `0` rows, which keeps locks
short and avoids one giant transaction. Tune `:batch` (5,000–10,000 is a good
start) and `:days` (default `30`).

### Pipeline A — trace data (gated by `trace_runs`)

```sql
-- 1. children of ended, expired trace runs
DELETE FROM spans
WHERE ctid IN (
  SELECT s.ctid FROM spans s
  WHERE s.run_id IN (
    SELECT run_id FROM trace_runs
    WHERE status IN (50,300,400,500,600)
      AND ended_at < (extract(epoch FROM now()) * 1000 - :days * 86400000)::bigint
  )
  LIMIT :batch
);

DELETE FROM traces
WHERE ctid IN (
  SELECT t.ctid FROM traces t
  WHERE t.run_id IN (
    SELECT run_id FROM trace_runs
    WHERE status IN (50,300,400,500,600)
      AND ended_at < (extract(epoch FROM now()) * 1000 - :days * 86400000)::bigint
  )
  LIMIT :batch
);

DELETE FROM event_batches
WHERE ctid IN (
  SELECT eb.ctid FROM event_batches eb
  WHERE eb.run_id IN (
    SELECT run_id FROM trace_runs
    WHERE status IN (50,300,400,500,600)
      AND ended_at < (extract(epoch FROM now()) * 1000 - :days * 86400000)::bigint
  )
  LIMIT :batch
);

-- 2. the anchor itself (run LAST in this pipeline)
DELETE FROM trace_runs
WHERE ctid IN (
  SELECT ctid FROM trace_runs
  WHERE status IN (50,300,400,500,600)
    AND ended_at < (extract(epoch FROM now()) * 1000 - :days * 86400000)::bigint
  LIMIT :batch
);
```

### Pipeline B — function/history data (gated by `function_finishes`)

`run_id` is `BYTEA` in all three tables, so the join needs no conversion.

```sql
-- 1. history + function_runs for finished, expired runs
DELETE FROM history
WHERE ctid IN (
  SELECT h.ctid FROM history h
  WHERE h.run_id IN (
    SELECT run_id FROM function_finishes
    WHERE created_at < now() - make_interval(days => :days)
  )
  LIMIT :batch
);

DELETE FROM function_runs
WHERE ctid IN (
  SELECT fr.ctid FROM function_runs fr
  WHERE fr.run_id IN (
    SELECT run_id FROM function_finishes
    WHERE created_at < now() - make_interval(days => :days)
  )
  LIMIT :batch
);

-- 2. the anchor itself (run LAST in this pipeline)
DELETE FROM function_finishes
WHERE ctid IN (
  SELECT ctid FROM function_finishes
  WHERE created_at < now() - make_interval(days => :days)
  LIMIT :batch
);
```

### Pipeline C — events (own age + unfinished-run guard)

```sql
DELETE FROM events
WHERE ctid IN (
  SELECT e.ctid FROM events e
  WHERE e.received_at < now() - make_interval(days => :days)
    AND NOT EXISTS (
      SELECT 1 FROM function_runs fr
      LEFT JOIN function_finishes ff ON ff.run_id = fr.run_id
      WHERE fr.event_id = e.internal_id
        AND ff.run_id IS NULL          -- a run that has NOT finished
    )
  LIMIT :batch
);
```

> The `ctid IN (... LIMIT :batch)` form is used everywhere so a single `DELETE`
> caps its row count (plain `DELETE ... WHERE` has no `LIMIT`). Re-run each
> statement until it affects `0` rows.

### Example batch-loop wrapper (psql)

```sql
-- repeat until 0 rows; psql reports "DELETE n" after each call
\set days 30
\set batch 10000
DELETE FROM trace_runs
WHERE ctid IN (
  SELECT ctid FROM trace_runs
  WHERE status IN (50,300,400,500,600)
    AND ended_at < (extract(epoch FROM now()) * 1000 - :days * 86400000)::bigint
  LIMIT :batch
);
```

Wrap the full ordered sequence in a shell/`pg_cron` job that repeats each
statement while `rows affected > 0`.

## Dry run / verification

Before deleting, confirm the predicates select what you expect — and, crucially,
that nothing in-flight is selected:

```sql
-- How much WOULD be removed (Pipeline A / B / C anchors):
SELECT count(*) FROM trace_runs
  WHERE status IN (50,300,400,500,600)
    AND ended_at < (extract(epoch FROM now()) * 1000 - 30*86400000)::bigint;
SELECT count(*) FROM function_finishes WHERE created_at < now() - INTERVAL '30 days';

-- Safety check: these MUST stay (running/scheduled, or unfinished) and must
-- NOT intersect the delete sets above.
SELECT count(*) FROM trace_runs WHERE status IN (100,200);          -- scheduled/running
SELECT count(*) FROM function_runs fr
  LEFT JOIN function_finishes ff ON ff.run_id = fr.run_id
  WHERE ff.run_id IS NULL;                                          -- never finished
```

Use `EXPLAIN` on each gating query after adding the indexes to confirm index
usage (no sequential scans on `trace_runs` / `function_finishes`).

## Operational guidance

- **Scheduling:** run nightly off-peak via cron or `pg_cron`; loop each
  statement in small batches with a short sleep between iterations.
- **Bloat:** large `DELETE`s leave dead tuples. Let autovacuum reclaim them, or
  schedule `VACUUM (ANALYZE)` after a purge. For heavy one-time reclamation
  consider `pg_repack` (rewrites tables without a long exclusive lock).
- **Window tuning:** the data is safe at any window because deletion is
  status-gated, but a very short window hides recently-finished runs from the
  UI/API. Keep the window comfortably above your longest debounce/wait usage if
  you rely on run history for debugging. 30 days is the default.
- **SQLite:** the same status-gated model applies, but `make_interval` /
  `extract(epoch ...)` are Postgres-specific. A SQLite variant is out of scope
  for this runbook.

## Future work

An in-server background GC (a ticker goroutine like the devserver poll loops),
driven by a `--retention` flag, could run these exact predicates through the
`db.Adapter` / `cqrs.Manager` layer (`pkg/cqrs/manager/cqrs.go`,
`pkg/db/postgres/querier.go`), following the `DeleteOldQueueSnapshots`
precedent (`pkg/db/postgres/sqlc/queries.sql`).
