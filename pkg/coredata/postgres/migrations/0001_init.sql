-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto" WITH SCHEMA public;

-- action_versions are individual versions of runnable actions.
-- actions are steps that run as part of a step function.
-- once published/enabled, they cannot be overwritten.
CREATE TABLE action_versions (
  action_dsn character varying(255) NOT NULL,
	version_major integer NOT NULL,
  version_minor integer NOT NULL,
	-- cue configuration
	config text NOT NULL,
  -- container image sha256
  image_sha256 character(64),
  -- valid date range marks when the action has been enabled
  valid_from timestamp without time zone,
  valid_to timestamp without time zone,
	created_at timestamp without time zone NOT NULL default now(),
	PRIMARY KEY (action_dsn, version_major, version_minor)
);

-- functions are named functions that have n number of versions
CREATE TABLE functions (
  function_id character varying(255) NOT NULL,
  name character varying(255) NOT NULL,
  PRIMARY KEY (function_id)
);

-- function_versions is a store of immutable configurations for a given function.
-- config is a serialized cue configuration instructing how the step function should run
-- and which actions/action_versions to run.
-- only one function currently can be live (valid) at a current time.
CREATE TABLE function_versions (
  function_id character varying(255) NOT NULL,
  version integer NOT NULL,
  -- cue configuration
  config text NOT NULL,
  -- valid date range marks when the version is "live"
  valid_from timestamp without time zone,
  valid_to timestamp without time zone,
  created_at timestamp without time zone DEFAULT now() NOT NULL,
  updated_at timestamp without time zone DEFAULT now() NOT NULL,
  PRIMARY KEY (function_id, version)
);

ALTER TABLE ONLY function_versions
  ADD CONSTRAINT function_versions_function_id FOREIGN KEY (function_id) REFERENCES functions(function_id) ON DELETE CASCADE;

CREATE INDEX function_versions_valid ON function_versions USING btree (valid_from, valid_to, function_id);

-- function_triggers is used to query for matching functions when an event is received
CREATE TABLE function_triggers (
  id uuid DEFAULT gen_random_uuid() NOT NULL,
  function_id character varying(255) NOT NULL,
  -- the matching function_version
  version integer NOT NULL,
  event_name character varying(255),
  schedule character varying(50),
  expression text,
  PRIMARY KEY (id)
);

ALTER TABLE ONLY function_triggers
ADD CONSTRAINT function_triggers_function_id_version FOREIGN KEY (function_id, version) REFERENCES function_versions(function_id, version) ON DELETE CASCADE;

-- indexes to match by trigger
CREATE INDEX function_triggers_event_name_function_id ON function_triggers USING btree (event_name, function_id) WHERE event_name IS NOT NULL;
CREATE INDEX function_triggers_schedule_function_id ON function_triggers USING btree (schedule, function_id) WHERE schedule IS NOT NULL;


-- +goose Down
DROP TABLE action_versions;
DROP TABLE function_triggers;
DROP TABLE function_versions;
DROP TABLE functions;