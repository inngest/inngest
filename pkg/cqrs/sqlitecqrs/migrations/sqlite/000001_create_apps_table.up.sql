CREATE TABLE apps (
	id CHAR(36) PRIMARY KEY,
	name VARCHAR NOT NULL,
	sdk_language VARCHAR NOT NULL,
	sdk_version VARCHAR NOT NULL,
	framework VARCHAR,
	metadata VARCHAR DEFAULT '{}' NOT NULL,
	status VARCHAR NOT NULL,
	error TEXT,
	checksum VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP,
	url VARCHAR NOT NULL
);

-- XXX: - this is very basic right now.  it does not conform to the cloud.
CREATE TABLE functions (
	-- id CHAR(36) PRIMARY KEY, -- ADD this when https://github.com/duckdb/duckdb/issues/1631 is fixed.
	id CHAR(36),
	app_id CHAR(36),
	name VARCHAR NOT NULL,
	slug VARCHAR NOT NULL,
	config VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- XXX: This does not conform to the cloud.  It only includes basic fields.
CREATE TABLE events (
	internal_id BLOB,
	-- cannot use CHAR(26) for ulids, nor primary keys for null ter
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

CREATE TABLE function_runs (
	run_id BLOB NOT NULL,
	run_started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	function_id CHAR(36),
	function_version INT NOT NULL,
	trigger_type VARCHAR NOT NULL DEFAULT 'event',
	-- or 'cron' if this is a cron-based function.
	event_id BLOB NOT NULL,
	batch_id BLOB,
	original_run_id BLOB,
	cron VARCHAR
);

CREATE TABLE function_finishes (
	run_id BLOB,
	status VARCHAR NOT NULL,
	output VARCHAR NOT NULL DEFAULT '{}',
	completed_step_count INT NOT NULL DEFAULT 1,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE history (
	id BLOB,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	run_started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	function_id CHAR(36),
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
	cron_schedule TEXT
);
