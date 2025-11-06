'use client';

import { transformJSONSchema } from '@inngest/components/SchemaViewer/transform/transform';
import type { JSONSchema, SchemaNode } from '@inngest/components/SchemaViewer/types';

export const EVENT_SCHEMA_JSON: JSONSchema = {
  title: 'event',
  type: 'object',
  properties: {
    data: { type: 'string' },
    id: { type: 'string' },
    name: { type: 'string' },
    ts: { type: 'number' },
    v: { type: 'string' },
  },
};

type UseSchemasReturn = {
  schemas: SchemaNode[];
};

// TODO: Fetch all schemas.
export function useSchemas(): UseSchemasReturn {
  return { schemas: [transformJSONSchema(EVENT_SCHEMA_JSON)] };
}
