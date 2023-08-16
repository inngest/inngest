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
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP,
	url VARCHAR NOT NULL
);

-- XXX: - this is very basic right now.  it does not conform to the cloud.
CREATE TABLE functions (
	-- id UUID PRIMARY KEY, -- ADD this when https://github.com/duckdb/duckdb/issues/1631 is fixed.
	id UUID,
	app_id UUID,
	name VARCHAR NOT NULL,
	slug VARCHAR NOT NULL,
	config VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- XXX: This does not conform to the cloud.  It only includes basic fields.
CREATE TABLE events (
	internal_id CHAR(26) PRIMARY KEY,
	event_id VARCHAR NOT NULL,
	event_data VARCHAR DEFAULT '{}' NOT NULL,
	event_user VARCHAR DEFAULT '{}' NOT NULL,
	event_v VARCHAR,
	event_ts TIMESTAMP NOT NULL
);

CREATE TABLE function_runs (
	run_id CHAR(26) NOT NULL, 
	run_started_at TIMESTAMP NOT NULL DEFAULT NOW(),
	function_id UUID,
	function_version INT NOT NULL,
	event_id CHAR(26) NOT NULL, 
	batch_id CHAR(26), 
	original_run_id CHAR(26)
);

CREATE TABLE history (
	id UUID,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	run_started_at TIMESTAMP NOT NULL DEFAULT NOW(),
	function_id UUID,
	function_version INT NOT NULL,
	run_id CHAR(26) NOT NULL, 
	event_id CHAR(26) NOT NULL, 
	batch_id CHAR(26), 
	group_id CHAR(36),
	idempotency_key VARCHAR NOT NULL,
	type VARCHAR NOT NULL,
	attempt INT NOT NULL,
	step_name VARCHAR,
	step_id VARCHAR,
	url VARCHAR,
	cancel_request VARCHAR,
	sleep VARCHAR,
	wait_for_event VARCHAR,
	result VARCHAR
);
