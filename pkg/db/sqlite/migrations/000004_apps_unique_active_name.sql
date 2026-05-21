-- +goose Up

-- Same shape as 000006 on postgres. Dedup any same-name rows before
-- adding the unique index. The relaxed predicate (WHERE name <> '', no
-- archived_at check) means (active, archived) and (archived, archived)
-- pairs would also violate, so the dedup walks every status combination.
-- Tiebreaker order:
--   1. Active wins over archived (customer's most recent intent).
--   2. Among same status, more active functions wins.
--   3. Newer created_at wins.
--   4. Higher id breaks any remaining ties deterministically.
--
-- Losers are force-archived and their name is suffixed with their id so
-- the row stays reachable for debugging or manual recovery but no longer
-- collides on the index.
WITH fn_counts AS (
    SELECT app_id, COUNT(*) AS n
    FROM functions
    WHERE archived_at IS NULL
    GROUP BY app_id
),
ranked AS (
    SELECT a.id,
           ROW_NUMBER() OVER (
               PARTITION BY a.name
               ORDER BY (a.archived_at IS NULL) DESC,
                        COALESCE(fc.n, 0) DESC,
                        a.created_at DESC,
                        a.id DESC
           ) AS rn
    FROM apps a
    LEFT JOIN fn_counts fc ON fc.app_id = a.id
    WHERE a.name <> ''
)
UPDATE apps
SET archived_at = COALESCE(archived_at, datetime('now')),
    name       = name || ' (id:' || id || ')'
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

CREATE UNIQUE INDEX apps_name_unique_key
    ON apps (name)
    WHERE name <> '';

-- +goose Down

DROP INDEX IF EXISTS apps_name_unique_key;
