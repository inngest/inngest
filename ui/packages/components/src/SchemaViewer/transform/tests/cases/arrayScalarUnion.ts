import type { TransformCase } from '../types';

/*
  Asserts: For a union between a structural type and a scalar (array|string),
  the transform chooses the structural shape for tree rendering. The resulting
  node is an array with its element variant(s); scalar union info is not shown
  at the structural level.
*/
export const ARRAY_SCALAR_UNION_CASE: TransformCase = {
  name: 'should prefer array over scalar in unions',
  schema: {
    title: 'tags',
    type: ['string', 'array'],
    items: { type: 'string' },
  },
  expected: {
    kind: 'array',
    name: 'tags',
    path: 'tags',
    elementVariants: [
      {
        kind: 'value',
        name: '[*]',
        path: 'tags[*]',
        type: 'string',
      },
    ],
    various: false,
  },
};
