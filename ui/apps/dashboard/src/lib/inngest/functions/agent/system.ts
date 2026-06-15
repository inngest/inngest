import Mustache from 'mustache';

import type { InsightsClientState } from './loop';
import systemPrompt from './system.md?raw';
import { listDataSources } from './tables';

export function buildSystemPrompt(state: InsightsClientState): string {
  return Mustache.render(systemPrompt, {
    dataSources: listDataSources(),
    hasCurrentQuery: !!state.currentQuery,
    currentQuery: state.currentQuery ?? '',
  });
}
