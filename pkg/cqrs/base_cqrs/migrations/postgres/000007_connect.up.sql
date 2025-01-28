ALTER TABLE apps DROP COLUMN is_connect;
ALTER TABLE apps ADD COLUMN connection_type VARCHAR(32) NOT NULL DEFAULT 'serverless';
