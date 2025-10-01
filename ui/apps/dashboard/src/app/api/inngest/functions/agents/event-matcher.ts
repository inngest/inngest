import { createAgent, createTool, openai, type AnyZodType } from '@inngest/agent-kit';
import { z } from 'zod';

import type { InsightsAgentState as InsightsState, SelectEventsResult } from './types';

const SelectEventsParams = z.object({
  events: z
    .array(
      z.object({
        event_name: z.string(),
        reason: z.string(),
      })
    )
    .min(1)
    .max(6)
    .describe(
      "An array of 1-6 event names selected from the list of available events that best match the user's intent."
    ),
});

export const selectEventsTool = createTool({
  name: 'select_events',
  description:
    "Select 1-6 event names from the provided list that are most relevant to the user's query.",
  parameters: SelectEventsParams as unknown as AnyZodType, // (ted): need to align zod version; version 3.25 does not support same types as 3.22
  handler: (args: z.infer<typeof SelectEventsParams>, { network }) => {
    const { events } = args;
    if (!Array.isArray(events) || events.length === 0) {
      throw new Error('The model must select at least one event.');
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
    } as SelectEventsResult;
    return result;
  },
});

export const eventMatcherAgent = createAgent<InsightsState>({
  name: 'Insights Event Matcher',
  description: "Analyzes available events and selects 1-5 that best match the user's intent.",
  system: async ({ network }): Promise<string> => {
    const events = network?.state.data.eventTypes || [];
    const sample = events.slice(0, 500); // avoid overly long prompts
    return [
      'You are an event selection specialist.',
      "Your job is to analyze the user's request and the list of available event names, then choose the 1-5 most relevant events.",
      '',
      'Instructions:',
      '- Review the list of available events provided below.',
      "- Based on the user's query, decide which 1-5 events are the best match.",
      '- Call the `select_events` tool and pass your final choice in the `events` parameter.',
      '- Do not guess event names; only use names from the provided list.',
      '',
      sample.length
        ? `Available events (${events.length} total, showing up to 500):\n${sample.join('\n')}`
        : 'No event list is available. Ask the user to clarify which events they are interested in.',
    ].join('\n');
  },
  model: openai({ model: 'gpt-4.1-2025-04-14' }),
  tools: [selectEventsTool],
  tool_choice: 'select_events',
});

export type { InsightsState as InsightsAgentState };
