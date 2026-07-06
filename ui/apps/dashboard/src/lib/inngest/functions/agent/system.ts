import Mustache from 'mustache';

import type { Lesson } from './lessons';
import systemPrompt from './system.md?raw';

export function buildSystemPrompt(params: {
  currentQuery?: string;
  lessons: Lesson[];
}): string {
  return Mustache.render(systemPrompt, {
    hasCurrentQuery: !!params.currentQuery,
    currentQuery: params.currentQuery || '',
    hasLessons: params.lessons.length > 0,
    lessons: params.lessons,
  });
}
