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

const SelectEventsParams = z.object({
  events: z
    .array(
      z.object({
        event_name: z.string(),
        reason: z.string(),
      }),
    )
    .min(1)
    .max(6)
    .describe(
      "An array of 1-6 event names selected from the list of available events that best match the user's intent.",
    ),
});

export const selectEventsTool = createTool({
  name: 'select_events',
  description:
    "Select 1-6 event names from the provided list that are most relevant to the user's query.",
  parameters: SelectEventsParams as unknown as AnyZodType, // (ted): need to align zod version; version 3.25 does not support same types as 3.22
  handler: (args: unknown, { network }) => {
    const { events } = args as z.infer<typeof SelectEventsParams>;
    if (!Array.isArray(events) || events.length === 0) {
      return {
        selected: [],
        reason: 'No events selected.',
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

    return Mustache.render(systemPrompt, {
      totalEvents: events.length,
      hasEvents: sample.length > 0,
      eventsList: sample.join('\n'),
      maxEvents: 500,
    });
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
