import { transformJSONSchema } from '@inngest/components/SchemaViewer/transform/transform';
import type { JSONSchema, SchemaNode } from '@inngest/components/SchemaViewer/types';

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

export const EVENT_SCHEMA_TREE: SchemaNode = transformJSONSchema(EVENT_SCHEMA_JSON);
