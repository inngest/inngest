import type { TransformCase } from '../types';

export const BASIC_EVENT_CASE: TransformCase = {
  name: 'should transform the basic event schema',
  schema: {
    title: 'event',
    type: 'object',
    properties: {
      // Skip "data" for now given its variability.
      id: {
        type: 'string',
        description: 'Unique identifier for the event',
      },
      name: {
        type: 'string',
        description: 'The name/type of the event',
      },
      ts: {
        type: 'number',
        description: 'Unix timestamp in milliseconds when the event occurred',
      },
      v: {
        type: 'string',
        description: 'Event format version',
      },
    },
  },
  expected: {
    kind: 'object',
    children: [
      {
        kind: 'value',
        name: 'id',
        path: 'event.id',
        type: 'string',
      },
      {
        kind: 'value',
        name: 'name',
        path: 'event.name',
        type: 'string',
      },
      {
        kind: 'value',
        name: 'ts',
        path: 'event.ts',
        type: 'number',
      },
      {
        kind: 'value',
        name: 'v',
        path: 'event.v',
        type: 'string',
      },
    ],
    name: 'event',
    path: 'event',
  },
};
