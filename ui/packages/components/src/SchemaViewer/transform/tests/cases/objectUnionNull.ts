import type { TransformCase } from '../types';

/*
  Asserts: For object|null unions, the transform chooses the object structure
  and renders its children; it does not branch or indicate nullability.
*/
export const OBJECT_UNION_NULL_CASE: TransformCase = {
  name: 'should treat object|null as object for structure',
  schema: {
    title: 'meta',
    type: ['object', 'null'],
    properties: {
      version: { type: 'string' },
    },
  },
  expected: {
    kind: 'object',
    name: 'meta',
    path: 'meta',
    children: [
      {
        kind: 'value',
        name: 'version',
        path: 'meta.version',
        type: 'string',
      },
    ],
  },
};
