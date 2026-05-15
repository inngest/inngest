import type { TransformCase } from '../types';

export const OBJECT_SCALAR_UNION_CASE: TransformCase = {
  name: 'should prefer object over scalar in unions',
  schema: {
    title: 'meta',
    type: ['string', 'object'],
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
