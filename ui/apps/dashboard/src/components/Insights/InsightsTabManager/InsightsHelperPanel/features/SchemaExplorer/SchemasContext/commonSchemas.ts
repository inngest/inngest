import type { JSONSchema } from '@inngest/components/SchemaViewer/types';

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
