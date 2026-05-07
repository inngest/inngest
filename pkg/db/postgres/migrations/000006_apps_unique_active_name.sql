-- +goose Up

-- Defensive dedup: archive older copies if two active rows share a non-empty
-- name. Soft-delete only — function_runs / history reference app_id by value
-- and the loser row may have runs attached. Keep the highest ctid (most
-- recent insert) per name. Mirrors the precedent in 000005 for functions.
UPDATE apps
SET archived_at = NOW()
WHERE id IN (
    SELECT a.id
    FROM apps a
    JOIN apps b
      ON a.name = b.name
     AND a.ctid < b.ctid
    WHERE a.archived_at IS NULL
      AND b.archived_at IS NULL
      AND a.name <> ''
);

-- One active row per non-empty name. Empty names are excluded so the
-- placeholder paths (-u startup, autodiscovery, UI add-by-URL) can keep
-- writing rows with name='' for any number of distinct URLs.
CREATE UNIQUE INDEX apps_name_active_key
    ON apps (name)
    WHERE archived_at IS NULL AND name <> '';

-- +goose Down

DROP INDEX IF EXISTS apps_name_active_key;
