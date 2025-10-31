import type { TransformCase } from '../types';

/*
  Asserts: Mixed structural unions (object|array) should not arise when deriving
  from a single concrete instance; you only get one shape or a tuple (positional).

  If encountered (authored schemas or merged specimens), we deliberately collapse
  to a value node with type 'unknown' to avoid arbitrarily picking a structure
  and misleading the UI. Deterministic and honest over guessing.
*/
export const MIXED_STRUCTURAL_UNION_CASE: TransformCase = {
  name: 'should collapse object|array union to value unknown',
  schema: {
    title: 'mixed',
    type: ['object', 'array'],
  },
  expected: {
    kind: 'value',
    name: 'mixed',
    path: 'mixed',
    type: 'unknown',
  },
};
