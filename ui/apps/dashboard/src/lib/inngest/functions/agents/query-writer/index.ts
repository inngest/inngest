import {
  anthropic,
  createAgent,
  createTool,
  type AnyZodType,
} from '@inngest/agent-kit';
import Mustache from 'mustache';
import { z } from 'zod';

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
    if (!network.state.data.observability) {
      network.state.data.observability = {};
    }
    if (!network.state.data.observability.queryWriter) {
      network.state.data.observability.queryWriter = {
        promptContext: {
          selectedEventsCount: 0,
          selectedEventNames: [],
          schemasCount: 0,
          schemaNames: [],
          schemas: [],
        },
      };
    }
    network.state.data.observability.queryWriter.output = {
      sql,
      title,
      reasoning,
    };

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

    // Prepare context for system prompt hydration
    const promptContext = {
      hasSelectedEvents: selectedEvents.length > 0,
      selectedEvents: selectedEvents.join(', '),
      hasSchemas: selectedSchemas.length > 0,
      schemas: selectedSchemas,
    };

    // Store prompt context in observability format with schemas
    if (network?.state.data) {
      if (!network.state.data.observability) {
        network.state.data.observability = {};
      }
      network.state.data.observability.queryWriter = {
        promptContext: {
          selectedEventsCount: selectedEvents.length,
          selectedEventNames: selectedEvents,
          schemasCount: selectedSchemas.length,
          schemaNames: selectedSchemas.map((s) => s.eventName),
          // Include actual schemas (truncated for observability)
          schemas: selectedSchemas.map((schema) => ({
            eventName: schema.eventName,
            schema: schema.schema.substring(0, 2000),
            schemaLength: schema.schema.length,
          })),
        },
      };
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
