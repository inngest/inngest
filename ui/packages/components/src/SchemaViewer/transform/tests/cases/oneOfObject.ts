import type { TransformCase } from '../types';

/*
  Asserts: oneOf with multiple object branches resolves deterministically to
  the first concrete branch; other branches are ignored for rendering.
*/
export const ONE_OF_OBJECT_CASE: TransformCase = {
  name: 'should resolve oneOf by choosing the first concrete schema',
  schema: {
    title: 'payload',
    oneOf: [
      {
        type: 'object',
        properties: {
          type: { type: 'string' },
          foo: { type: 'number' },
        },
      },
      {
        type: 'object',
        properties: {
          type: { type: 'string' },
          bar: { type: 'boolean' },
        },
      },
    ],
  },
  expected: {
    kind: 'object',
    name: 'payload',
    path: 'payload',
    children: [
      {
        kind: 'value',
        name: 'type',
        path: 'payload.type',
        type: 'string',
      },
      {
        kind: 'value',
        name: 'foo',
        path: 'payload.foo',
        type: 'number',
      },
    ],
  },
};
