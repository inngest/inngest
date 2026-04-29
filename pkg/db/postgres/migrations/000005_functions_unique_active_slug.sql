-- +goose Up

-- With PRIMARY KEY (id) now in place (000004), the remaining shape of the
-- #3556 corruption is two rows with distinct ids sharing the same
-- (app_id, slug) while both archived_at IS NULL. Archive the older copies
-- — we cannot DELETE because function_runs / history reference function ids
-- by value and the loser row may have runs attached. Keep the most recent
-- (highest ctid) per (app_id, slug).
UPDATE functions
SET archived_at = NOW()
WHERE id IN (
    SELECT a.id
    FROM functions a
    JOIN functions b
      ON a.app_id = b.app_id
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
