-- Adds new column for soft deletes
ALTER TABLE functions
ADD COLUMN deleted_at TIMESTAMP;
