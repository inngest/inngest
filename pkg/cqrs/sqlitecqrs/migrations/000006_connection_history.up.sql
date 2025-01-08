CREATE TABLE "worker_connections" (
    account_id CHAR(36) NOT NULL,
    workspace_id CHAR(36) NOT NULL,

    app_id CHAR(36),

    id CHAR(26) PRIMARY KEY,
    gateway_id CHAR(26) NOT NULL,
    instance_id VARCHAR,
    status INT NOT NULL,

    connected_at INT NOT NULL,
    last_heartbeat_at INT NOT NULL,
    disconnected_at INT NOT NULL,

    group_hash BLOB NOT NULL,
    sdk_lang VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    sdk_platform VARCHAR NOT NULL,
    sync_id CHAR(36),

    cpu_cores INT NOT NULL,
    mem_bytes INT NOT NULL,
    os VARCHAR NOT NULL
);
