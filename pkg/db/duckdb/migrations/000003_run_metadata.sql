-- +goose Up

-- inngest.run_metadata holds one row per metadata emission (executor
-- generator/experiment metadata, the AddRunMetadata API, and userland span
-- metadata extraction -- see pkg/execution/sync_lifecycle.go's
-- OnMetadataEntry and its callers). It is a standalone table, joined onto
-- inngest.run_trace_spans by (run_id, span_id) rather than a set of columns
-- on that table, mirroring the reference metadata insights table's own
-- separation of concerns.
--
-- Deliberately simpler than that reference design: this table drops the
-- concept of a metadata "op" (merge/set/delete/add) entirely. Each row is
-- the caller's full, already-resolved value set for one metadata emission,
-- not a delta to be folded with prior emissions by key. A reader wanting
-- "the current metadata for this span" collapses to the latest row per
-- (run_id, span_id, kind) -- the same latest-row-wins pattern
-- pkg/cqrs/duckdbquery already uses for inngest.runs -- rather than
-- replaying an op history.
CREATE TABLE IF NOT EXISTS inngest.run_metadata (
  account_id     UUID NOT NULL,
  env_id         UUID NOT NULL,
  run_id         VARCHAR NOT NULL,
  run_queued_at  TIMESTAMP_MS NOT NULL,
  app_id         UUID NOT NULL,
  function_id    UUID NOT NULL,
  -- span_id is the run/step/request span this metadata annotates (the
  -- "parent" every metadata span is created under) -- the join key onto
  -- inngest.run_trace_spans.span_id, not this row's own identity.
  span_id        VARCHAR NOT NULL,
  scope          VARCHAR NOT NULL,
  -- step_id/step_index/step_attempt identify the step this metadata belongs
  -- to, independently of span_id (a request-scoped metadata span's span_id
  -- is the request's execution span, not the step span, but it still
  -- belongs to a step). All three are NULL for run-scoped metadata.
  -- step_id is the same hashed step ID used to compute a step's own
  -- deterministic span identity, not the SDK-facing userland step ID.
  step_id        VARCHAR,
  step_index     INTEGER,
  step_attempt   INTEGER,
  kind           VARCHAR NOT NULL,
  is_user        BOOLEAN NOT NULL,
  values         JSON NOT NULL,
  created_at     TIMESTAMP_MS NOT NULL
);
ALTER TABLE inngest.run_metadata
  SET SORTED BY (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id, scope, step_id, step_index, step_attempt, span_id, kind);
ALTER TABLE inngest.run_metadata
  SET PARTITIONED BY (year(run_queued_at), month(run_queued_at), account_id);

-- +goose Down
DROP TABLE IF EXISTS inngest.run_metadata;
