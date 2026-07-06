import Mustache from 'mustache';

import systemPrompt from './system.md?raw';

export function buildSystemPrompt(params: { currentQuery?: string }): string {
  return Mustache.render(systemPrompt, {
    hasCurrentQuery: !!params.currentQuery,
    currentQuery: params.currentQuery || '',
  });
}
