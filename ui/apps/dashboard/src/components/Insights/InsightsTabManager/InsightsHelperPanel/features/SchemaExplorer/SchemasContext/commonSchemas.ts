import { STANDARD_EVENT_FIELDS } from '@inngest/components/constants';
import type { JSONSchema } from '@inngest/components/SchemaViewer/types';

export { STANDARD_EVENT_FIELDS };
export type StandardEventField = (typeof STANDARD_EVENT_FIELDS)[number];

// Schema definition for standard event fields
const STANDARD_FIELD_SCHEMAS: Record<StandardEventField, JSONSchema> = {
  data: { type: 'string' }, // technically a JSON object; other code will override the type to "JSON"
  id: { type: 'string' },
  name: { type: 'string' },
  received_at_dt: { type: 'string' },
  received_at: { type: 'number' },
  ts_dt: { type: 'string' },
  ts: { type: 'number' },
  v: { type: 'string' },
};

export const EVENT_SCHEMA_JSON: JSONSchema = {
  title: 'events',
  type: 'object',
  properties: STANDARD_FIELD_SCHEMAS,
};
