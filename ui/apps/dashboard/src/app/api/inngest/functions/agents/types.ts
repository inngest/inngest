// Shared TypeScript-only types for Insights agents and UI

export type InsightsToolName = 'select_events' | 'generate_sql';

export type InsightsState = {
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

// Tool I/O (no runtime validation; types only)
export type SelectEventsInput = {
  events: {
    event_name: string;
    reason: string;
  }[];
};

export type SelectEventsResult = {
  selected: {
    event_name: string;
    reason: string;
  }[];
  reason: string;
  totalCandidates: number;
};

export type GenerateSqlInput = {
  sql: string; // single SELECT statement
  title?: string;
  reasoning?: string;
};

export type GenerateSqlResult = {
  sql: string;
  title?: string;
  reasoning?: string;
};

// AgentKit streams tool results wrapped in an envelope
export type ToolEnvelope<T> = { data: T } | { error: unknown };
