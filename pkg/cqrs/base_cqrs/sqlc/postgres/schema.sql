CREATE TABLE apps (
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

CREATE TABLE event_batches (
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

CREATE TABLE events (
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

CREATE TABLE function_finishes (
    run_id bytea,
    status character varying NOT NULL,
    output character varying DEFAULT '{}'::character varying NOT NULL,
    completed_step_count integer DEFAULT 1 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE function_runs (
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

CREATE TABLE functions (
    id character(36),
    app_id character(36),
    name character varying NOT NULL,
    slug character varying NOT NULL,
    config character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    archived_at timestamp without time zone
);

CREATE TABLE history (
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

CREATE TABLE queue_snapshot_chunks (
    snapshot_id character(26) NOT NULL,
    chunk_id integer NOT NULL,
    data bytea
);

CREATE TABLE spans (
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

CREATE TABLE trace_runs (
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

CREATE TABLE traces (
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

CREATE TABLE worker_connections (
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

ALTER TABLE ONLY apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (id);

ALTER TABLE ONLY event_batches
    ADD CONSTRAINT event_batches_pkey PRIMARY KEY (id);

ALTER TABLE ONLY queue_snapshot_chunks
    ADD CONSTRAINT queue_snapshot_chunks_pkey PRIMARY KEY (snapshot_id, chunk_id);

ALTER TABLE ONLY spans
    ADD CONSTRAINT spans_pkey PRIMARY KEY (trace_id, span_id);

ALTER TABLE ONLY trace_runs
    ADD CONSTRAINT trace_runs_pkey PRIMARY KEY (run_id);

ALTER TABLE ONLY worker_connections
    ADD CONSTRAINT worker_connections_pkey PRIMARY KEY (id, app_name);

CREATE INDEX idx_events_internal_id ON events USING btree (internal_id);

CREATE INDEX idx_events_internal_id_received_range ON events USING btree (internal_id, received_at DESC);

CREATE INDEX idx_events_received_name ON events USING btree (received_at, event_name);

CREATE INDEX idx_function_finishes_run_id ON function_finishes USING btree (run_id);

CREATE INDEX idx_function_runs_event_id ON function_runs USING btree (event_id);

CREATE INDEX idx_function_runs_run_id ON function_runs USING btree (run_id);

CREATE INDEX idx_function_runs_timebound ON function_runs USING btree (run_started_at DESC, function_id);

CREATE INDEX idx_history_id ON history USING btree (id);

CREATE INDEX idx_history_run_id_created ON history USING btree (run_id, created_at);

CREATE INDEX idx_spans_account_status_time ON spans USING btree (account_id, status, start_time);

CREATE INDEX idx_spans_run_id ON spans USING btree (run_id);

CREATE INDEX idx_spans_run_id_dynamic_start_time ON spans USING btree (run_id, dynamic_span_id, start_time);

CREATE INDEX idx_spans_run_status ON spans USING btree (run_id, status);

CREATE INDEX idx_spans_status ON spans USING btree (status);

CREATE INDEX idx_traces_trace_id ON traces USING btree (trace_id);
