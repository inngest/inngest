import type { JSONSchema } from "@inngest/components/SchemaViewer/types";

export const EVENT_SCHEMA_JSON: JSONSchema = {
  title: "events",
  type: "object",
  properties: {
    // data is technically a JSON object; other code will override the type to "JSON"
    data: { type: "string" },
    id: { type: "string" },
    name: { type: "string" },
    ts: { type: "number" },
    ts_dt: { type: "string" },
    received_at: { type: "number" },
    received_at_dt: { type: "string" },
    v: { type: "string" },
  },
};
