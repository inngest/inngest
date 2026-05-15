import type { TransformCase } from '../types';

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
