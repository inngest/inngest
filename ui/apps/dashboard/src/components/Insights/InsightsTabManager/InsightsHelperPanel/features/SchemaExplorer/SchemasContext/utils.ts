import { transformJSONSchema } from "@inngest/components/SchemaViewer/transform/transform";

import type { SchemaEntry } from "./types";

export function makeTitleOnlyEntry(name: string): SchemaEntry {
  return {
    key: `fetched:${name}`,
    isShared: false,
    node: transformJSONSchema({ title: name }),
  };
}
