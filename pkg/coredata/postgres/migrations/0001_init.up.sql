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

-- function_triggers is used to query for matching functions when an event is received
CREATE TABLE function_triggers (
  function_id character varying(255) NOT NULL,
  event_name character varying(255),
  schedule character varying(50),
  -- only 1 schedule trigger may exist for a given function. those will have blank event_names
  PRIMARY KEY (function_id)
);

ALTER TABLE ONLY function_triggers
  ADD CONSTRAINT function_triggers_function_id FOREIGN KEY (function_id) REFERENCES functions(function_id) ON DELETE CASCADE;

CREATE INDEX function_triggers_event_name ON function_triggers (event_name);
CREATE INDEX function_triggers_schedule ON function_triggers (schedule) WHERE schedule IS NOT NULL;


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