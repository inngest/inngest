import Mustache from 'mustache';

import systemPrompt from './system.md?raw';

/**
 * Build the summarizer system prompt by hydrating the Mustache template
 * with selected events, SQL, and user intent.
 */
export function buildSystemPrompt(params: {
  selectedEvents: { event_name: string; reason: string }[];
  sql?: string;
  userIntent?: string;
}): string {
  const events = params.selectedEvents.map((e) => e.event_name);

  const promptContext = {
    hasSelectedEvents: events.length > 0,
    selectedEvents: events.join(', '),
    hasSql: !!params.sql,
    generated_sql: params.sql || '',
    user_intent: params.userIntent || '',
  };

  return Mustache.render(systemPrompt, promptContext);
}

/**
 * Parse the Anthropic Messages API response to extract the summary text.
 */
export function parseResult(result: {
  content: Array<{ type: string; text?: string }>;
}): string {
  const textBlock = result.content.find((block) => block.type === 'text');
  return textBlock && 'text' in textBlock && typeof textBlock.text === 'string'
    ? textBlock.text
    : '';
}
