CREATE TABLE "worker_connections" (
    account_id CHAR(36) NOT NULL,
    workspace_id CHAR(36) NOT NULL,

    app_id CHAR(36),

    id CHAR(26) PRIMARY KEY,
    gateway_id CHAR(26) NOT NULL,
    instance_id VARCHAR NOT NULL,
    status INT NOT NULL,
    worker_ip VARCHAR NOT NULL,

    connected_at INT NOT NULL,
    last_heartbeat_at INT,
    disconnected_at INT,
    recorded_at INT,
    inserted_at INT,

    disconnect_reason VARCHAR,

    group_hash BLOB NOT NULL,
    sdk_lang VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    sdk_platform VARCHAR NOT NULL,
    sync_id CHAR(36),
    build_id VARCHAR,
    function_count INT NOT NULL,

    cpu_cores INT NOT NULL,
    mem_bytes INT NOT NULL,
    os VARCHAR NOT NULL
);
