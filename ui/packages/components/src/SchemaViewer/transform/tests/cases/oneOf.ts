import type { TransformCase } from '../types';

export const ONE_OF_STRUCTURAL_OBJECT_FIRST_CASE: TransformCase = {
  name: 'oneOf: picks first structural when multiple (same)',
  schema: {
    title: 'payload',
    oneOf: [
      {
        type: 'object',
        properties: {
          shape: { type: 'string' },
          foo: { type: 'number' },
        },
      },
      {
        type: 'object',
        properties: {
          shape: { type: 'string' },
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
      { kind: 'value', name: 'shape', path: 'payload.shape', type: 'string' },
      { kind: 'value', name: 'foo', path: 'payload.foo', type: 'number' },
    ],
  },
};

export const ONE_OF_STRUCTURAL_ARRAY_FIRST_CASE: TransformCase = {
  name: 'oneOf: picks first structural when multiple (mixed)',
  schema: {
    title: 'payload',
    oneOf: [
      { type: 'array', items: { type: 'string' } },
      {
        type: 'object',
        properties: { id: { type: 'string' } },
      },
    ],
  },
  expected: {
    kind: 'array',
    name: 'payload',
    path: 'payload',
    element: { kind: 'value', name: '[*]', path: 'payload[*]', type: 'string' },
  },
};

export const ONE_OF_SCALAR_FIRST_CASE: TransformCase = {
  name: 'oneOf: picks first scalar when no structures',
  schema: {
    title: 'amount',
    oneOf: [{ type: 'null' }, { type: 'number' }, { type: 'integer' }],
  },
  expected: {
    kind: 'value',
    name: 'amount',
    path: 'amount',
    type: 'number',
  },
};

export const ONE_OF_FALLBACK_NULL_BOOLEAN_CASE: TransformCase = {
  name: 'oneOf: falls back to first null when otherwise only boolean shortcuts',
  schema: {
    title: 'maybeNull',
    oneOf: [true, false, { type: 'null' }],
  },
  expected: {
    kind: 'value',
    name: 'maybeNull',
    path: 'maybeNull',
    type: 'null',
  },
};

export const ONE_OF_BOOLEAN_SCHEMA_FALLBACK_UNKNOWN_CASE: TransformCase = {
  name: 'oneOf: boolean schema (true/false) falls back to unknown',
  schema: {
    title: 'maybeAnything',
    oneOf: [true, false],
  },
  expected: {
    kind: 'value',
    name: 'maybeAnything',
    path: 'maybeAnything',
    type: 'unknown',
  },
};
