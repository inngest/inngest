import type { TransformCase } from '../types';

/*
  Asserts: For object|scalar unions, the transform prefers the object structure
  and renders its children; scalar branch is not shown structurally.
*/
export const OBJECT_SCALAR_UNION_CASE: TransformCase = {
  name: 'should prefer object over scalar in unions',
  schema: {
    title: 'meta',
    type: ['object', 'string'],
    properties: {
      version: { type: 'string' },
    },
  },
  expected: {
    kind: 'object',
    name: 'meta',
    path: 'meta',
    children: [
      {
        kind: 'value',
        name: 'version',
        path: 'meta.version',
        type: 'string',
      },
    ],
  },
};
