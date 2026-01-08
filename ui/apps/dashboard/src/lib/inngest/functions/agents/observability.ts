import type { Network } from '@inngest/agent-kit';

import type { InsightsAgentState } from './types';

/**
 * Truncation limits for observability data to prevent excessively large payloads.
 * These limits ensure observability data remains manageable while preserving useful context.
 */
export const OBSERVABILITY_LIMITS = {
  /** Maximum length for current query strings in observability data */
  CURRENT_QUERY_LENGTH: 500,
  /** Maximum length for schema strings in observability data */
  SCHEMA_LENGTH: 2000,
  /** Maximum length for events list strings in observability data */
  EVENTS_LIST_LENGTH: 5000,
} as const;

/**
 * Ensures the observability structure exists for a specific agent and returns it.
 * This helper prevents duplication of initialization logic across agents.
 *
 * @param network - The agent network instance
 * @param agent - The agent name (eventMatcher, queryWriter, or summarizer)
 * @param defaults - Default values for the agent's observability structure
 * @returns The agent's observability object
 */
export function ensureObservability<
  T extends keyof NonNullable<InsightsAgentState['observability']>,
>(
  network: Network<InsightsAgentState>,
  agent: T,
  defaults: NonNullable<InsightsAgentState['observability']>[T],
): NonNullable<NonNullable<InsightsAgentState['observability']>[T]> {
  if (!network.state.data.observability) {
    network.state.data.observability = {};
  }
  if (!network.state.data.observability[agent]) {
    network.state.data.observability[agent] = defaults as any;
  }
  return network.state.data.observability[agent] as NonNullable<
    NonNullable<InsightsAgentState['observability']>[T]
  >;
}

/**
 * Default observability structures for each agent.
 * These provide type-safe defaults to avoid duplication.
 */
export const OBSERVABILITY_DEFAULTS: {
  eventMatcher: NonNullable<
    NonNullable<InsightsAgentState['observability']>['eventMatcher']
  >;
  queryWriter: NonNullable<
    NonNullable<InsightsAgentState['observability']>['queryWriter']
  >;
  summarizer: NonNullable<
    NonNullable<InsightsAgentState['observability']>['summarizer']
  >;
} = {
  eventMatcher: {
    promptContext: {
      totalEvents: 0,
      hasEvents: false,
      eventsList: '',
      maxEvents: 0,
      hasCurrentQuery: false,
      currentQuery: '',
      currentQueryLength: 0,
    },
  },
  queryWriter: {
    promptContext: {
      selectedEventsCount: 0,
      selectedEventNames: [],
      schemasCount: 0,
      schemaNames: [],
      schemas: [],
      hasCurrentQuery: false,
      currentQueryLength: 0,
    },
  },
  summarizer: {
    promptContext: {
      selectedEventsCount: 0,
      selectedEventNames: [],
      hasSql: false,
    },
  },
};
