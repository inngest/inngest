-- Backup the table data
CREATE TABLE worker_connections_backup AS SELECT * FROM worker_connections;

-- Drop the original table
DROP TABLE worker_connections;

-- Recreate the table without max_worker_concurrency
CREATE TABLE worker_connections (
    account_id UUID NOT NULL,
    workspace_id UUID NOT NULL,

    app_name VARCHAR NOT NULL,
    app_id UUID,

    id BYTEA NOT NULL,
    gateway_id BYTEA NOT NULL,
    instance_id VARCHAR NOT NULL,
    status smallint NOT NULL,
    worker_ip VARCHAR NOT NULL,

    connected_at BIGINT NOT NULL,
    last_heartbeat_at BIGINT,
    disconnected_at BIGINT,
    recorded_at BIGINT NOT NULL,
    inserted_at BIGINT NOT NULL,

    disconnect_reason VARCHAR,

    group_hash BYTEA NOT NULL,
    sdk_lang VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    sdk_platform VARCHAR NOT NULL,
    sync_id UUID,
    app_version VARCHAR,
    function_count integer NOT NULL,

    cpu_cores integer NOT NULL,
    mem_bytes bigint NOT NULL,
    os VARCHAR NOT NULL,

    PRIMARY KEY(id, app_name)
);

-- Restore the data without max_worker_concurrency column
INSERT INTO worker_connections (
    account_id, workspace_id, app_name, app_id, id, gateway_id, instance_id, status, 
    worker_ip, connected_at, last_heartbeat_at, disconnected_at, 
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, 
    sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
)
SELECT 
    account_id, workspace_id, app_name, app_id, id, gateway_id, instance_id, status, 
    worker_ip, connected_at, last_heartbeat_at, disconnected_at, 
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, 
    sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
FROM worker_connections_backup;

-- Drop the backup table
DROP TABLE worker_connections_backup;
