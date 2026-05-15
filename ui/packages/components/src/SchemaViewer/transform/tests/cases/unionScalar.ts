import type { TransformCase } from '../types';

export const UNION_SCALAR_CASE: TransformCase = {
  name: 'should transform union scalar types',
  schema: {
    title: 'maybeCount',
    type: ['integer', 'null', 'string'],
  },
  expected: {
    kind: 'value',
    name: 'maybeCount',
    path: 'maybeCount',
    type: ['integer', 'null', 'string'],
  },
};
