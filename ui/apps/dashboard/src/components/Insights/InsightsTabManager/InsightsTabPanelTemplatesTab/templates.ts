import type { QueryTemplate } from '@/components/Insights/types';

// TODO: Update templates.

export const TEMPLATES: QueryTemplate[] = [
  {
    id: 'event-volume-trends',
    name: 'Event volume trends',
    query: `<Query text from "Event volume trends">`,
    explanation: 'Track hourly event volume by type',
    templateKind: 'time',
  },
  {
    id: 'event-frequency-analysis',
    name: 'Event frequency analysis',
    query: `<Query text from "Event frequency analysis">`,
    explanation: 'Examine frequency patterns over time',
    templateKind: 'time',
  },
  {
    id: 'recent-event-errors',
    name: 'Recent event errors',
    query: `<Query text from "Recent event errors">`,
    explanation: 'Find events with errors from the last day',
    templateKind: 'error',
  },
  {
    id: 'event-error-patterns',
    name: 'Event error patterns',
    query: `<Query text from "Event error patterns">`,
    explanation: 'Calculate error rates by event type',
    templateKind: 'error',
  },
  {
    id: 'large-event-payloads',
    name: 'Large event payloads',
    query: `<Query text from "Large event payloads">`,
    explanation: 'Identify unusually large event payloads',
    templateKind: 'warning',
  },
];
