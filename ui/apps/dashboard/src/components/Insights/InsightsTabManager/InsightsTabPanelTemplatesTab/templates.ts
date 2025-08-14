import type { Template } from '../../QueryHelperPanel/types';

// TODO: Update templates.

export const TEMPLATES: Template[] = [
  {
    id: 'event-volume-trends',
    isSavedQuery: false,
    name: 'Event volume trends',
    query: `<Query text from "Event volume trends">`,
    explanation: 'Track hourly event volume by type',
    templateKind: 'time',
    type: 'template',
  },
  {
    id: 'event-frequency-analysis',
    isSavedQuery: false,
    name: 'Event frequency analysis',
    query: `<Query text from "Event frequency analysis">`,
    explanation: 'Examine frequency patterns over time',
    templateKind: 'time',
    type: 'template',
  },
  {
    id: 'recent-event-errors',
    isSavedQuery: false,
    name: 'Recent event errors',
    query: `<Query text from "Recent event errors">`,
    explanation: 'Find events with errors from the last day',
    templateKind: 'error',
    type: 'template',
  },
  {
    id: 'event-error-patterns',
    isSavedQuery: false,
    name: 'Event error patterns',
    query: `<Query text from "Event error patterns">`,
    explanation: 'Calculate error rates by event type',
    templateKind: 'error',
    type: 'template',
  },
  {
    id: 'large-event-payloads',
    isSavedQuery: false,
    name: 'Large event payloads',
    query: `<Query text from "Large event payloads">`,
    explanation: 'Identify unusually large event payloads',
    templateKind: 'warning',
    type: 'template',
  },
  {
    id: 'suspicious-event-patterns',
    isSavedQuery: false,
    name: 'Suspicious event patterns',
    query: `<Query text from "Suspicious event patterns">`,
    explanation: 'Detect abnormally high event rates',
    templateKind: 'warning',
    type: 'template',
  },
];
