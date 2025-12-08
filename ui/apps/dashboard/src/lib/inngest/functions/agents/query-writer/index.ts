import {
  anthropic,
  createAgent,
  createTool,
  type AnyZodType,
} from "@inngest/agent-kit";
import { z } from "zod";

import type { InsightsAgentState } from "../types";
import systemPrompt from "./system.md";

const GenerateSqlParams = z.object({
  sql: z
    .string()
    .min(1)
    .describe(
      "A single valid SELECT statement. Do not include DDL/DML or multiple statements.",
    ),
  title: z
    .string()
    .min(1)
    .describe("Short 20-30 character title for this query"),
  reasoning: z
    .string()
    .min(1)
    .describe(
      "Brief 1-2 sentence explanation of how this query addresses the request",
    ),
});

export const generateSqlTool = createTool({
  name: "generate_sql",
  description:
    "Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.",
  parameters: GenerateSqlParams as unknown as AnyZodType, // (ted): need to update to latest version of zod + agent-kit
  handler: ({ sql, title, reasoning }: z.infer<typeof GenerateSqlParams>) => {
    return {
      sql,
      title,
      reasoning,
    };
  },
});

export const queryWriterAgent = createAgent<InsightsAgentState>({
  name: "Insights Query Writer",
  description:
    "Generates a safe, read-only SQL SELECT statement for ClickHouse.",
  system: async ({ network }) => {
    const selected =
      network?.state.data.selectedEvents?.map(
        (e: { event_name: string }) => e.event_name,
      ) ?? [];
    return [
      systemPrompt,
      "",
      selected.length
        ? `Target the following events if relevant: ${selected.join(", ")}`
        : "If events were selected earlier, incorporate them appropriately.",
      "",
      "When ready, call the generate_sql tool with the final SQL and a short 20-30 character title.",
    ].join("\n");
  },
  model: anthropic({
    model: "claude-sonnet-4-5-20250929",
    defaultParameters: {
      max_tokens: 4096,
    },
  }),
  tools: [generateSqlTool],
  tool_choice: "generate_sql",
});
