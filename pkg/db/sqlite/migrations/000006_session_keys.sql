-- +goose Up
CREATE TABLE session_keys (
  workspace_id CHAR(36) NOT NULL,
  session_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (workspace_id, session_key)
);

CREATE INDEX session_keys_workspace_created_at
  ON session_keys (workspace_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS session_keys_workspace_created_at;
DROP TABLE IF EXISTS session_keys;
