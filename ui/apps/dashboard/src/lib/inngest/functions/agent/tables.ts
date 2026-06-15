import type { SchemaNode } from '@inngest/components/SchemaViewer/types';

import { tableEntries } from '@/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/tableSchemas';

export interface TableColumn {
  name: string;
  type: string;
}
export interface TableMeta {
  name: string;
  purpose: string;
  dynamic: boolean;
  columns: TableColumn[];
}

const PURPOSE: Record<string, string> = {
  events:
    'Raw event stream — one row per event ingested. Use for questions about events sent/received, event payload fields (data.*), event volumes, and triggers.',
  runs: 'Function executions — one row per run. Use for questions about function invocations, run status (Queued/Running/Failed/Cancelled/Completed), durations, inputs/outputs, and function-level failures.',
  steps:
    'Latest attempt of each step within runs. Use for questions about individual step execution, step status, step types (StepRun/Sleep/AIGateway/...), and step-level failures.',
  step_attempts:
    'Every step attempt including retries (same schema as steps). Use when the user asks about retries or per-attempt behavior.',
  extended_trace_spans:
    'OpenTelemetry spans for runs/steps. Use for low-level tracing and span timing questions, and for SCORE data: scores (from inngest.score / step.score) are recorded as span attributes/metadata — there is no dedicated scores table — so query this table and read the score value out of the `attributes`/`metadata` columns. Experiment results are NOT queryable here.',
};

export const PRIMARY_TABLES = ['events', 'runs', 'steps'];

function columnType(node: SchemaNode): string {
  return 'type' in node && typeof node.type === 'string'
    ? node.type
    : node.kind;
}

export const TABLES: TableMeta[] = tableEntries.map((entry) => {
  const children = 'children' in entry.node ? entry.node.children : [];
  return {
    name: entry.node.name,
    purpose:
      PURPOSE[entry.node.name] ?? `Queryable table \`${entry.node.name}\`.`,
    dynamic: entry.node.name === 'events',
    columns: children.map((c) => ({ name: c.name, type: columnType(c) })),
  };
});

export function getTable(name: string): TableMeta | undefined {
  return TABLES.find((t) => t.name === name);
}

export function listDataSources(): string {
  return TABLES.map(
    (t) =>
      `- ${t.name}: ${t.purpose}${
        t.dynamic
          ? ' (event data fields are dynamic — use find_events / get_event_schemas)'
          : ''
      }`,
  ).join('\n');
}

export function describeTable(name: string): string {
  const t = getTable(name);
  if (!t) {
    return `Unknown table "${name}". Available tables: ${TABLES.map(
      (x) => x.name,
    ).join(', ')}.`;
  }
  const cols = t.columns.map((c) => `- ${c.name} (${c.type})`).join('\n');
  const dynamicHint = t.dynamic
    ? '\n\nThe `data` column is event-specific JSON. Use find_events to discover event names and get_event_schemas to see their data fields before referencing data.* paths.'
    : '';
  return `Table \`${t.name}\`: ${t.purpose}\n\nColumns:\n${cols}${dynamicHint}`;
}
