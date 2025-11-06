import type { TransformCase } from '../types';

export const BASIC_OBJECT_CASE: TransformCase = {
  name: 'should transform basic object',
  schema: {
    title: 'obj',
    type: 'object',
    properties: {
      id: { type: 'integer' },
    },
  },
  expected: {
    kind: 'object',
    name: 'obj',
    path: 'obj',
    children: [
      {
        kind: 'value',
        name: 'id',
        path: 'obj.id',
        type: 'integer',
      },
    ],
  },
};
