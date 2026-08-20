-- +goose Up
CREATE TABLE IF NOT EXISTS runs_staging (
	run_id VARCHAR NOT NULL,
	function_id VARCHAR,
	account_id VARCHAR,
	workspace_id VARCHAR,
	year INTEGER NOT NULL,
	month INTEGER NOT NULL,
	event_type VARCHAR NOT NULL,
	status VARCHAR,
	created_at TIMESTAMP NOT NULL,
	metadata VARCHAR
);

CREATE TABLE IF NOT EXISTS run_spans_staging (
	run_id VARCHAR NOT NULL,
	step_name VARCHAR,
	account_id VARCHAR,
	workspace_id VARCHAR,
	year INTEGER NOT NULL,
	month INTEGER NOT NULL,
	event_type VARCHAR NOT NULL,
	error VARCHAR,
	created_at TIMESTAMP NOT NULL,
	metadata VARCHAR
);

CREATE TABLE IF NOT EXISTS events_staging (
	event_id VARCHAR NOT NULL,
	event_name VARCHAR NOT NULL,
	account_id VARCHAR,
	workspace_id VARCHAR,
	year INTEGER NOT NULL,
	month INTEGER NOT NULL,
	occurred_at TIMESTAMP NOT NULL,
	raw_data VARCHAR
);

-- +goose Down
DROP TABLE IF EXISTS runs_staging;
DROP TABLE IF EXISTS run_spans_staging;
DROP TABLE IF EXISTS events_staging;
