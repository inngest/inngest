import type { InsightsColumn, InsightsSchemaCatalog } from './types';

const appIdColumn: InsightsColumn = {
  name: 'app_id',
  type: 'UUID',
  description:
    'The app slug as defined in your app, translated to UUID by the query layer.',
  notes: [
    'Write complete app slug strings with = or IN.',
    'Do not compare against a raw UUID.',
    'Do not use LIKE, ILIKE, match, position, substring, lower, upper, concat, or other string functions.',
    'Preferred operators: =, IN.',
    'Forbidden operators: LIKE, ILIKE, match, position, substring, lower, upper, concat.',
  ],
  examples: ["app_id = 'my-app'"],
};

const functionIdColumn: InsightsColumn = {
  name: 'function_id',
  type: 'UUID',
  description:
    'The fully qualified function slug, translated to UUID by the query layer.',
  notes: [
    'Write complete function slug strings with = or IN.',
    'The slug is the app slug concatenated to the function slug with a hyphen, for example my-app-my-function.',
    'Do not compare against a raw UUID.',
    'Do not use LIKE, ILIKE, match, position, substring, lower, upper, concat, or other string functions.',
    'Preferred operators: =, IN.',
    'Forbidden operators: LIKE, ILIKE, match, position, substring, lower, upper, concat.',
  ],
  examples: [
    "function_id = 'my-app-my-function'",
    "function_id IN ('my-app-a', 'my-app-b')",
  ],
};

const attributesColumn: InsightsColumn = {
  name: 'attributes',
  type: 'Map(String, String)',
  description: 'Raw attributes from the underlying span.',
  notes: [
    'Use bracket syntax for arbitrary attribute keys.',
    'Some system attributes are useful for filtering experiment selector spans.',
  ],
  examples: ["attributes['_inngest.step.run.type']"],
  children: [
    {
      name: '<key>',
      type: 'String',
      description: 'An arbitrary span attribute value.',
    },
  ],
};

const inngestMetadataColumn: InsightsColumn = {
  name: 'inngest',
  type: 'Map(String, Tuple(updated_at DateTime, values Dynamic))',
  description: 'System-defined metadata emitted by Inngest.',
  notes: [
    'Use dot syntax for fixed metadata paths.',
    'Use backticks for map keys containing dots, such as score names.',
    'Scores, experiment data, and AI token usage are stored here.',
  ],
  children: [
    {
      name: '<kind>.updated_at',
      type: 'DateTime',
      description: 'When this Inngest metadata value was last updated.',
    },
    {
      name: '<kind>.values',
      type: 'Dynamic',
      description:
        'System-defined metadata payload for an arbitrary Inngest kind.',
    },
  ],
};

const userMetadataColumn: InsightsColumn = {
  name: 'metadata',
  type: 'Map(String, Tuple(updated_at DateTime, values Dynamic))',
  description: 'User-defined metadata attached to runs, steps, or spans.',
  notes: ['Use dot syntax for fixed metadata paths when the key is known.'],
  children: [
    {
      name: '<kind>.updated_at',
      type: 'DateTime',
      description: 'When this metadata value was last updated.',
    },
    {
      name: '<kind>.values',
      type: 'Dynamic',
      description: 'User-defined metadata payload.',
    },
  ],
};

const stepsColumns: InsightsColumn[] = [
  {
    name: 'run_id',
    type: 'ULID String',
    description: 'Unique identifier for the run that owns this step.',
  },
  appIdColumn,
  functionIdColumn,
  {
    name: 'type',
    type: 'String',
    description: 'Step type.',
    notes: [
      'Values: StepRun, StepPlanned, StepFailed, InvokeFunction, Sleep, AIGateway, StepError.',
    ],
  },
  {
    name: 'name',
    type: 'String',
    description:
      'Display name of the step; same as id unless an explicit display name is provided.',
  },
  {
    name: 'id',
    type: 'String',
    description:
      "The id used when creating the step, such as step.run('<id>', ...).",
  },
  {
    name: 'loop_index',
    type: 'Int64',
    description: 'The index for repeated steps.',
  },
  {
    name: 'attempt',
    type: 'Int64',
    description: 'The attempt number for retried steps.',
  },
  {
    name: 'status',
    type: 'String',
    description: 'Step status.',
    notes: ['Values: Queued, Running, Failed, Errored, Completed.'],
  },
  {
    name: 'queued_at',
    type: 'DateTime',
    description: 'When the step was queued.',
  },
  {
    name: 'started_at',
    type: 'Nullable(DateTime)',
    description:
      "When the step started executing; NULL if it hasn't started yet.",
  },
  {
    name: 'ended_at',
    type: 'Nullable(DateTime)',
    description: 'When the step ended; NULL if still running.',
  },
  {
    name: 'output',
    type: 'JSONString',
    description: 'The output or return value from the step.',
    examples: ['output.some_property'],
  },
  {
    name: 'error',
    type: 'JSONString',
    description: 'Error details if the step failed.',
    examples: ['error.message'],
  },
  attributesColumn,
  inngestMetadataColumn,
  userMetadataColumn,
];

