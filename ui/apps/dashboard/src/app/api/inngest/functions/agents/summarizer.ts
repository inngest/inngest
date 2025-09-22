import { createAgent, openai } from '@inngest/agent-kit';

import type { InsightsAgentState as InsightsState } from './types';

export const summarizerAgent = createAgent<InsightsState>({
  name: 'Insights Summarizer',
  description: 'Writes a concise summary describing what the generated SQL does and why.',
  system: async ({ network }) => {
    const events = network?.state.data.selectedEvents?.map((e) => e.event_name) ?? [];
    const sql = network?.state.data.sql;
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
});
