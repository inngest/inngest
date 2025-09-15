import { createAgent, createTool, openai, type Network, type StateData } from '@inngest/agent-kit';
import { z } from 'zod';

import type {
  SelectEventsInput,
  SelectEventsResult,
  InsightsState as SharedInsightsState,
} from './types';

export interface InsightsState extends StateData, SharedInsightsState {}

const selectEventsTool = createTool({
  name: 'select_events',
  description:
    "Select 1-5 event names from the provided list that are most relevant to the user's query.",
  parameters: z
    .object({
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
    })
    .strict() as any,
  handler: (args: SelectEventsInput, ctx): SelectEventsResult => {
    const network = ctx?.network as Network<InsightsState>;
    const selected = args.events;

    if (!Array.isArray(selected) || selected.length === 0) {
      throw new Error('The model must select at least one event.');
    }

    const reason = "Selected by the LLM based on the user's query.";

    // Persist selection on network state for downstream agents
    network.state.data.selectedEvents = selected;
    network.state.data.selectionReason = reason;

    const result: SelectEventsResult = {
      selected,
      reason,
      totalCandidates: network?.state?.data?.eventTypes?.length || 0,
    };
    return result;
  },
});

export const eventMatcherAgent = createAgent<InsightsState>({
  name: 'Insights Event Matcher',
  description: "Analyzes available events and selects 1-5 that best match the user's intent.",
  system: async ({ network }): Promise<string> => {
    const events = network?.state?.data?.eventTypes || [];
    const sample = events.slice(0, 50); // avoid overly long prompts
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
        ? `Available events (${events.length} total, showing up to 50):\n${sample.join('\n')}`
        : 'No event list is available. Ask the user to clarify which events they are interested in.',
    ].join('\n');
  },
  model: openai({ model: 'gpt-5-nano-2025-08-07' }),
  tools: [selectEventsTool],
  tool_choice: 'select_events',
});

export type { InsightsState as InsightsAgentState };
