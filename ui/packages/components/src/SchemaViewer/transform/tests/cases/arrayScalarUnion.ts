import type { TransformCase } from '../types';

// NOTE: We intentionally and arbitrarily choose the structural shape.
// In practice, we expect this not to happen given that we currently derive schemas from concrete events.
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
    element: {
      kind: 'value',
      name: '[*]',
      path: 'tags[*]',
      type: 'string',
    },
  },
};
