import { transformJSONSchema } from "@inngest/components/SchemaViewer/transform/transform";
import type { JSONSchema } from "@inngest/components/SchemaViewer/types";
import type { InfiniteData } from "@tanstack/react-query";

import { EVENT_SCHEMA_JSON } from "./commonSchemas";
import type { SchemaEntry, SchemaEventPage } from "./types";
import { makeTitleOnlyEntry } from "./utils";

export function buildSchemaEntriesFromQueryData(
  data: InfiniteData<SchemaEventPage> | undefined,
): SchemaEntry[] {
  const list: SchemaEntry[] = [];

  list.push({
    key: "common:events",
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

    const dataSchema = extractDataProperty(parsed);
    if (dataSchema === null) return null;

    return {
      key: `fetched:${eventName}`,
      isShared: false,
      node: transformJSONSchema({ ...dataSchema, title: eventName }),
    };
  } catch {
    return null;
  }
}

export function safeParseJSONSchema(
  input: string | undefined | null,
): JSONSchema | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    if (!obj || typeof obj !== "object") return null;

    // TODO: Consider validating that `obj` conforms to JSONSchema before casting.
    return obj as JSONSchema;
  } catch {
    return null;
  }
}

export function extractDataProperty(schema: JSONSchema): JSONSchema | null {
  const dataDefinition = schema.properties?.data;
  if (!dataDefinition || typeof dataDefinition === "boolean") return null;

  return dataDefinition;
}
