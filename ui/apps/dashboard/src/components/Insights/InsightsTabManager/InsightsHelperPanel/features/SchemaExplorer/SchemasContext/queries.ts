import { transformJSONSchema } from '@inngest/components/SchemaViewer/transform/transform';
import type { JSONSchema } from '@inngest/components/SchemaViewer/types';
import type { InfiniteData } from '@tanstack/react-query';

import { EVENT_SCHEMA_JSON, STANDARD_EVENT_FIELDS } from './commonSchemas';
import type { SchemaEntry, SchemaEventPage } from './types';
import { makeTitleOnlyEntry } from './utils';

export function buildSchemaEntriesFromQueryData(
  data: InfiniteData<SchemaEventPage> | undefined,
): SchemaEntry[] {
  const list: SchemaEntry[] = [];

  list.push({
    key: 'common:events',
    isShared: true,
    node: transformJSONSchema(EVENT_SCHEMA_JSON),
  });

  const pages = data?.pages ?? [];
  const items = pages.flatMap((p) => p.events);
  for (const evt of items) {
    const entry = buildEntryFromLatestSchema(evt.schema, evt.name);
    if (entry === null) {
      list.push(makeTitleOnlyEntry(evt.name));
    } else {
      list.push(entry);
    }
  }

  return list;
}

export function buildEntryFromLatestSchema(
  latestSchema: string | undefined | null,
  eventName: string,
): SchemaEntry | null {
  try {
    const parsed = safeParseJSONSchema(latestSchema);
    if (parsed === null) return null;

    // Normalize the schema to only include standard event fields
    const normalizedSchema = normalizeEventSchema(parsed);

    return {
      key: `fetched:${eventName}`,
      isShared: false,
      node: transformJSONSchema({ ...normalizedSchema, title: eventName }),
    };
  } catch {
    return null;
  }
}

/**
 * Normalizes an event schema to only include the standard event fields.
 * This ensures all events show the same structure with only standard fields.
 * - Keeps existing standard fields from the schema
 * - Adds missing standard fields with default types
 * - Filters out any non-standard fields
 */
function normalizeEventSchema(schema: JSONSchema): JSONSchema {
  const normalizedProperties: Record<string, JSONSchema> = {};

  for (const field of STANDARD_EVENT_FIELDS) {
    const existingField = schema.properties?.[field];
    if (existingField && typeof existingField !== 'boolean') {
      // Keep the existing field definition
      normalizedProperties[field] = existingField;
    } else {
      // Add missing standard field with appropriate default type
      normalizedProperties[field] = getDefaultFieldSchema(field);
    }
  }

  return {
    ...schema,
    properties: normalizedProperties,
  };
}

/**
 * Returns the default schema for a standard event field
 */
function getDefaultFieldSchema(field: string): JSONSchema {
  switch (field) {
    case 'name':
    case 'id':
    case 'ts_dt':
    case 'received_at_dt':
    case 'v':
      return { type: 'string' };
    case 'ts':
    case 'received_at':
      return { type: 'number' };
    case 'data':
      return { type: 'object' };
    default:
      return { type: 'string' };
  }
}

export function safeParseJSONSchema(
  input: string | undefined | null,
): JSONSchema | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    if (!obj || typeof obj !== 'object') return null;

    // TODO: Consider validating that `obj` conforms to JSONSchema before casting.
    return obj as JSONSchema;
  } catch {
    return null;
  }
}

export function extractDataProperty(schema: JSONSchema): JSONSchema | null {
  const dataDefinition = schema.properties?.data;
  if (!dataDefinition || typeof dataDefinition === 'boolean') return null;

  return dataDefinition;
}
