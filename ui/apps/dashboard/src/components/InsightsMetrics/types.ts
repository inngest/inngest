// Plain domain types for the insightsMetric result shapes, deliberately
// decoupled from the codegen'd query types — these are consumed by the
// reusable display components (HeadlineStats, TrendChart, RankedTable,
// CategoricalChart, BoxPlot), which are shared across the env-level AI
// Overview page and the function-level AI tab, and shouldn't be coupled to
// one specific query's generated shape. Composing components adapt the
// codegen'd query result into these before passing them down.
//
// insightsMetric returns a generic InsightsResponse (columns/rows), the
// same shape the free-form Insights query returns, rather than a
// shape-specific union — every insightsMetric column is untyped (DYNAMIC),
// so every cell arrives as a JSON string. The toScalarValues/toTrendPoints/
// toDimensionedTrendPoints/toListItems functions below reconstruct the
// scalar/time-series/list shapes these components expect, the same way
// pkg/applogic/dashboards/query.go's parse* functions used to on the
// backend before that parsing moved to the client.

export type NamedValue = {
  name: string;
  value: number;
};

export type InsightsMetricItem = {
  identifier: string;
  // Present only for list entries whose underlying query also selects
  // function_id (e.g. most-expensive-runs/-steps, which rank by run/step
  // but still belong to one function each) — the slug-translated function
  // identifier for that row, distinct from `identifier` itself.
  functionId?: string;
  // Present only for list entries whose underlying query also selects
  // session_key (e.g. most-expensive-sessions, whose `identifier` is a
  // session id — sessions are keyed, so the same id string under two
  // different keys is a different session; this disambiguates which).
  sessionKey?: string;
  values: NamedValue[];
};

export type InsightsMetricPoint = {
  timestamp: string;
  // Exactly one of these is populated per query — `dimensions` for a
  // per-category breakdown within this bucket (e.g. one entry per model),
  // `values` otherwise. See pkg/applogic/dashboards.Point.
  values?: NamedValue[];
  dimensions?: InsightsMetricItem[];
};

// valuesToMap turns a NamedValue[] into a lookup by name — every display
// component reads its configured value names out of one of these rather
// than re-scanning the array.
export function valuesToMap(values: NamedValue[]): Map<string, number> {
  const m = new Map<string, number>();
  for (const v of values) {
    m.set(v.name, v.value);
  }
  return m;
}

// MetricTable is the shape every insightsMetric field alias resolves to:
// an InsightsResponse (columns/rows/query), or null for an unrecognized
// key. Structural (not imported from the generated types) so it matches
// every aliased field in both InsightsOverviewMetricsQuery and
// InsightsFunctionMetricsQuery without a per-alias type.
export type MetricTable =
  | {
      query: string;
      columns: { name: string }[];
      rows: { values: unknown[] }[];
    }
  | null
  | undefined;

const BUCKET_COLUMN = 'bucket_start';
const IDENTIFIER_COLUMN = 'identifier';
const FUNCTION_ID_COLUMN = 'function_id';
const SESSION_KEY_COLUMN = 'session_key';

// toNumber coerces one raw JSON cell value into a number, or undefined for
// a NULL/unparseable cell — mirrors getNullableFloat in the backend's
// (now-removed) parse layer: skip the value entirely rather than emit a
// misleading 0.
function toNumber(raw: unknown): number | undefined {
  if (raw === null || raw === undefined) return undefined;
  const value = typeof raw === 'number' ? raw : parseFloat(String(raw));
  return Number.isNaN(value) ? undefined : value;
}

// namedValuesFromRow reads one NamedValue per column in row, skipping the
// given column names (e.g. the bucket/identifier columns, which aren't
// values) and any column whose cell doesn't parse as a number.
function namedValuesFromRow(
  columns: { name: string }[],
  row: { values: unknown[] },
  skip: Set<string>
): NamedValue[] {
  const out: NamedValue[] = [];
  columns.forEach((col, i) => {
    if (skip.has(col.name)) return;
    const value = toNumber(row.values[i]);
    if (value === undefined) return;
    out.push({ name: col.name, value });
  });
  return out;
}

