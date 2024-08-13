CREATE TABLE queue_snapshot_versions (
	snapshot_id INT PRIMARY KEY,
	created_at TIMESTAMP NOT NULL
);

CREATE TABLE queue_snapshot_chunks (
	snapshot_id INT NOT NULL,
	chunk_id INT NOT NULL,
	data BLOB
);
