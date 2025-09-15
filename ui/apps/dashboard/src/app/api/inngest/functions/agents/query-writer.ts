import { createAgent, createTool, openai, type Network } from '@inngest/agent-kit';
import { z } from 'zod';

import type { InsightsAgentState as InsightsState } from './event-matcher';
import type { GenerateSqlInput, GenerateSqlResult } from './types';

function sanitizeSql(text: string): string {
  const sql = String(text || '').trim();
  // Lightweight guardrail: reject clearly unsafe statements
  const lower = sql.replace(/\s+/g, ' ').toLowerCase();
  const forbidden = [
    'insert ',
    'update ',
    'delete ',
    'drop ',
    'alter ',
    'create ',
    'grant ',
    'revoke ',
    'truncate ',
  ];
  for (const kw of forbidden) {
    if (lower.includes(kw)) {
      throw new Error('Only read-only SELECT queries are allowed');
    }
  }
  if (!/^select\s/i.test(sql)) {
    throw new Error('SQL must start with SELECT');
  }
  return sql;
}

const generateSqlTool = createTool({
  name: 'generate_sql',
  description:
    'Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.',
  parameters: z
    .object({
      sql: z
        .string()
        .min(1)
        .describe(
          'A single valid SELECT statement. Do not include DDL/DML or multiple statements.'
        ),
      title: z.string().min(1).describe('Short 20-30 character title for this query'),
      reasoning: z
        .string()
        .min(1)
        .describe('Brief 1-2 sentence explanation of how this query addresses the request'),
    })
    .strict() as any,
  handler: ({ sql: rawSql, title, reasoning }: GenerateSqlInput, ctx: any): GenerateSqlResult => {
    const network = (ctx?.network as Network<InsightsState>) || undefined;
    const raw = typeof rawSql === 'string' ? (rawSql as string) : '';
    const sql = sanitizeSql(raw);
    network.state.data.sql = sql;

    const result: GenerateSqlResult = {
      sql: sql,
      title,
      reasoning,
    };
    return result;
  },
});

export const queryWriterAgent = createAgent<InsightsState>({
  name: 'Insights Query Writer',
  description: 'Generates a safe, read-only SQL SELECT statement for ClickHouse.',
  system: async ({ network }): Promise<string> => {
    const selected = Array.isArray(network?.state?.data?.selectedEvents)
      ? (network!.state.data.selectedEvents as string[])
      : [];
    return [
      'You write ClickHouse-compatible SQL for analytics.',
      '',
      'Constraints:',
      '- Only SELECT statements are allowed.',
      '- Never include INSERT/UPDATE/DELETE/DDL.',
      '- Use event filters (e.g., WHERE event_name IN (...)) when applicable.',
      '- Favor readable aliases and consistent casing.',
      '',
      selected.length
        ? `Target the following events if relevant: ${selected.join(', ')}`
        : 'If events were selected earlier, incorporate them appropriately.',
      '',
      'When ready, call the propose_sql tool with the final SQL and, optionally, a short title.',
    ].join('\n');
  },
  model: openai({ model: 'gpt-5-nano-2025-08-07' }),
  tools: [generateSqlTool],
  tool_choice: 'any',
});
