import type { TransformCase } from '../types';

/*
Asserts: Boolean schemas within tuple items render as 'unknown' to preserve
tuple positions; multiple indexed variants are shown deterministically.
*/
export const ARRAY_TUPLE_WITH_BOOLEAN_UNKNOWN_CASE: TransformCase = {
  name: 'should render boolean tuple items as unknown',
  schema: {
    title: 'data',
    type: 'array',
    items: [true, { type: 'string' }],
  },
  expected: {
    kind: 'tuple',
    name: 'data',
    path: 'data',
    elements: [
      {
        kind: 'value',
        name: '[0]',
        path: 'data[0]',
        type: 'unknown',
      },
      {
        kind: 'value',
        name: '[1]',
        path: 'data[1]',
        type: 'string',
      },
    ],
  },
};
