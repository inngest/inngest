import type { TransformCase } from '../types';

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
