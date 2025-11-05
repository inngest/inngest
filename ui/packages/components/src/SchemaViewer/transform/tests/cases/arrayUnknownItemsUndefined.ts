import type { TransformCase } from '../types';

export const ARRAY_UNKNOWN_ITEMS_UNDEFINED_CASE: TransformCase = {
  name: 'should render unknown element when items is undefined',
  schema: {
    title: 'list',
    type: 'array',
  },
  expected: {
    kind: 'array',
    name: 'list',
    path: 'list',
    element: {
      kind: 'value',
      name: '[*]',
      path: 'list[*]',
      type: 'unknown',
    },
  },
};
