import type { TransformCase } from '../types';

/*
Asserts: When items is boolean, the array renders a single '[*]' element of
type 'unknown'.
*/
export const ARRAY_UNKNOWN_ITEMS_BOOLEAN_CASE: TransformCase = {
  name: 'should render unknown element when items is boolean',
  schema: {
    title: 'list',
    type: 'array',
    items: true,
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
