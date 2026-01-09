import {
  anthropic,
  createAgent,
  createTool,
  type AnyZodType,
} from '@inngest/agent-kit';
import Mustache from 'mustache';
import { z } from 'zod';

import { setObservability, OBSERVABILITY_LIMITS } from '../observability';
import type { InsightsAgentState } from '../types';
import systemPrompt from './system.md?raw';

const GenerateSqlParams = z.object({
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

export const generateSqlTool = createTool({
  name: 'generate_sql',
  description:
    'Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.',
  parameters: GenerateSqlParams as unknown as AnyZodType, // (ted): need to update to latest version of zod + agent-kit
  handler: (args: unknown, { network }) => {
    const { sql, title, reasoning } = args as z.infer<typeof GenerateSqlParams>;

    // Store output in observability format
    setObservability(network, 'queryWriter', {
      output: {
        sql,
        title,
        reasoning,
      },
    });

    return {
      sql,
      title,
      reasoning,
    };
  },
});

export const queryWriterAgent = createAgent<InsightsAgentState>({
  name: 'Insights Query Writer',
  description:
    'Generates a safe, read-only SQL SELECT statement for ClickHouse.',
  system: async ({ network }) => {
    const selectedEvents =
      network?.state.data.selectedEvents?.map(
        (e: { event_name: string }) => e.event_name,
      ) ?? [];

    // Filter schemas to only include selected events
    const allSchemas = network?.state.data.schemas ?? [];
    const selectedSchemas = allSchemas
      .filter((schema) => selectedEvents.includes(schema.name))
      .map((schema) => ({
        eventName: schema.name,
        schema: schema.schema,
      }));

    // Get current query if it exists
    const currentQuery = network?.state.data.currentQuery;

    // Prepare context for system prompt hydration
    const promptContext = {
      hasSelectedEvents: selectedEvents.length > 0,
      selectedEvents: selectedEvents.join(', '),
      hasSchemas: selectedSchemas.length > 0,
      schemas: selectedSchemas,
      hasCurrentQuery: !!currentQuery,
      currentQuery: currentQuery || '',
    };

    // Store prompt context in observability format with schemas
    if (network?.state.data) {
      setObservability(network, 'queryWriter', {
        promptContext: {
          selectedEventsCount: selectedEvents.length,
          selectedEventNames: selectedEvents,
          schemasCount: selectedSchemas.length,
          schemaNames: selectedSchemas.map((s) => s.eventName),
          // Include actual schemas (truncated for observability)
          schemas: selectedSchemas.map((schema) => ({
            eventName: schema.eventName,
            schema: schema.schema.substring(
              0,
              OBSERVABILITY_LIMITS.SCHEMA_LENGTH,
            ),
            schemaLength: schema.schema.length,
          })),
          hasCurrentQuery: !!currentQuery,
          currentQueryLength: currentQuery?.length || 0,
        },
      });
    }

    return Mustache.render(systemPrompt, promptContext);
  },
  model: anthropic({
    model: 'claude-sonnet-4-5-20250929',
    defaultParameters: {
      max_tokens: 4096,
    },
  }),
  tools: [generateSqlTool],
  tool_choice: 'generate_sql',
});
