ALTER TABLE worker_connections ADD COLUMN max_worker_concurrency INT NOT NULL DEFAULT 0;
