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
	deleted_at TIMESTAMP,
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
	created_at TIMESTAMP NOT NULL
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
	url VARCHAR,
	cancel_request VARCHAR,
	sleep VARCHAR,
	wait_for_event VARCHAR,
	wait_result VARCHAR,
	result VARCHAR
);
