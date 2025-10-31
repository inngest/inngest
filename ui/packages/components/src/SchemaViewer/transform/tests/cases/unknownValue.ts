import type { TransformCase } from '../types';

/*
  Asserts: When a value node lacks an explicit primitive type, the transform
  labels it as 'unknown' to avoid misleading assumptions.
*/
export const UNKNOWN_VALUE_CASE: TransformCase = {
  name: 'should mark scalars without type as unknown',
  schema: {
    title: 'mystery',
  },
  expected: {
    kind: 'value',
    name: 'mystery',
    path: 'mystery',
    type: 'unknown',
  },
};
