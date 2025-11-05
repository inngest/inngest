import { transformJSONSchema } from '@inngest/components/SchemaViewer/transform/transform';

import { EVENT_SCHEMA_JSON } from './commonSchemas';
import type { SchemaEntry } from './types';

export function buildSchemaEntriesFromQueryData(): SchemaEntry[] {
  const list: SchemaEntry[] = [];

  list.push({
    key: 'common:event',
    displayName: 'event',
    isShared: true,
    node: transformJSONSchema(EVENT_SCHEMA_JSON),
  });

  // TODO: Add entries for fetched schemas.

  return list;
}
