-- Revert: remove the PRIMARY KEY constraint from the functions table.
-- Note: this does NOT restore any duplicate rows that were removed.
ALTER TABLE functions DROP CONSTRAINT functions_pkey;
