ALTER TABLE apps DROP COLUMN is_connect;
ALTER TABLE apps ADD COLUMN connection_type VARCHAR DEFAULT 'serverless' NOT NULL;
