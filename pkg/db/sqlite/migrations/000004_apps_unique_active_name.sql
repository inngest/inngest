-- +goose Up

-- Same shape as 000006 on postgres: archive any older active duplicates that
-- share a non-empty name before adding the partial unique index. Keep the row
-- with the highest rowid (most recent insert) per name.
UPDATE apps
SET archived_at = datetime('now')
WHERE archived_at IS NULL
  AND name <> ''
  AND rowid NOT IN (
      SELECT MAX(rowid)
      FROM apps
      WHERE archived_at IS NULL
        AND name <> ''
      GROUP BY name
  );

CREATE UNIQUE INDEX apps_name_active_key
    ON apps (name)
    WHERE archived_at IS NULL AND name <> '';

-- +goose Down

DROP INDEX IF EXISTS apps_name_active_key;
