CREATE TABLE queue_snapshot_chunks (
    snapshot_id CHAR(26) NOT NULL,
    chunk_id INT NOT NULL,
    data BLOB,
    PRIMARY KEY (snapshot_id, chunk_id)
);
