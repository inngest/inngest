-- +goose Up

-- Same shape as 000005 on postgres: archive any older active duplicates that
-- share (app_id, slug) before adding the partial unique index. Keep the row
-- with the highest rowid (most recent insert) per (app_id, slug).
UPDATE functions
SET archived_at = datetime('now')
WHERE archived_at IS NULL
  AND rowid NOT IN (
      SELECT MAX(rowid)
      FROM functions
      WHERE archived_at IS NULL
      GROUP BY app_id, slug
  );

CREATE UNIQUE INDEX functions_app_id_slug_active_key
    ON functions (app_id, slug)
    WHERE archived_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS functions_app_id_slug_active_key;
