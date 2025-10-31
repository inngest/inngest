import type { TransformCase } from '../types';

/*
  Asserts: A root array of scalars renders as an array node with a single [*]
  value element variant; no tuple indices and various=false.
*/
export const ARRAY_SCALAR_CASE: TransformCase = {
  name: 'should transform arrays of scalars',
  schema: {
    title: 'tags',
    type: 'array',
    items: { type: 'string' },
  },
  expected: {
    kind: 'array',
    name: 'tags',
    path: 'tags',
    elementVariants: [
      {
        kind: 'value',
        name: '[*]',
        path: 'tags[*]',
        type: 'string',
      },
    ],
    various: false,
  },
};
