import type { TransformCase } from '../types';

export const OBJECT_BOOLEAN_PROPERTY_UNKNOWN_CASE: TransformCase = {
  name: 'should render true boolean properties as unknown',
  schema: {
    title: 'obj',
    type: 'object',
    properties: {
      x: true,
      y: { type: 'string' },
    },
  },
  expected: {
    kind: 'object',
    name: 'obj',
    path: 'obj',
    children: [
      {
        kind: 'value',
        name: 'x',
        path: 'obj.x',
        type: 'unknown',
      },
      {
        kind: 'value',
        name: 'y',
        path: 'obj.y',
        type: 'string',
      },
    ],
  },
};
