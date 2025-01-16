-- Adds new table for storing snapshot chunks
CREATE TABLE queue_snapshot_chunks (
    snapshot_id CHAR(26) NOT NULL,
    chunk_id INT NOT NULL,
    data BLOB,
    PRIMARY KEY (snapshot_id, chunk_id)
);

-- Adds new required column for SQLite history storage
ALTER TABLE history
ADD COLUMN step_type VARCHAR;
