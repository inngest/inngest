import type { TransformCase } from '../types';

/*
  Asserts: Pure scalar unions (e.g., integer|null) are preserved on value
  nodes.
*/
export const UNION_SCALAR_CASE: TransformCase = {
  name: 'should transform union scalar types',
  schema: {
    title: 'maybeCount',
    type: ['integer', 'null'],
  },
  expected: {
    kind: 'value',
    name: 'maybeCount',
    path: 'maybeCount',
    type: ['integer', 'null'],
  },
};
