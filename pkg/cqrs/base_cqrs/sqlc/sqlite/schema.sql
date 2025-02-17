CREATE TABLE apps (
	id UUID PRIMARY KEY,
	name VARCHAR NOT NULL,
	sdk_language VARCHAR NOT NULL,
	sdk_version VARCHAR NOT NULL,
	framework VARCHAR,
	metadata VARCHAR DEFAULT '{}' NOT NULL,
	status VARCHAR NOT NULL,
	error TEXT,
	checksum VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL,
	archived_at TIMESTAMP,
	url VARCHAR NOT NULL,
    method VARCHAR NOT NULL DEFAULT 'serve',
    app_version VARCHAR
);

CREATE TABLE events (
	internal_id CHAR(26) PRIMARY KEY,
	account_id CHAR(36),
	workspace_id CHAR(36),
	source VARCHAR(255),
	source_id CHAR(35),
	received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	event_id VARCHAR NOT NULL,
	event_name VARCHAR NOT NULL,
	event_data VARCHAR DEFAULT '{}' NOT NULL,
	event_user VARCHAR DEFAULT '{}' NOT NULL,
	event_v VARCHAR,
	event_ts TIMESTAMP NOT NULL
);

CREATE TABLE functions (
	id UUID PRIMARY KEY,
	app_id UUID,
	name VARCHAR NOT NULL,
	slug VARCHAR NOT NULL,
	config VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL,
	archived_at TIMESTAMP
);

CREATE TABLE function_runs (
	run_id CHAR(26) NOT NULL,
	run_started_at TIMESTAMP NOT NULL,
	function_id UUID,
	function_version INT NOT NULL,
	trigger_type VARCHAR NOT NULL,
	event_id CHAR(26) NOT NULL,
	batch_id CHAR(26),
	original_run_id CHAR(26),
	cron VARCHAR,
	workspace_id UUID
);

CREATE TABLE function_finishes (
	run_id BLOB,
	-- Ignoring not null because of https://github.com/sqlc-dev/sqlc/issues/2806#issuecomment-1750038624
	status VARCHAR,
	output VARCHAR DEFAULT '{}',
	completed_step_count INT DEFAULT 1,
	created_at TIMESTAMP
);

CREATE TABLE history (
	id BLOB,
	created_at TIMESTAMP NOT NULL,
	run_started_at TIMESTAMP NOT NULL,
	function_id UUID,
	function_version INT NOT NULL,
	run_id BLOB NOT NULL,
	event_id BLOB NOT NULL,
	batch_id BLOB,
	group_id VARCHAR,
	idempotency_key VARCHAR NOT NULL,
	type VARCHAR NOT NULL,
	attempt INT NOT NULL,
	latency_ms INT,
	step_name VARCHAR,
	step_id VARCHAR,
	step_type VARCHAR,
	url VARCHAR,
	cancel_request VARCHAR,
	sleep VARCHAR,
	wait_for_event VARCHAR,
	wait_result VARCHAR,
	invoke_function VARCHAR,
	invoke_function_result VARCHAR,
	result VARCHAR
);

CREATE TABLE event_batches (
	id CHAR(26) PRIMARY KEY,
	account_id UUID,
	workspace_id UUID,
	app_id UUID,
	workflow_id UUID,
	run_id CHAR(26) NOT NULL,
	started_at TIMESTAMP NOT NULL,
	executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	event_ids BLOB NOT NULL
);

CREATE TABLE traces (
	timestamp TIMESTAMP NOT NULL,
	timestamp_unix_ms INT NOT NULL,
	trace_id VARCHAR NOT NULL,
	span_id VARCHAR NOT NULL,
	parent_span_id VARCHAR,
	trace_state VARCHAR,
	span_name VARCHAR NOT NULL,
	span_kind VARCHAR NOT NULL,
	service_name VARCHAR NOT NULL,
	resource_attributes BLOB NOT NULL,
	scope_name VARCHAR NOT NULL,
	scope_version VARCHAR NOT NULL,
	span_attributes BLOB NOT NULL,
	duration INT NOT NULL, -- duration in milli
	status_code VARCHAR NOT NULL,
	status_message TEXT,
	events BLOB NOT NULL, -- list of events
	links BLOB NOT NULL,  -- list of links
	run_id CHAR(26)
);

CREATE TABLE trace_runs (
	run_id CHAR(26) PRIMARY KEY,

	account_id CHAR(36) NOT NULL,
	workspace_id CHAR(36) NOT NULL,
	app_id CHAR(36) NOT NULL,
	function_id CHAR(36) NOT NULL,
	trace_id BLOB NOT NULL,

	queued_at INT NOT NULL,
	started_at INT NOT NULL,
	ended_at INT NOT NULL,

	status INT NOT NULL, -- more like enum values
	source_id VARCHAR NOT NULL,
	trigger_ids BLOB NOT NULL,
	output BLOB,
	is_debounce BOOLEAN NOT NULL,
	batch_id BLOB,
	cron_schedule TEXT,
	has_ai BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE queue_snapshot_chunks (
    snapshot_id CHAR(26) NOT NULL,
    chunk_id INT NOT NULL,
    data BLOB,
    PRIMARY KEY (snapshot_id, chunk_id)
);

CREATE TABLE worker_connections (
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
    os VARCHAR NOT NULL
);
