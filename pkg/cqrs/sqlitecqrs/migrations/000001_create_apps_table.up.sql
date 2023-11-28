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
	original_run_id BLOB
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
