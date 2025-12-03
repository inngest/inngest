import { anthropic, createAgent } from '@inngest/agent-kit';

import systemPrompt from './system.md';
import type { InsightsAgentState as InsightsState } from './types';

export const summarizerAgent = createAgent<InsightsState>({
  name: 'Insights Summarizer',
  description: 'Writes a concise summary describing what the generated SQL does and why.',
  system: async ({ network }) => {
    const events = network?.state.data.selectedEvents?.map((e) => e.event_name) ?? [];
    const sql = network?.state.data.sql;
    return [
      systemPrompt,
      events.length ? `Selected events: ${events.join(', ')}` : '',
      sql ? 'A SQL statement has been prepared; summarize its intent, not its exact text.' : '',
    ]
      .filter(Boolean)
      .join('\n');
  },
  model: anthropic({
    model: 'claude-haiku-4-5',
    defaultParameters: {
      max_tokens: 4096,
    },
  }),
});
