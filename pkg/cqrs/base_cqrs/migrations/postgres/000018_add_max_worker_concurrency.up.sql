ALTER TABLE worker_connections ADD COLUMN max_worker_concurrency BIGINT NOT NULL DEFAULT 0;
