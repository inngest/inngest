import { z } from 'zod';

import type { ToolDef } from './loop';
import { describeTable, listDataSources } from './tables';

const NoParams = {
  type: 'object',
  properties: {},
  additionalProperties: false,
} as const;

export const listDataSourcesTool: ToolDef = {
  tool: {
    name: 'list_data_sources',
    description:
      'List the tables you can query and what each one is for (events, runs, steps, ...). Call this first when it is not obvious which table answers the user.',
    input_schema: NoParams as any,
  },
  execute: async () => ({ observation: listDataSources() }),
};

const DescribeTableParams = z.object({
  table_name: z
    .string()
    .describe('A table name from list_data_sources, e.g. "runs" or "steps".'),
});
export const describeTableTool: ToolDef = {
  tool: {
    name: 'describe_table',
    description:
      'Show the columns and types for a table. Call this for the table you intend to query before writing SQL.',
    input_schema: z.toJSONSchema(DescribeTableParams) as any,
  },
  execute: async (input: z.infer<typeof DescribeTableParams>) => ({
    observation: describeTable(input.table_name),
  }),
};

const FindEventsParams = z.object({
  search: z
    .string()
    .optional()
    .describe('Optional case-insensitive substring to filter event names.'),
});
export const findEventsTool: ToolDef = {
  tool: {
    name: 'find_events',
    description:
      'List the event type names available in this environment, optionally filtered. Use only when querying the events table and you need exact event names.',
    input_schema: z.toJSONSchema(FindEventsParams) as any,
  },
  execute: async (input: z.infer<typeof FindEventsParams>, ctx) => {
    const all = ctx.clientState.eventTypes ?? [];
    const term = input.search?.toLowerCase();
    const matches = term
      ? all.filter((e) => e.toLowerCase().includes(term))
      : all;
    const shown = matches.slice(0, 200);
    const observation =
      matches.length === 0
        ? 'No matching event types.'
        : `${matches.length} matching event type(s)${
            matches.length > shown.length
              ? ` (showing first ${shown.length})`
              : ''
          }:\n${shown.join('\n')}`;
    return { observation };
  },
};

const GetEventSchemasParams = z.object({
  event_names: z
    .array(z.string())
    .min(1)
    .describe('Event names to fetch JSON data schemas for.'),
});
export const getEventSchemasTool: ToolDef = {
  tool: {
    name: 'get_event_schemas',
    description:
      'Fetch the JSON schema(s) for specific event names so you can reference real data.* field paths. Call before reading event data fields.',
    input_schema: z.toJSONSchema(GetEventSchemasParams) as any,
  },
  execute: async (input: z.infer<typeof GetEventSchemasParams>, ctx) => {
    const schemas = ctx.clientState.schemas ?? [];
    const parts = input.event_names.map((name) => {
      const found = schemas.find((s) => s.name === name);
      return found
        ? `### ${name}\n${found.schema}`
        : `### ${name}\n(no schema available — query without referencing its data fields)`;
    });
    return { observation: parts.join('\n\n') };
  },
};

const SubmitQueryParams = z.object({
  sql: z
    .string()
    .min(1)
    .describe(
      'A single valid ClickHouse SELECT statement. No DDL/DML, no multiple statements.',
    ),
  title: z
    .string()
    .min(1)
    .describe('Short 20-30 character title for this query.'),
  reasoning: z
    .string()
    .min(1)
    .describe(
      'Brief 1-2 sentence explanation of how the query addresses the request.',
    ),
  tables: z
    .array(z.string())
    .min(1)
    .describe('The table(s) this query reads from, e.g. ["runs"].'),
  event_names: z
    .array(z.string())
    .default([])
    .describe(
      'If querying the events table, the event names this query filters on or reads from.',
    ),
});
export const submitQueryTool: ToolDef = {
  tool: {
    name: 'submit_query',
    description:
      'Record the final SQL query (call exactly once when confident), then respond with a short natural-language summary to finish the run.',
    input_schema: z.toJSONSchema(SubmitQueryParams) as any,
  },
  execute: async (input: z.infer<typeof SubmitQueryParams>) => {
    const selectedEvents = input.event_names.map((event_name) => ({
      event_name,
      reason: 'used in query',
    }));
    return {
      observation:
        'Query recorded. Now respond with a short natural-language summary of what the query does — that text ends the run.',
      draftPatch: {
        sql: input.sql,
        title: input.title,
        reasoning: input.reasoning,
        tables: input.tables,
        selectedEvents,
      },
      publish: {
        event: 'step.completed',
        data: {
          step: 'query-writer',
          sql: input.sql,
          title: input.title,
          reasoning: input.reasoning,
          tables: input.tables,
        },
      },
    };
  },
};

export const insightsTools: ToolDef[] = [
  listDataSourcesTool,
  describeTableTool,
  findEventsTool,
  getEventSchemasTool,
  submitQueryTool,
];
