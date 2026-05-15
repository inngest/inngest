import type { TransformCase } from '../types';

export const OBJECT_NESTING_CASE: TransformCase = {
  name: 'should transform minimal object nesting',
  schema: {
    title: 'wrapper',
    type: 'object',
    properties: {
      event: {
        title: 'event',
        type: 'object',
        properties: {
          id: {
            type: 'string',
          },
        },
      },
    },
  },
  expected: {
    kind: 'object',
    name: 'wrapper',
    path: 'wrapper',
    children: [
      {
        kind: 'object',
        name: 'event',
        path: 'wrapper.event',
        children: [
          {
            kind: 'value',
            name: 'id',
            path: 'wrapper.event.id',
            type: 'string',
          },
        ],
      },
    ],
  },
};
