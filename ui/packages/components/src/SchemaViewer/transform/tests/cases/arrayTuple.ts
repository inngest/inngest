import type { TransformCase } from '../types';

/*
  Asserts: A tuple-based array renders as an array node with multiple value
  element variants ([0], [1], etc.); various=true.
*/
export const ARRAY_TUPLE_CASE: TransformCase = {
  name: 'should transform tuple-based arrays with multiple variants',
  schema: {
    title: 'pair',
    type: 'array',
    items: [{ type: 'string' }, { type: 'number' }],
  },
  expected: {
    kind: 'array',
    name: 'pair',
    path: 'pair',
    elementVariants: [
      {
        kind: 'value',
        name: '[0]',
        path: 'pair[0]',
        type: 'string',
      },
      {
        kind: 'value',
        name: '[1]',
        path: 'pair[1]',
        type: 'number',
      },
    ],
    various: true,
  },
};