export const insightsSchemaCatalog: InsightsSchemaCatalog = {
  version: '2026-06-29',
  tables: [
    {
      name: 'events',
      description: 'Raw ingested event stream.',
      notes: [
        'Rows: One row per ingested event.',
        'Use for: event volumes; event payload fields; what triggered runs; event timestamp analysis.',
        'Avoid for: function run status; use runs; step failures or retries; use steps or step_attempts; AI token usage; use steps or step_attempts.',
        'Notes: Selected event schemas describe the data field for specific event names; ts and received_at are milliseconds since epoch; Prefer ts_dt or received_at_dt for time filtering.',
      ],
      defaultTimeColumn: 'ts_dt',
      columns: [
        {
          name: 'id',
          type: 'String',
          description: 'Unique identifier for the event.',
        },
        {
          name: 'name',
          type: 'String',
          description: 'Event name or type.',
          notes: [
            'Filter with exact event names selected by the event matcher unless the user asks for all events.',
          ],
          examples: ["name = 'app/user.created'"],
        },
        {
          name: 'v',
          type: 'Int64',
          description: 'Event version number.',
        },
        {
          name: 'ts',
          type: 'Int64',
          description: 'Event timestamp in milliseconds since epoch.',
          notes: ['Use millisecond comparisons, not Unix seconds.'],
        },
        {
          name: 'ts_dt',
          type: 'DateTime',
          description: 'Event timestamp as DateTime.',
          notes: ['Recommended column for event time filtering.'],
        },
        {
          name: 'received_at',
          type: 'Int64',
          description: 'Ingestion timestamp in milliseconds since epoch.',
          notes: ['Use millisecond comparisons, not Unix seconds.'],
        },
        {
          name: 'received_at_dt',
          type: 'DateTime',
          description: 'Ingestion timestamp as DateTime.',
        },
        {
          name: 'data',
          type: 'JSONString',
          description: 'JSON payload containing event-specific properties.',
          notes: [
            'Use selected event schemas to decide which data.* properties exist.',
            'Use dot syntax for JSON properties, such as data.user_id or data.nested.value.',
          ],
          examples: ['data.user_id'],
        },
      ],
    },
    {
      name: 'runs',
      description: 'Function executions.',
      notes: [
        'Rows: One row per function run.',
        'Use for: run status; function-level failures; run durations; run inputs and outputs.',
        'Avoid for: step-level failures; use steps; retry analysis; use step_attempts; AI token usage; use steps or step_attempts.',
        'Notes: Use triggering_event_name to group or filter runs by their trigger event.',
      ],
      defaultTimeColumn: 'queued_at',
      columns: [
        {
          name: 'id',
          type: 'ULID String',
          description: 'Unique identifier for the run.',
        },
        appIdColumn,
        functionIdColumn,
        {
          name: 'triggering_event_name',
          type: 'String',
          description: 'The name of the event that triggered the run.',
        },
        {
          name: 'status',
          type: 'String',
          description: 'Run status.',
          notes: ['Values: Queued, Running, Failed, Cancelled, Completed.'],
        },
        {
          name: 'queued_at',
          type: 'DateTime',
          description: 'When the run was queued.',
        },
        {
          name: 'started_at',
          type: 'Nullable(DateTime)',
          description:
            "When the run started executing; NULL if it hasn't started yet.",
        },
        {
          name: 'ended_at',
          type: 'Nullable(DateTime)',
          description: 'When the run ended; NULL if still running.',
        },
        {
          name: 'inputs',
          type: 'Array(JSONString)',
          description:
            'Input events for batch functions or functions triggered by multiple events.',
        },
        {
          name: 'input',
          type: 'JSONString',
          description: 'Equivalent to inputs[1].',
          examples: ['input.data.user_id'],
        },
        {
          name: 'output',
          type: 'JSONString',
          description: 'The output or return value from the function run.',
          examples: ['output.some_property'],
        },
        {
          name: 'error',
          type: 'JSONString',
          description: 'Error details if the run failed.',
          examples: ['error.message'],
        },
        attributesColumn,
        inngestMetadataColumn,
        userMetadataColumn,
        {
          name: 'sessions',
          type: 'Nested(key String, id String)',
          description:
            'Session associations from the triggering event, as an array of key/id pairs.',
          notes: [
            'key names the session type and id is its value.',
            'Select sessions for the pairs, or sessions.key / sessions.id for the parallel key and value arrays.',
          ],
          children: [
            {
              name: 'key',
              type: 'String',
              description: 'Session type name.',
            },
            {
              name: 'id',
              type: 'String',
              description: 'Session identifier value.',
            },
          ],
        },
      ],
    },
    {
      name: 'steps',
      description: 'Latest attempt of each step.',
      notes: [
        'Rows: One row per step, latest attempt only.',
        'Use for: step status; step-level failures; scores; experiments; latest AI token usage by step.',
        'Avoid for: true retry counts; use step_attempts; true AI token totals across retries; use step_attempts.',
        'Notes: steps and step_attempts have the same columns; Use steps when the user cares about current or final step state.',
      ],
      defaultTimeColumn: 'queued_at',
      columns: stepsColumns,
    },
    {
      name: 'step_attempts',
      description: 'Every step attempt, including retries.',
      notes: [
        'Rows: One row per step attempt.',
        'Use for: retry analysis; true AI token totals; step-level failures including retried attempts.',
        'Avoid for: latest step state only; use steps instead.',
        'Notes: Use step_attempts for token totals because retries consume tokens; steps has the same columns but only contains the latest attempt.',
      ],
      defaultTimeColumn: 'queued_at',
      columns: stepsColumns,
    },
    {
      name: 'extended_trace_spans',
      description: 'OpenTelemetry spans for runs and steps.',
      notes: [
        'Rows: One row per span.',
        'Use for: low-level span timing; span hierarchy; OpenTelemetry service and scope analysis; scores on spans.',
        'Avoid for: function run status summaries; use runs; latest step state; use steps; retry analysis; use step_attempts.',
        'Notes: Use parent_span_id and span_id for hierarchy analysis.',
      ],
      defaultTimeColumn: 'start_time',
      columns: [
        {
          name: 'run_id',
          type: 'ULID String',
          description: 'Unique identifier for the run that owns this span.',
        },
        appIdColumn,
        functionIdColumn,
        {
          name: 'step_id',
          type: 'String',
          description:
            "The id used when creating the step, such as step.run('<id>', ...).",
        },
        {
          name: 'step_index',
          type: 'Int64',
          description: 'The index for repeated steps.',
        },
        {
          name: 'step_attempt',
          type: 'Int64',
          description: 'The attempt number for retried steps.',
        },
        {
          name: 'trace_id',
          type: 'String',
          description: 'The OpenTelemetry trace ID.',
        },
        {
          name: 'span_id',
          type: 'String',
          description: 'The OpenTelemetry span ID.',
        },
        {
          name: 'parent_span_id',
          type: 'String',
          description: "The OpenTelemetry span ID of this span's parent.",
        },
        {
          name: 'start_time',
          type: 'DateTime',
          description: 'The start time of the span.',
        },
        {
          name: 'end_time',
          type: 'DateTime',
          description: 'The end time of the span.',
        },
        {
          name: 'name',
          type: 'String',
          description: 'The name of the span.',
        },
        {
          name: 'kind',
          type: 'String',
          description: 'The OpenTelemetry span kind.',
        },
        {
          name: 'scope_name',
          type: 'String',
          description: 'The OpenTelemetry instrumentation scope name.',
        },
        {
          name: 'scope_version',
          type: 'String',
          description: 'The OpenTelemetry instrumentation scope version.',
        },
        {
          name: 'service_name',
          type: 'String',
          description: 'The OpenTelemetry service name.',
        },
        {
          name: 'output',
          type: 'JSONString',
          description: 'Output associated with the span, when available.',
        },
        {
          name: 'error',
          type: 'JSONString',
          description:
            'Error details associated with the span, when available.',
        },
        attributesColumn,
        inngestMetadataColumn,
        userMetadataColumn,
      ],
    },
    {
      name: 'metadata',
      description: 'Span-level system and user metadata.',
      notes: [
        'Rows: One row per span metadata record, at run, step, or extended trace level.',
        'Use for: metadata-specific analysis; finding metadata attached at run, step, or extended trace level; querying metadata level and step_type without reading full span rows; AI token and model metadata when metadata-level fields are needed.',
        'Avoid for: raw event payload fields; use events; run status summaries; use runs; latest step state; use steps.',
        'Notes: level is one of run, step, or extended_trace; it is not a log level; step_type comes from the _inngest.step.type attribute; AI token and model fields are available under inngest.ai.values.* on this table too.',
      ],
      defaultTimeColumn: 'updated_at',
      columns: [
        {
          name: 'run_id',
          type: 'ULID String',
          description: 'Unique identifier for the run.',
        },
        {
          name: 'run_queued_at',
          type: 'DateTime',
          description: 'When the run was queued.',
        },
        {
          name: 'updated_at',
          type: 'DateTime',
          description: 'When the metadata row was last updated.',
        },
        appIdColumn,
        functionIdColumn,
        {
          name: 'step_id',
          type: 'String',
          description:
            "The id used when creating the step, such as step.run('<id>', ...).",
        },
        {
          name: 'step_index',
          type: 'Int64',
          description: 'The index for repeated steps.',
        },
        {
          name: 'step_attempt',
          type: 'Int64',
          description: 'The attempt number for retried steps.',
        },
        {
          name: 'span_id',
          type: 'String',
          description: 'The OpenTelemetry span ID.',
        },
        {
          name: 'level',
          type: 'String',
          description: 'The span level derived from the span name.',
          notes: [
            'Values: run, step, extended_trace.',
            'This is not a log level.',
          ],
        },
        {
          name: 'step_type',
          type: 'String',
          description: 'The step type from the _inngest.step.type attribute.',
          examples: ['run', 'groupExperiment'],
        },
        inngestMetadataColumn,
        userMetadataColumn,
      ],
    },
  ],
};
