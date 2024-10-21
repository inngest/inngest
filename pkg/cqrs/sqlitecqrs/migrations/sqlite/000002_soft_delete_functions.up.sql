-- Adds new column for soft deletes
ALTER TABLE functions
ADD COLUMN archived_at TIMESTAMP;

-- Renames existing apps.deleted_at column to archived_at
ALTER TABLE apps
RENAME COLUMN deleted_at TO archived_at;
