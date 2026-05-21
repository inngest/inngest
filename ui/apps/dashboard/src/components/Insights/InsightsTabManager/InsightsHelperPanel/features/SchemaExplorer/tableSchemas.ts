import type { SchemaNode } from '@inngest/components/SchemaViewer/types';
import type { SchemaEntry } from './SchemasContext/types';

const metadataNode = (table: string, column: string): SchemaNode => ({
  kind: 'object',
  name: column,
  path: `${table}.${column}`,
  type: 'JSON',
  children: [
    {
      kind: 'object',
      name: '<kind>',
      path: `${table}.${column}.<kind>`,
      type: 'String',
      children: [
        {
          kind: 'value',
          name: 'updated_at',
          path: `${table}.${column}.<kind>.updated_at`,
          type: 'DateTime',
        },
        {
          kind: 'value',
          name: 'values',
          path: `${table}.${column}.<kind>.values`,
          type: 'JSON',
        },
      ],
    },
  ],
});

const attributesNode = (table: string, column: string): SchemaNode => ({
  kind: 'object',
  name: column,
  path: `${table}.${column}`,
  type: 'JSON',
  children: [
    {
      kind: 'value',
      name: '<key>',
      path: `${table}.${column}.<attr>`,
      type: 'String',
    },
  ],
});

const stepsTable = (table: string): SchemaEntry => ({
  key: table,
  node: {
    kind: 'table',
    name: table,
    path: table,
    children: [
      {
        kind: 'value',
        name: 'run_id',
        path: `${table}.run_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'app_id',
        path: `${table}.app_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'function_id',
        path: `${table}.function_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'type',
        path: `${table}.type`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'name',
        path: `${table}.name`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'id',
        path: `${table}.id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'loop_index',
        path: `${table}.loop_index`,
        type: 'Integer',
      },
      {
        kind: 'value',
        name: 'attempt',
        path: `${table}.attempt`,
        type: 'Integer',
      },
      {
        kind: 'value',
        name: 'status',
        path: `${table}.status`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'queued_at',
        path: `${table}.queued_at`,
        type: 'DateTime',
      },
      {
        kind: 'value',
        name: 'started_at',
        path: `${table}.started_at`,
        type: 'DateTime',
      },
      {
        kind: 'value',
        name: 'ended_at',
        path: `${table}.ended_at`,
        type: 'DateTime',
      },
      {
        kind: 'value',
        name: 'output',
        path: `${table}.output`,
        type: 'JSON',
      },
      {
        kind: 'value',
        name: 'error',
        path: `${table}.error`,
        type: 'JSON',
      },
      attributesNode(table, 'attributes'),
      metadataNode(table, 'inngest'),
      metadataNode(table, 'metadata'),
    ],
  },
});

const extendedTraceSpansTable = (table: string): SchemaEntry => ({
  key: table,
  node: {
    kind: 'table',
    name: table,
    path: table,
    children: [
      {
        kind: 'value',
        name: 'run_id',
        path: `${table}.run_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'app_id',
        path: `${table}.app_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'function_id',
        path: `${table}.function_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'step_id',
        path: `${table}.step_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'step_index',
        path: `${table}.step_index`,
        type: 'Integer',
      },
      {
        kind: 'value',
        name: 'step_attempt',
        path: `${table}.step_attempt`,
        type: 'Integer',
      },
      {
        kind: 'value',
        name: 'span_id',
        path: `${table}.span_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'parent_span_id',
        path: `${table}.parent_span_id`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'start_time',
        path: `${table}.start_time`,
        type: 'DateTime',
      },
      {
        kind: 'value',
        name: 'end_time',
        path: `${table}.end_time`,
        type: 'DateTime',
      },
      {
        kind: 'value',
        name: 'name',
        path: `${table}.name`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'kind',
        path: `${table}.kind`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'scope_name',
        path: `${table}.scope_name`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'scope_version',
        path: `${table}.scope_version`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'service_name',
        path: `${table}.service_name`,
        type: 'String',
      },
      {
        kind: 'value',
        name: 'output',
        path: `${table}.output`,
        type: 'JSON',
      },
      {
        kind: 'value',
        name: 'error',
        path: `${table}.error`,
        type: 'JSON',
      },
      attributesNode(table, 'attributes'),
      metadataNode(table, 'inngest'),
      metadataNode(table, 'metadata'),
    ],
  },
});

export const tableEntries: SchemaEntry[] = [
  {
    key: 'events',
    node: {
      kind: 'table',
      name: 'events',
      path: 'events',
      children: [
        {
          kind: 'value',
          name: 'name',
          path: 'events.name',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'id',
          path: 'events.id',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'data',
          path: 'events.data',
          type: 'JSON',
        },
        {
          kind: 'value',
          name: 'ts',
          path: 'events.ts',
          type: 'Integer',
        },
        {
          kind: 'value',
          name: 'ts_dt',
          path: 'events.ts_dt',
          type: 'DateTime',
        },
        {
          kind: 'value',
          name: 'received_at',
          path: 'events.received_at',
          type: 'Integer',
        },
        {
          kind: 'value',
          name: 'received_at_dt',
          path: 'events.received_at_dt',
          type: 'DateTime',
        },
      ],
    },
  },
  {
    key: 'runs',
    node: {
      kind: 'table',
      name: 'runs',
      path: 'runs',
      children: [
        {
          kind: 'value',
          name: 'id',
          path: 'runs.id',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'app_id',
          path: 'runs.app_id',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'function_id',
          path: 'runs.function_id',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'triggering_event_name',
          path: 'runs.triggering_event_name',
          type: 'String',
        },
        {
          kind: 'value',
          name: 'queued_at',
          path: 'runs.queued_at',
          type: 'DateTime',
        },
        {
          kind: 'value',
          name: 'started_at',
          path: 'runs.started_at',
          type: 'DateTime',
        },
        {
          kind: 'value',
          name: 'ended_at',
          path: 'runs.ended_at',
          type: 'DateTime',
        },
        {
          kind: 'value',
          name: 'inputs',
          path: 'runs.inputs',
          type: 'Array(JSON)',
        },
        {
          kind: 'value',
          name: 'input',
          path: 'runs.input',
          type: 'JSON',
        },
        {
          kind: 'value',
          name: 'output',
          path: 'runs.output',
          type: 'JSON',
        },
        {
          kind: 'value',
          name: 'error',
          path: 'runs.error',
          type: 'JSON',
        },
      ],
    },
  },
  stepsTable('steps'),
  stepsTable('step_attempts'),
  extendedTraceSpansTable('extended_trace_spans'),
];
