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

type DeepPartial<T> = T extends object
  ? {
      [P in keyof T]?: DeepPartial<T[P]>;
    }
  : T;

type ObservabilityData = NonNullable<InsightsAgentState['observability']>;

/**
 * Sets observability data for a specific agent.
 * Ensures the observability structure exists and merges in the provided data.
 *
 * @param network - The agent network instance
 * @param agent - The agent name (eventMatcher, queryWriter, or summarizer)
 * @param data - Observability data to set (promptContext and/or output)
 */
export function setObservability<T extends keyof ObservabilityData>(
  network: Network<InsightsAgentState>,
  agent: T,
  data: DeepPartial<ObservabilityData[T]>,
) {
  if (!network.state.data.observability) {
    network.state.data.observability = {};
  }
  const obs = network.state.data.observability;
  const merged = { ...obs[agent], ...data };
  obs[agent] = merged;
}
