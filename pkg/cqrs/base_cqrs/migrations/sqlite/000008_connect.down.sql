ALTER TABLE apps DROP COLUMN connection_type;
ALTER TABLE apps ADD COLUMN is_connect BOOLEAN;
