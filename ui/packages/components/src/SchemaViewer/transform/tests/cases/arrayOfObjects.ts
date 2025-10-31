import type { TransformCase } from '../types';

/*
  Asserts: A root array of objects renders as an array node with a single
  object element variant ([*]); the object's properties appear under that
  variant with correct paths.
*/
export const ARRAY_OF_OBJECTS_CASE: TransformCase = {
  name: 'should transform arrays of objects',
  schema: {
    title: 'items',
    type: 'array',
    items: {
      type: 'object',
      properties: {
        id: { type: 'string' },
        qty: { type: 'integer' },
      },
    },
  },
  expected: {
    kind: 'array',
    name: 'items',
    path: 'items',
    elementVariants: [
      {
        kind: 'object',
        name: '[*]',
        path: 'items[*]',
        children: [
          {
            kind: 'value',
            name: 'id',
            path: 'items[*].id',
            type: 'string',
          },
          {
            kind: 'value',
            name: 'qty',
            path: 'items[*].qty',
            type: 'integer',
          },
        ],
      },
    ],
    various: false,
  },
};
