-- +goose Up

-- Dedup any same-name rows before adding the unique index. The relaxed
-- predicate (WHERE name <> '', no archived_at check) means (active,
-- archived) and (archived, archived) pairs would also violate, so the
-- dedup walks every status combination. Tiebreaker order:
--   1. Active wins over archived (customer's most recent intent: an active
--      row is currently in use; an archived row was deliberately retired).
--   2. Among same status, the row with more active functions wins (keeps
--      the row that is actually doing work).
--   3. Newer created_at wins.
--   4. Higher id breaks any remaining ties deterministically.
--
-- Losers are force-archived and their name is suffixed with their id so
-- the row stays reachable for debugging or manual recovery but no longer
-- collides on the index.
--
-- For realistic dev-server DBs this is a no-op: same-name dupes only arise
-- from buggy past flows (e.g., the v1.13 → v1.19 upgrade path) or manual
-- SQL.
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
SET archived_at = COALESCE(archived_at, NOW()),
    name       = name || ' (id:' || id::text || ')'
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

-- One row per non-empty name, regardless of archived state. This lets
-- UpsertAppByName conflict on name and DO UPDATE archived_at = NULL,
-- reviving an archived row when an SDK re-syncs under the same name.
-- Empty names stay unconstrained so the placeholder paths (-u startup,
-- autodiscovery, UI add-by-URL) can keep writing rows with name='' for
-- any number of distinct URLs.
CREATE UNIQUE INDEX apps_name_unique_key
    ON apps (name)
    WHERE name <> '';

-- +goose Down

DROP INDEX IF EXISTS apps_name_unique_key;
