import Mustache from 'mustache';
import { z } from 'zod';

import systemPrompt from './system.md?raw';

// Zod schema for the select_events tool (structured output extraction)
export const SelectEventsParams = z.object({
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

// Anthropic tool definition for step.ai.infer()
export const selectEventsTool = {
  name: 'select_events' as const,
  description:
    "Select 0-6 event names from the provided list that are most relevant to the user's query. Return an empty array when no specific events should be filtered.",
  input_schema: z.toJSONSchema(SelectEventsParams) as {
    type: 'object';
    [k: string]: unknown;
  },
};

/**
 * Build the event matcher system prompt by hydrating the Mustache template
 * with the available event types and optional current query.
 */
export function buildSystemPrompt(params: {
  eventTypes: string[];
  currentQuery?: string;
}): string {
  const events = params.eventTypes || [];
  const sample = events.slice(0, 500);

  const promptContext = {
    totalEvents: events.length,
    hasEvents: sample.length > 0,
    eventsList: sample.join('\n'),
    maxEvents: 500,
    hasCurrentQuery: !!params.currentQuery,
    currentQuery: params.currentQuery || '',
  };

  return Mustache.render(systemPrompt, promptContext);
}

export type SelectEventsResult = {
  selectedEvents: { event_name: string; reason: string }[];
  selectionReason: string;
  totalCandidates: number;
};

/**
 * Parse the Anthropic Messages API response to extract the select_events
 * tool call result.
 */
export function parseToolResult(
  result: {
    content: Array<{
      type: string;
      name?: string;
      input?: unknown;
    }>;
  },
  totalCandidates: number,
): SelectEventsResult {
  const toolUse = result.content.find(
    (block) => block.type === 'tool_use' && block.name === 'select_events',
  );

  if (!toolUse || !('input' in toolUse)) {
    return {
      selectedEvents: [],
      selectionReason: 'No tool call found in response.',
      totalCandidates,
    };
  }

  const input = toolUse.input as z.infer<typeof SelectEventsParams>;
  const events = input.events || [];

  return {
    selectedEvents: events,
    selectionReason:
      events.length === 0
        ? 'No specific events selected - query will include all events.'
        : "Selected by the LLM based on the user's query.",
    totalCandidates,
  };
}
