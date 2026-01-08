import { anthropic, createAgent } from '@inngest/agent-kit';
import Mustache from 'mustache';

import type { InsightsAgentState as InsightsState } from '../types';
import systemPrompt from './system.md?raw';

export const summarizerAgent = createAgent<InsightsState>({
  name: 'Insights Summarizer',
  description:
    'Writes a concise summary describing what the generated SQL does and why.',
  system: async ({ network }) => {
    const events =
      network?.state.data.selectedEvents?.map(
        (e: { event_name: string }) => e.event_name,
      ) ?? [];
    const sql = network?.state.data.sql;

    return Mustache.render(systemPrompt, {
      hasSelectedEvents: events.length > 0,
      selectedEvents: events.join(', '),
      hasSql: !!sql,
    });
  },
  model: anthropic({
    model: 'claude-haiku-4-5',
    defaultParameters: {
      max_tokens: 4096,
    },
  }),
});
