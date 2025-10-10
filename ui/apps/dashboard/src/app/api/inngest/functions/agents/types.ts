// Shared TypeScript-only types for Insights agents and UI
import { createToolManifest, type StateData } from '@inngest/agent-kit';

import { selectEventsTool } from './event-matcher';
import { generateSqlTool } from './query-writer';

// Consolidated into a single type that extends StateData as required by AgentKit
export type InsightsAgentState = StateData & {
  // Common conversation/user context
  userId?: string;

  // Event catalog and schemas (UI-provided)
  eventTypes?: string[];
  schemas?: Record<string, unknown>;

  // Working selections and artifacts
  selectedEvents?: { event_name: string; reason: string }[];
  selectionReason?: string;
  currentQuery?: string;
  sql?: string;
};

export type SelectEventsResult = {
  selected: {
    event_name: string;
    reason: string;
  }[];
  reason: string;
  totalCandidates: number;
};

export type GenerateSqlResult = {
  sql: string;
  title?: string;
  reasoning?: string;
};

// Build a strongly-typed tool manifest from tool definitions - used in the UI to render tool calls
const manifest = createToolManifest([generateSqlTool, selectEventsTool] as const);

export type ToolManifest = typeof manifest;
