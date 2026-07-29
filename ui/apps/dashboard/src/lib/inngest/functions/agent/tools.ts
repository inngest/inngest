import { z } from 'zod';

import { VALIDATE_QUERY, type ToolDef } from './loop';

// Tools for the Insights agent. The static table schemas live in the system
// prompt (system.md); these tools cover the dynamic events layer (names and
// per-event data schemas come from client state), validation, and submission.

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
      'List the event type names available in this environment, optionally filtered by a search term. Use only when querying the events table and you need exact event names.',
    input_schema: z.toJSONSchema(FindEventsParams) as {
      type: 'object';
      [k: string]: unknown;
    },
  },
  execute: async (input, ctx) => {
    const parsed = FindEventsParams.safeParse(input);
    if (!parsed.success) {
      return { observation: `Invalid input: ${parsed.error.message}` };
    }

    const all = ctx.clientState.eventTypes ?? [];
    const term = parsed.data.search?.toLowerCase();
    const matches = term
      ? all.filter((name) => name.toLowerCase().includes(term))
      : all;
    const shown = matches.slice(0, 200);

    if (matches.length === 0) {
      return { observation: 'No matching event types.' };
    }
    const truncation =
      matches.length > shown.length ? ` (showing first ${shown.length})` : '';
    return {
      observation: `${
        matches.length
      } matching event type(s)${truncation}:\n${shown.join('\n')}`,
    };
  },
};

const GetEventSchemasParams = z.object({
  event_names: z
    .array(z.string())
    .min(1)
    .max(6)
    .describe('Event names to fetch JSON data schemas for.'),
});

export const getEventSchemasTool: ToolDef = {
  tool: {
    name: 'get_event_schemas',
    description:
      'Fetch the JSON schema for specific event names so you can reference real data.* field paths. Call this before referencing event data fields in SQL.',
    input_schema: z.toJSONSchema(GetEventSchemasParams) as {
      type: 'object';
      [k: string]: unknown;
    },
  },
  execute: async (input, ctx) => {
    const parsed = GetEventSchemasParams.safeParse(input);
    if (!parsed.success) {
      return { observation: `Invalid input: ${parsed.error.message}` };
    }

    const schemas = ctx.clientState.schemas ?? [];
    const parts = parsed.data.event_names.map((name) => {
      const found = schemas.find((s) => s.name === name);
      return found
        ? `### ${name}\n${found.schema}`
        : `### ${name}\n(no schema available — avoid referencing this event's data fields)`;
    });
    return { observation: parts.join('\n\n') };
  },
};

// Execution is intercepted by the loop (it needs durable wait primitives);
// this entry only contributes the tool schema the model sees.
export const validateQueryTool: ToolDef = {
  tool: {
    name: VALIDATE_QUERY,
    description:
      'Run a SQL query against the live environment to check that it executes. Returns the result columns and row count, or the exact error diagnostics. Always validate before submit_query; fix and re-validate on failure (at most 2 retries).',
    input_schema: z.toJSONSchema(
      z.object({
        sql: z
          .string()
          .min(1)
          .describe('The ClickHouse SELECT statement to validate.'),
      }),
    ) as { type: 'object'; [k: string]: unknown },
  },
  execute: async () => {
    throw new Error('validate_query is executed by the agent loop');
  },
};

const SubmitQueryParams = z.object({
  sql: z
    .string()
    .min(1)
    .describe(
      'A single valid SELECT statement. Do not include DDL/DML or multiple statements.',
    ),
  title: z
    .string()
    .min(1)
    .describe('Short 20-30 character title for this query'),
  reasoning: z
    .string()
    .min(1)
    .describe(
      'Brief 1-2 sentence explanation of how this query addresses the request',
    ),
  event_names: z
    .array(z.string())
    .default([])
    .describe(
      'If the query reads the events table, the event names it filters on. Otherwise empty.',
    ),
});

export const submitQueryTool: ToolDef = {
  tool: {
    name: 'submit_query',
    description:
      'Record the final SQL query for the user. Call exactly once, when the query is ready (after validating it if validate_query is available), then respond with a short natural-language summary to finish.',
    input_schema: z.toJSONSchema(SubmitQueryParams) as {
      type: 'object';
      [k: string]: unknown;
    },
  },
  execute: async (input) => {
    const parsed = SubmitQueryParams.safeParse(input);
    if (!parsed.success) {
      return { observation: `Invalid input: ${parsed.error.message}` };
    }

    const { sql, title, reasoning, event_names } = parsed.data;
    return {
      observation:
        'Query recorded. Now respond with a 1-2 sentence summary of what the query does — that text ends the run.',
      draftPatch: {
        sql,
        title,
        reasoning,
        selectedEvents: event_names.map((event_name) => ({
          event_name,
          reason: 'used in query',
        })),
      },
      publish: {
        event: 'step.completed',
        data: { step: 'query-writer', sql, title, reasoning },
      },
    };
  },
};

export const insightsTools: ToolDef[] = [
  findEventsTool,
  getEventSchemasTool,
  submitQueryTool,
];
