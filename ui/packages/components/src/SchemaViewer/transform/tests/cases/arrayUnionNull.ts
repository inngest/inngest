import type { TransformCase } from '../types';

/*
  Asserts: For array|null unions, the transform chooses the array structure
  and renders its element variant.
*/
export const ARRAY_UNION_NULL_CASE: TransformCase = {
  name: 'should treat array|null as array for structure',
  schema: {
    title: 'numbers',
    type: ['array', 'null'],
    items: { type: 'integer' },
  },
  expected: {
    kind: 'array',
    name: 'numbers',
    path: 'numbers',
    elementVariants: [
      {
        kind: 'value',
        name: '[*]',
        path: 'numbers[*]',
        type: 'integer',
      },
    ],
    various: false,
  },
};
