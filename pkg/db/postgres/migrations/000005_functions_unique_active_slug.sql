-- +goose Up

-- With PRIMARY KEY (id) now in place (000004), the remaining shape of the
-- #3556 corruption is two rows with distinct ids sharing the same
-- (app_id, slug) while both archived_at IS NULL. Archive the older copies
-- — we cannot DELETE because function_runs / history reference function ids
-- by value and the loser row may have runs attached. Keep the most recent
-- (highest ctid) per (app_id, slug).
-- IS NOT DISTINCT FROM (rather than =) so two NULL app_ids are treated as
-- equal during dedup. SQLite's GROUP BY in the sibling migration already
-- collapses NULLs into a single group; this keeps the two dialects aligned.
-- (The partial unique index itself still permits multiple NULL-app_id rows
-- per ANSI semantics, but at least we don't ship divergent dedup logic.)
UPDATE functions
SET archived_at = NOW()
WHERE id IN (
    SELECT a.id
    FROM functions a
    JOIN functions b
      ON a.app_id IS NOT DISTINCT FROM b.app_id
     AND a.slug = b.slug
     AND a.ctid < b.ctid
    WHERE a.archived_at IS NULL
      AND b.archived_at IS NULL
);

CREATE UNIQUE INDEX functions_app_id_slug_active_key
    ON functions (app_id, slug)
    WHERE archived_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS functions_app_id_slug_active_key;
