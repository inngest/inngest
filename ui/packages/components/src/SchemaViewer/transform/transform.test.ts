import { describe, expect, it } from 'vitest';

import type { JSONSchema, SchemaNode } from '../types';
import { transformJSONSchema } from './transform';

export const EVENT_SCHEMA_JSON: JSONSchema = {
  title: 'event',
  type: 'object',
  properties: {
    // Skip "data" for now given its variability.
    id: {
      type: 'string',
      description: 'Unique identifier for the event',
    },
    name: {
      type: 'string',
      description: 'The name/type of the event',
    },
    ts: {
      type: 'number',
      description: 'Unix timestamp in milliseconds when the event occurred',
    },
    v: {
      type: 'string',
      description: 'Event format version',
    },
  },
};

const EXPECTED_EVENT_SCHEMA_TREE: SchemaNode = {
  kind: 'object',
  children: [
    {
      kind: 'value',
      name: 'id',
      path: 'event.id',
      type: 'string',
    },
    {
      kind: 'value',
      name: 'name',
      path: 'event.name',
      type: 'string',
    },
    {
      kind: 'value',
      name: 'ts',
      path: 'event.ts',
      type: 'number',
    },
    {
      kind: 'value',
      name: 'v',
      path: 'event.v',
      type: 'string',
    },
  ],
  name: 'event',
  path: 'event',
};

describe('transformJSONSchema', () => {
  it('should transform the basic event schema', () => {
    const tree = transformJSONSchema(EVENT_SCHEMA_JSON);
    expect(tree).toEqual(EXPECTED_EVENT_SCHEMA_TREE);
  });

  it('should transform a schema with multiple layers of object nesting', () => {
    const tree = transformJSONSchema({
      title: 'wrapper',
      type: 'object',
      properties: {
        event: EVENT_SCHEMA_JSON,
      },
    });
    expect(tree).toEqual({
      kind: 'object',
      name: 'wrapper',
      path: 'wrapper',
      children: [
        {
          ...EXPECTED_EVENT_SCHEMA_TREE,
          children: EXPECTED_EVENT_SCHEMA_TREE.children.map((child) => ({
            ...child,
            path: `wrapper.${child.path}`,
          })),
          path: 'wrapper.event',
        },
      ],
    });
  });
});
