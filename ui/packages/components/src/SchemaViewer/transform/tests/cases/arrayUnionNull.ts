import type { TransformCase } from '../types';

// NOTE: We intentionally and arbitrarily choose the structural shape.
// In practice, we expect this not to happen given that we currently derive schemas from concrete events.
export const ARRAY_UNION_NULL_CASE: TransformCase = {
  name: 'should treat array|null as array for structure',
  schema: {
    title: 'numbers',
    type: ['array', 'null'],
    items: { type: 'integer' },
  },
  expected: {
    kind: 'array',
    name: 'numbers',
    path: 'numbers',
    element: {
      kind: 'value',
      name: '[*]',
      path: 'numbers[*]',
      type: 'integer',
    },
  },
};
