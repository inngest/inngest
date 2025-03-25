ALTER TABLE apps DROP COLUMN "method";
ALTER TABLE apps ADD COLUMN connection_type VARCHAR(32) NOT NULL DEFAULT 'serverless';
