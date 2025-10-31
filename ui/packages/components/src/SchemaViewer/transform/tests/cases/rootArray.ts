import type { TransformCase } from '../types';

/*
  Asserts: A root array schema transforms to a top-level array node with a
  single [*] value variant; baseline for array handling.
*/
export const ROOT_ARRAY_CASE: TransformCase = {
  name: 'should transform root arrays',
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
