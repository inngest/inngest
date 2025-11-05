import type { TransformCase } from '../types';

export const OBJECT_UNION_NULL_CASE: TransformCase = {
  name: 'should treat object|null as object for structure',
  schema: {
    title: 'meta',
    type: ['null', 'object'],
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
