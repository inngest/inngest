import type { TransformCase } from '../types';

/*
  Asserts: A tuple-based array renders as a tuple node with positional elements.
*/
export const ARRAY_TUPLE_CASE: TransformCase = {
  name: 'should transform tuple-based arrays with multiple variants',
  schema: {
    title: 'pair',
    type: 'array',
    items: [{ type: 'string' }, { type: 'number' }],
  },
  expected: {
    kind: 'tuple',
    name: 'pair',
    path: 'pair',
    elements: [
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
  },
};
