-- Backup the table data
CREATE TABLE worker_connections_backup AS SELECT * FROM worker_connections;

-- Drop the original table
DROP TABLE worker_connections;

-- Recreate the table with max_worker_concurrency after worker_ip
CREATE TABLE worker_connections (
    account_id CHAR(36) NOT NULL,
    workspace_id CHAR(36) NOT NULL,

    app_name VARCHAR NOT NULL,
    app_id CHAR(36),

    id CHAR(26) NOT NULL,
    gateway_id CHAR(26) NOT NULL,
    instance_id VARCHAR NOT NULL,
    status INT NOT NULL,
    worker_ip VARCHAR NOT NULL,
    max_worker_concurrency INT NOT NULL DEFAULT 0,

    connected_at INT NOT NULL,
    last_heartbeat_at INT,
    disconnected_at INT,
    recorded_at INT NOT NULL,
    inserted_at INT NOT NULL,

    disconnect_reason VARCHAR,

    group_hash BLOB NOT NULL,
    sdk_lang VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    sdk_platform VARCHAR NOT NULL,
    sync_id CHAR(36),
    app_version VARCHAR,
    function_count INT NOT NULL,

    cpu_cores INT NOT NULL,
    mem_bytes INT NOT NULL,
    os VARCHAR NOT NULL,

    PRIMARY KEY(id, app_name)
);

-- Restore the data with default value for max_worker_concurrency
INSERT INTO worker_connections (
    account_id, workspace_id, app_name, app_id, id, gateway_id, instance_id, status, 
    worker_ip, max_worker_concurrency, connected_at, last_heartbeat_at, disconnected_at, 
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, 
    sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
)
SELECT 
    account_id, workspace_id, app_name, app_id, id, gateway_id, instance_id, status, 
    worker_ip, 0, connected_at, last_heartbeat_at, disconnected_at, 
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, 
    sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
FROM worker_connections_backup;

-- Drop the backup table
DROP TABLE worker_connections_backup;