function columnIndex(columns: { name: string }[], name: string): number {
  return columns.findIndex((c) => c.name === name);
}

// toScalarValues reconstructs a single-aggregate-row result (e.g.
// headline): every non-identifier/bucket column becomes one NamedValue.
export function toScalarValues(table: MetricTable): NamedValue[] {
  if (!table || table.rows.length === 0) return [];
  return namedValuesFromRow(table.columns, table.rows[0], new Set());
}

// toListItems reconstructs a List-shaped result (ranked or categorical):
// every row becomes one Item keyed by the "identifier" column. A
// "function_id"/"session_key" column, when present, becomes `functionId`/
// `sessionKey` rather than a NamedValue (they're strings, not numbers).
export function toListItems(table: MetricTable): InsightsMetricItem[] {
  if (!table) return [];
  const idIdx = columnIndex(table.columns, IDENTIFIER_COLUMN);
  const functionIdIdx = columnIndex(table.columns, FUNCTION_ID_COLUMN);
  const sessionKeyIdx = columnIndex(table.columns, SESSION_KEY_COLUMN);
  return table.rows.map((row) => ({
    identifier: idIdx >= 0 ? String(row.values[idIdx] ?? '') : '',
    functionId: functionIdIdx >= 0 ? String(row.values[functionIdIdx] ?? '') : undefined,
    sessionKey: sessionKeyIdx >= 0 ? String(row.values[sessionKeyIdx] ?? '') : undefined,
    values: namedValuesFromRow(
      table.columns,
      row,
      new Set([IDENTIFIER_COLUMN, FUNCTION_ID_COLUMN, SESSION_KEY_COLUMN]),
    ),
  }));
}

// toTrendPoints reconstructs a flat bucketed time-series result: every row
// becomes one Point keyed by the "bucket_start" column.
export function toTrendPoints(table: MetricTable): InsightsMetricPoint[] {
  if (!table) return [];
  const bucketIdx = columnIndex(table.columns, BUCKET_COLUMN);
  return table.rows.map((row) => ({
    timestamp: bucketIdx >= 0 ? String(row.values[bucketIdx] ?? '') : '',
    values: namedValuesFromRow(table.columns, row, new Set([BUCKET_COLUMN])),
  }));
}

// toDimensionedTrendPoints reconstructs a bucketed, per-dimension
// time-series result: the SQL groups by both "bucket_start" and
// "identifier", so several rows share one bucket_start. Every distinct
// bucket_start becomes one Point whose dimensions holds one Item per
// identifier row in that bucket, in row order — rows must already be
// ORDER BY bucket_start (ties within a bucket don't matter, but buckets
// must not interleave, since a run of matching bucket_start values is what
// groups rows into one Point).
export function toDimensionedTrendPoints(table: MetricTable): InsightsMetricPoint[] {
  if (!table) return [];
  const bucketIdx = columnIndex(table.columns, BUCKET_COLUMN);
  const idIdx = columnIndex(table.columns, IDENTIFIER_COLUMN);
  const skip = new Set([BUCKET_COLUMN, IDENTIFIER_COLUMN]);

  const points: InsightsMetricPoint[] = [];
  let currentBucket: string | undefined;

  for (const row of table.rows) {
    const bucket = bucketIdx >= 0 ? String(row.values[bucketIdx] ?? '') : '';
    let point = points[points.length - 1];
    if (!point || bucket !== currentBucket) {
      point = { timestamp: bucket, dimensions: [] };
      points.push(point);
      currentBucket = bucket;
    }
    point.dimensions!.push({
      identifier: idIdx >= 0 ? String(row.values[idIdx] ?? '') : '',
      values: namedValuesFromRow(table.columns, row, skip),
    });
  }

  return points;
}
