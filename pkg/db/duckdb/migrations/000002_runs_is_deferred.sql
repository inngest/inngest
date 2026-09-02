-- +goose Up

-- Nullable boolean: TRUE for a run scheduled via `defer()` (i.e. its
-- OnFunctionScheduled hook saw at least one inngest/deferred.schedule
-- trigger event), NULL otherwise. Set once at schedule time and re-included
-- verbatim by every later inngest.runs row for the same run_id (Started/
-- Finished/Cancelled) -- see pkg/execution/dualwrite/listener.go's
-- runCommonFields, which derives it fresh from that hook's own evts each
-- time rather than relying on an update, matching every other column's
-- append-only, latest-row-wins semantics.
ALTER TABLE inngest.runs ADD COLUMN is_deferred BOOLEAN;

-- +goose Down

ALTER TABLE inngest.runs DROP COLUMN is_deferred;
