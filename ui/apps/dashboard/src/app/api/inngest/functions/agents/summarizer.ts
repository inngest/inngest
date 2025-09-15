import { createAgent, openai } from '@inngest/agent-kit';

import type { InsightsAgentState as InsightsState } from './event-matcher';

export const summarizerAgent = createAgent<InsightsState>({
  name: 'Insights Summarizer',
  description: 'Writes a concise summary describing what the generated SQL does and why.',
  system: async ({ network }): Promise<string> => {
    const events = Array.isArray(network?.state?.data?.selectedEvents)
      ? (network!.state.data.selectedEvents as string[])
      : [];
    const sql =
      typeof network?.state?.data?.sql === 'string'
        ? (network!.state.data.sql as string)
        : undefined;
    return [
      'You are a helpful assistant summarizing the result of a SQL generation process.',
      'Write a one sentence short summary that explains:',
      '- What events were just analyzed (if known).',
      '- What the query returns and how it helps the user.',
      'Avoid restating the full SQL. Be clear and non-technical when possible.',
      events.length ? `Selected events: ${events.join(', ')}` : '',
      sql ? 'A SQL statement has been prepared; summarize its intent, not its exact text.' : '',
    ]
      .filter(Boolean)
      .join('\n');
  },
  model: openai({ model: 'gpt-5-nano-2025-08-07' }),
  tools: [],
});
