-- +goose NO TRANSACTION
-- +goose Up

-- All CREATE INDEX CONCURRENTLY statements are idempotent (IF NOT EXISTS)
-- and placed before the non-idempotent ALTER TABLE so a partial failure
-- mid-migration is safely retryable.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS functions_pkey ON functions(id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_functions_app_id ON functions(app_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_functions_active ON functions(app_id) WHERE archived_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_functions_slug ON functions(slug) WHERE archived_at IS NULL;

-- Promote the unique index to a PK (instant; index already exists).
-- Wrapped in a DO block so a retry after partial failure is safe.
-- +goose StatementBegin
DO $$ BEGIN
  ALTER TABLE functions ADD CONSTRAINT functions_pkey PRIMARY KEY USING INDEX functions_pkey;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE functions DROP CONSTRAINT IF EXISTS functions_pkey;
ALTER TABLE functions ALTER COLUMN id DROP NOT NULL;
DROP INDEX CONCURRENTLY IF EXISTS functions_pkey;
DROP INDEX CONCURRENTLY IF EXISTS idx_functions_app_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_functions_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_functions_slug;
