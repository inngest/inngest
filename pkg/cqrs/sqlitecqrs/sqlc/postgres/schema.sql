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
	url VARCHAR NOT NULL
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
	cron VARCHAR
);

CREATE TABLE function_finishes (
	run_id BYTEA,
	-- Ignoring not null because of https://github.com/sqlc-dev/sqlc/issues/2806#issuecomment-1750038624
	status VARCHAR,
	output VARCHAR DEFAULT '{}',
	completed_step_count INT DEFAULT 1,
	created_at TIMESTAMP
);

CREATE TABLE history (
	id BYTEA,
	created_at TIMESTAMP NOT NULL,
	run_started_at TIMESTAMP NOT NULL,
	function_id UUID,
	function_version INT NOT NULL,
	run_id BYTEA NOT NULL,
	event_id BYTEA NOT NULL,
	batch_id BYTEA,
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
	event_ids BYTEA NOT NULL
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
	resource_attributes BYTEA NOT NULL,
	scope_name VARCHAR NOT NULL,
	scope_version VARCHAR NOT NULL,
	span_attributes BYTEA NOT NULL,
	duration INT NOT NULL, -- duration in milli
	status_code VARCHAR NOT NULL,
	status_message TEXT,
	events BYTEA NOT NULL, -- list of events
	links BYTEA NOT NULL,  -- list of links
	run_id CHAR(26)
);

CREATE TABLE trace_runs (
	run_id CHAR(26) PRIMARY KEY,

	account_id CHAR(36) NOT NULL,
	workspace_id CHAR(36) NOT NULL,
	app_id CHAR(36) NOT NULL,
	function_id CHAR(36) NOT NULL,
	trace_id BYTEA NOT NULL,

	queued_at INT NOT NULL,
	started_at INT NOT NULL,
	ended_at INT NOT NULL,

	status INT NOT NULL, -- more like enum values
	source_id VARCHAR NOT NULL,
	trigger_ids BYTEA NOT NULL,
	output BYTEA,
	is_debounce BOOLEAN NOT NULL,
	batch_id BYTEA,
	cron_schedule TEXT
);

CREATE TABLE queue_snapshot_chunks (
    snapshot_id CHAR(26) NOT NULL,
    chunk_id INT NOT NULL,
    data BYTEA,
    PRIMARY KEY (snapshot_id, chunk_id)
);
