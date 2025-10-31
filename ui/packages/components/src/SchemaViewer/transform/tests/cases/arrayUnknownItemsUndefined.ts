import type { TransformCase } from '../types';

/*
Asserts: When items is undefined, the array renders a single '[*]' element of
type 'unknown'.
*/
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
