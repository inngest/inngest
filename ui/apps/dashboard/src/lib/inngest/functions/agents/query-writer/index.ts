import Mustache from 'mustache';
import { z } from 'zod';

import systemPrompt from './system.md?raw';

// Zod schema for the generate_sql tool (structured output extraction)
export const GenerateSqlParams = z.object({
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
});

// Anthropic tool definition for step.ai.infer()
export const generateSqlTool = {
  name: 'generate_sql' as const,
  description:
    'Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.',
  input_schema: z.toJSONSchema(GenerateSqlParams) as {
    type: 'object';
    [k: string]: unknown;
  },
};

/**
 * Build the query writer system prompt by hydrating the Mustache template
 * with selected events, schemas, optional current query, and user query.
 */
export function buildSystemPrompt(params: {
  selectedEvents: { event_name: string; reason: string }[];
  schemas: { name: string; schema: string }[];
  currentQuery?: string;
  query: string;
}): string {
  const selectedEventNames = params.selectedEvents.map((e) => e.event_name);

  // Filter schemas to only include selected events
  const selectedSchemas = params.schemas
    .filter((schema) => selectedEventNames.includes(schema.name))
    .map((schema) => ({
      eventName: schema.name,
      schema: schema.schema,
    }));

  const promptContext = {
    hasSelectedEvents: selectedEventNames.length > 0,
    selectedEvents: selectedEventNames.join(', '),
    hasSchemas: selectedSchemas.length > 0,
    schemas: selectedSchemas,
    hasCurrentQuery: !!params.currentQuery,
    currentQuery: params.currentQuery || '',
    query: params.query || '',
  };

  return Mustache.render(systemPrompt, promptContext);
}

export type GenerateSqlResult = {
  sql: string;
  title?: string;
  reasoning?: string;
};

/**
 * Parse the Anthropic Messages API response to extract the generate_sql
 * tool call result.
 */
export function parseToolResult(result: {
  content: Array<{
    type: string;
    name?: string;
    input?: unknown;
  }>;
}): GenerateSqlResult {
  const toolUse = result.content.find(
    (block) => block.type === 'tool_use' && block.name === 'generate_sql',
  );

  if (!toolUse || !('input' in toolUse)) {
    throw new Error('Expected generate_sql tool call not found in response');
  }

  const input = toolUse.input as z.infer<typeof GenerateSqlParams>;

  return {
    sql: input.sql,
    title: input.title,
    reasoning: input.reasoning,
  };
}
