import type { TransformCase } from '../types';

// NOTE: We intentionally select the first structural type encountered.
// In practice, we expect this not to happen given that we currently derive schemas from concrete events.
export const MIXED_STRUCTURAL_UNION_CASE: TransformCase = {
  name: 'should select first structural type (object) in union',
  schema: {
    title: 'mixed',
    type: ['object', 'array'],
  },
  expected: {
    kind: 'object',
    name: 'mixed',
    path: 'mixed',
    children: [],
  },
};
