import { transformJSONSchema } from '@inngest/components/TreeViewer/transform';
import type { JSONSchema, TreeNode } from '@inngest/components/TreeViewer/types';

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
  required: ['id', 'name', 'ts', 'v'],
};

export const EVENT_SCHEMA_TREE: TreeNode = transformJSONSchema(EVENT_SCHEMA_JSON);
