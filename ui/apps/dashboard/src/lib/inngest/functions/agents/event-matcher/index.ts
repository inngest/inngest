import {
  anthropic,
  createAgent,
  createTool,
  type AnyZodType,
} from '@inngest/agent-kit';
import Mustache from 'mustache';
import { z } from 'zod';

import {
  ensureObservability,
  OBSERVABILITY_DEFAULTS,
  OBSERVABILITY_LIMITS,
} from '../observability';
import type { InsightsAgentState } from '../types';
import systemPrompt from './system.md?raw';

const SelectEventsParams = z.object({
  events: z
    .array(
      z.object({
        event_name: z.string(),
        reason: z.string(),
      }),
    )
    .min(0)
    .max(6)
    .describe(
      "An array of 0-6 event names selected from the list of available events that best match the user's intent. Use an empty array when the user is asking general questions about events or when updating a query that doesn't filter by event name.",
    ),
});

export const selectEventsTool = createTool({
  name: 'select_events',
  description:
    "Select 0-6 event names from the provided list that are most relevant to the user's query. Return an empty array when no specific events should be filtered.",
  parameters: SelectEventsParams as unknown as AnyZodType, // (ted): need to align zod version; version 3.25 does not support same types as 3.22
  handler: (args: unknown, { network }) => {
    const { events } = args as z.infer<typeof SelectEventsParams>;

    // This should never happen - Zod validates it's an array
    // If this check fails, it indicates a serious type system failure
    if (!Array.isArray(events)) {
      throw new Error('Invalid events parameter: expected array');
    }

    // Empty array is valid - indicates query should not filter by event name
    if (events.length === 0) {
      const reason =
        'No specific events selected - query will include all events.';

      // Persist empty selection on network state for downstream agents
      network.state.data.selectedEvents = [];
      network.state.data.selectionReason = reason;

      // Store output in observability format
      const obs = ensureObservability(
        network,
        'eventMatcher',
        OBSERVABILITY_DEFAULTS.eventMatcher,
      );
      obs.output = {
        selectedEvents: [],
        selectionReason: reason,
      };

      return {
        selected: [],
        reason,
        totalCandidates: network.state.data.eventTypes?.length || 0,
      };
    }

    const selected = events.map((event) => {
      return {
        event_name: event.event_name,
        reason: event.reason,
      };
    });

    const reason = "Selected by the LLM based on the user's query.";

    // Persist selection on network state for downstream agents
    network.state.data.selectedEvents = events;
    network.state.data.selectionReason = reason;

    // Store output in observability format
    const obs = ensureObservability(
      network,
      'eventMatcher',
      OBSERVABILITY_DEFAULTS.eventMatcher,
    );
    obs.output = {
      selectedEvents: selected,
      selectionReason: reason,
    };

    const result = {
      selected,
      reason,
      totalCandidates: network.state.data.eventTypes?.length || 0,
    };
    return result;
  },
});

export const eventMatcherAgent = createAgent<InsightsAgentState>({
  name: 'Insights Event Matcher',
  description:
    "Analyzes available events and selects 1-5 that best match the user's intent.",
  system: async ({ network }): Promise<string> => {
    const events = network?.state.data.eventTypes || [];
    const sample = events.slice(0, 500); // avoid overly long prompts
    const currentQuery = network?.state.data.currentQuery;

    // Prepare context for system prompt hydration
    const promptContext = {
      totalEvents: events.length,
      hasEvents: sample.length > 0,
      eventsList: sample.join('\n'),
      maxEvents: 500,
      hasCurrentQuery: !!currentQuery,
      currentQuery: currentQuery || '',
    };

    // Store prompt context in observability format
    if (network?.state.data) {
      const obs = ensureObservability(
        network,
        'eventMatcher',
        OBSERVABILITY_DEFAULTS.eventMatcher,
      );
      obs.promptContext = {
        ...promptContext,
        // Truncate current query for observability
        currentQuery: currentQuery
          ? currentQuery.substring(0, OBSERVABILITY_LIMITS.CURRENT_QUERY_LENGTH)
          : '',
        currentQueryLength: currentQuery?.length || 0,
      };
    }

    return Mustache.render(systemPrompt, promptContext);
  },
  model: anthropic({
    model: 'claude-haiku-4-5',
    defaultParameters: {
      max_tokens: 4096,
    },
  }),
  tools: [selectEventsTool],
  tool_choice: 'select_events',
});
