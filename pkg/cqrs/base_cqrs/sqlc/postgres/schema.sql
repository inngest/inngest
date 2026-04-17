--
-- PostgreSQL database dump
--

-- Dumped from database version 16.13
-- Dumped by pg_dump version 16.13

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: apps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.apps (
    id character(36) NOT NULL,
    name character varying NOT NULL,
    sdk_language character varying NOT NULL,
    sdk_version character varying NOT NULL,
    framework character varying,
    metadata character varying DEFAULT '{}'::character varying NOT NULL,
    status character varying NOT NULL,
    error text,
    checksum character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    archived_at timestamp without time zone,
    url character varying NOT NULL,
    method character varying(32) DEFAULT 'serve'::character varying NOT NULL,
    app_version character varying(128)
);

--
-- Name: event_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_batches (
    id character(26) NOT NULL,
    account_id uuid,
    workspace_id uuid,
    app_id uuid,
    workflow_id uuid,
    run_id character(26) NOT NULL,
    started_at timestamp without time zone NOT NULL,
    executed_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    event_ids bytea NOT NULL
);

--
-- Name: events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.events (
    internal_id bytea,
    account_id character(36),
    workspace_id character(36),
    source character varying(255),
    source_id character(35),
    received_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    event_id character varying NOT NULL,
    event_name character varying NOT NULL,
    event_data character varying DEFAULT '{}'::character varying NOT NULL,
    event_user character varying DEFAULT '{}'::character varying NOT NULL,
    event_v character varying,
    event_ts timestamp without time zone NOT NULL
);

--
-- Name: function_finishes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.function_finishes (
    run_id bytea,
    status character varying NOT NULL,
    output character varying DEFAULT '{}'::character varying NOT NULL,
    completed_step_count integer DEFAULT 1 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

--
-- Name: function_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.function_runs (
    run_id bytea NOT NULL,
    run_started_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    function_id character(36),
    function_version integer NOT NULL,
    trigger_type character varying DEFAULT 'event'::character varying NOT NULL,
    event_id bytea NOT NULL,
    batch_id bytea,
    original_run_id bytea,
    cron character varying
);

--
-- Name: functions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.functions (
    id character(36),
    app_id character(36),
    name character varying NOT NULL,
    slug character varying NOT NULL,
    config character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    archived_at timestamp without time zone
);

--
-- Name: history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.history (
    id bytea,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    run_started_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    function_id character(36),
    function_version integer NOT NULL,
    run_id bytea NOT NULL,
    event_id bytea NOT NULL,
    batch_id bytea,
    group_id character varying,
    idempotency_key character varying NOT NULL,
    type character varying NOT NULL,
    attempt integer NOT NULL,
    latency_ms integer,
    step_name character varying,
    step_id character varying,
    url character varying,
    cancel_request character varying,
    sleep character varying,
    wait_for_event character varying,
    wait_result character varying,
    invoke_function character varying,
    invoke_function_result character varying,
    result character varying,
    step_type character varying
);

--
-- Name: migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);

--
-- Name: queue_snapshot_chunks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.queue_snapshot_chunks (
    snapshot_id character(26) NOT NULL,
    chunk_id integer NOT NULL,
    data bytea
);

--
-- Name: spans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.spans (
    span_id text NOT NULL,
    trace_id text NOT NULL,
    parent_span_id text,
    name text NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    attributes jsonb,
    links jsonb,
    dynamic_span_id text,
    account_id text NOT NULL,
    app_id text NOT NULL,
    function_id text NOT NULL,
    run_id text NOT NULL,
    env_id text NOT NULL,
    output jsonb,
    debug_run_id text,
    debug_session_id text,
    status text,
    input jsonb,
    event_ids jsonb
);

--
-- Name: trace_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trace_runs (
    run_id character(26) NOT NULL,
    account_id character(36) NOT NULL,
    workspace_id character(36) NOT NULL,
    app_id character(36) NOT NULL,
    function_id character(36) NOT NULL,
    trace_id bytea NOT NULL,
    queued_at bigint NOT NULL,
    started_at bigint NOT NULL,
    ended_at bigint NOT NULL,
    status integer NOT NULL,
    source_id character varying NOT NULL,
    trigger_ids bytea NOT NULL,
    output bytea,
    is_debounce boolean NOT NULL,
    batch_id bytea,
    cron_schedule text,
    has_ai boolean DEFAULT false NOT NULL
);

