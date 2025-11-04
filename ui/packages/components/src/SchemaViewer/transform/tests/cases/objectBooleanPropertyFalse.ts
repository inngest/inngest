import type { TransformCase } from '../types';

export const OBJECT_BOOLEAN_PROPERTY_FALSE_CASE: TransformCase = {
  name: 'should omit properties with boolean schema false',
  schema: {
    title: 'obj',
    type: 'object',
    properties: {
      x: false,
      y: { type: 'string' },
    },
  },
  expected: {
    kind: 'object',
    name: 'obj',
    path: 'obj',
    children: [
      {
        kind: 'value',
        name: 'y',
        path: 'obj.y',
        type: 'string',
      },
    ],
  },
};
