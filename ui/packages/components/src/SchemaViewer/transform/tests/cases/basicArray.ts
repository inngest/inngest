import type { TransformCase } from '../types';

export const BASIC_ARRAY_CASE: TransformCase = {
  name: 'should transform basic array',
  schema: {
    title: 'list',
    type: 'array',
    items: { type: 'integer' },
  },
  expected: {
    kind: 'array',
    name: 'list',
    path: 'list',
    element: {
      kind: 'value',
      name: '[*]',
      path: 'list[*]',
      type: 'integer',
    },
  },
};
