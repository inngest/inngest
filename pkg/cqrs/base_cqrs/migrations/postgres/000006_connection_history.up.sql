CREATE TABLE worker_connections (
    account_id UUID NOT NULL,
    workspace_id UUID NOT NULL,

    app_id UUID,

    id CHAR(26) PRIMARY KEY,
    gateway_id CHAR(26) NOT NULL,
    instance_id VARCHAR NOT NULL,
    status smallint NOT NULL,
    worker_ip VARCHAR NOT NULL,

    connected_at TIMESTAMP NOT NULL,
    last_heartbeat_at TIMESTAMP,
    disconnected_at TIMESTAMP,
    recorded_at TIMESTAMP NOT NULL,
    inserted_at TIMESTAMP NOT NULL,

    disconnect_reason VARCHAR,

    group_hash BYTEA NOT NULL,
    sdk_lang VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    sdk_platform VARCHAR NOT NULL,
    sync_id UUID,
    build_id VARCHAR,
    function_count integer NOT NULL,

    cpu_cores integer NOT NULL,
    mem_bytes bigint NOT NULL,
    os VARCHAR NOT NULL
);