--
-- Name: traces; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.traces (
    "timestamp" timestamp without time zone NOT NULL,
    timestamp_unix_ms bigint NOT NULL,
    trace_id character varying NOT NULL,
    span_id character varying NOT NULL,
    parent_span_id character varying,
    trace_state character varying,
    span_name character varying NOT NULL,
    span_kind character varying NOT NULL,
    service_name character varying NOT NULL,
    resource_attributes bytea NOT NULL,
    scope_name character varying NOT NULL,
    scope_version character varying NOT NULL,
    span_attributes bytea NOT NULL,
    duration integer NOT NULL,
    status_code character varying NOT NULL,
    status_message text,
    events bytea NOT NULL,
    links bytea NOT NULL,
    run_id character(26)
);

--
-- Name: worker_connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worker_connections (
    account_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    app_name character varying NOT NULL,
    app_id uuid,
    id bytea NOT NULL,
    gateway_id bytea NOT NULL,
    instance_id character varying NOT NULL,
    status smallint NOT NULL,
    worker_ip character varying NOT NULL,
    max_worker_concurrency bigint DEFAULT 0 NOT NULL,
    connected_at bigint NOT NULL,
    last_heartbeat_at bigint,
    disconnected_at bigint,
    recorded_at bigint NOT NULL,
    inserted_at bigint NOT NULL,
    disconnect_reason character varying,
    group_hash bytea NOT NULL,
    sdk_lang character varying NOT NULL,
    sdk_version character varying NOT NULL,
    sdk_platform character varying NOT NULL,
    sync_id uuid,
    app_version character varying,
    function_count integer NOT NULL,
    cpu_cores integer NOT NULL,
    mem_bytes bigint NOT NULL,
    os character varying NOT NULL
);

--
-- Name: apps apps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (id);

--
-- Name: event_batches event_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_batches
    ADD CONSTRAINT event_batches_pkey PRIMARY KEY (id);

--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (version);

--
-- Name: queue_snapshot_chunks queue_snapshot_chunks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.queue_snapshot_chunks
    ADD CONSTRAINT queue_snapshot_chunks_pkey PRIMARY KEY (snapshot_id, chunk_id);

--
-- Name: spans spans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spans
    ADD CONSTRAINT spans_pkey PRIMARY KEY (trace_id, span_id);

--
-- Name: trace_runs trace_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trace_runs
    ADD CONSTRAINT trace_runs_pkey PRIMARY KEY (run_id);

--
-- Name: worker_connections worker_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worker_connections
    ADD CONSTRAINT worker_connections_pkey PRIMARY KEY (id, app_name);

--
-- Name: idx_events_internal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_internal_id ON public.events USING btree (internal_id);

--
-- Name: idx_events_internal_id_received_range; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_internal_id_received_range ON public.events USING btree (internal_id, received_at DESC);

--
-- Name: idx_events_received_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_received_name ON public.events USING btree (received_at, event_name);

--
-- Name: idx_function_finishes_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_function_finishes_run_id ON public.function_finishes USING btree (run_id);

--
-- Name: idx_function_runs_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_function_runs_event_id ON public.function_runs USING btree (event_id);

--
-- Name: idx_function_runs_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_function_runs_run_id ON public.function_runs USING btree (run_id);

--
-- Name: idx_function_runs_timebound; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_function_runs_timebound ON public.function_runs USING btree (run_started_at DESC, function_id);

--
-- Name: idx_history_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_history_id ON public.history USING btree (id);

--
-- Name: idx_history_run_id_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_history_run_id_created ON public.history USING btree (run_id, created_at);

--
-- Name: idx_spans_account_status_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spans_account_status_time ON public.spans USING btree (account_id, status, start_time);

--
-- Name: idx_spans_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spans_run_id ON public.spans USING btree (run_id);

--
-- Name: idx_spans_run_id_dynamic_start_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spans_run_id_dynamic_start_time ON public.spans USING btree (run_id, dynamic_span_id, start_time);

--
-- Name: idx_spans_run_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spans_run_status ON public.spans USING btree (run_id, status);

--
-- Name: idx_spans_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spans_status ON public.spans USING btree (status);

--
-- Name: idx_traces_trace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_traces_trace_id ON public.traces USING btree (trace_id);

--
-- PostgreSQL database dump complete
--
