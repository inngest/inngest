import { createNetwork, openai, type Network, type State } from '@inngest/agent-kit';

import { eventMatcherAgent } from './event-matcher';
import { queryWriterAgent } from './query-writer';
import { summarizerAgent } from './summarizer';
import type { InsightsAgentState as InsightsState } from './types';

// Deterministic, code-first router: EventMatcher → QueryWriter → Summarizer
const sequenceRouter: Network.Router<InsightsState> = async ({ callCount }) => {
  if (callCount === 0) return eventMatcherAgent;
  if (callCount === 1) return queryWriterAgent;
  if (callCount === 2) return summarizerAgent;
  return undefined;
};

export function createInsightsNetwork(
  _threadId: string,
  initialState: State<InsightsState>,
  historyAdapter?: any
) {
  return createNetwork<InsightsState>({
    name: 'Insights SQL Generation Network',
    description: 'Selects relevant events, proposes a SQL query, and summarizes the result.',
    agents: [eventMatcherAgent, queryWriterAgent, summarizerAgent],
    defaultModel: openai({ model: 'gpt-5-nano-2025-08-07' }),
    maxIter: 6,
    defaultState: initialState,
    router: sequenceRouter,
    history: historyAdapter,
  });
}

export { eventMatcherAgent, queryWriterAgent, summarizerAgent };
