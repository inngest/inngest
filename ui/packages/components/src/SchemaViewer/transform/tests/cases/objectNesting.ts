import type { TransformCase } from '../types';

/*
  Asserts: Nested objects become nested object nodes with correct names and
  dot-paths, preserving hierarchy depth in the transformed tree.
*/
export const OBJECT_NESTING_CASE: TransformCase = {
  name: 'should transform minimal object nesting',
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
};
