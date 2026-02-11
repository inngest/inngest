// Shared TypeScript-only types for Insights agents and UI
import { createToolManifest, type StateData } from '@inngest/agent-kit';

import { selectEventsTool } from './event-matcher';
import { generateSqlTool } from './query-writer';

// Consolidated into a single type that extends StateData as required by AgentKit
export type InsightsAgentState = StateData & {
  // Common conversation/user context
  userId?: string;
  query?: string; // The user's natural language query/request

  // Event catalog and schemas (UI-provided)
  eventTypes?: string[];
  schemas?: { name: string; schema: string }[];

  // Working selections and artifacts
  selectedEvents?: { event_name: string; reason: string }[];
  selectionReason?: string;
  currentQuery?: string;
  sql?: string;

  // Observability: Complete agent observability data in final format
  observability?: {
    eventMatcher?: {
      promptContext: {
        totalEvents: number;
        hasEvents: boolean;
        eventsList: string;
        maxEvents: number;
        hasCurrentQuery: boolean;
        currentQuery: string;
        currentQueryLength: number;
      };
      output?: {
        selectedEvents: { event_name: string; reason: string }[];
        selectionReason: string;
      };
    };
    queryWriter?: {
      promptContext: {
        selectedEventsCount: number;
        selectedEventNames: string[];
        schemasCount: number;
        schemaNames: string[];
        schemas: Array<{
          eventName: string;
          schema: string;
          schemaLength: number;
        }>;
        hasCurrentQuery: boolean;
        currentQueryLength: number;
      };
      output?: {
        sql: string;
        title?: string;
        reasoning?: string;
      };
    };
    summarizer?: {
      promptContext: {
        selectedEventsCount: number;
        selectedEventNames: string[];
        hasSql: boolean;
      };
      output?: string;
    };
  };
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
const manifest = createToolManifest([
  generateSqlTool,
  selectEventsTool,
] as const);

export type ToolManifest = typeof manifest;
