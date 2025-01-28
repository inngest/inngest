ALTER TABLE functions
DROP COLUMN archived_at;

ALTER TABLE RENAME COLUMN archived_at TO deleted_at;
