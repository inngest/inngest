import { formatMilliseconds } from '@inngest/components/utils/date';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import type { InsightsMetricPoint, NamedValue } from '../InsightsMetrics/types';

// toSigFigs formats `value` to a fixed number of significant figures
// (not a fixed number of decimal places) without falling back to
// scientific notation the way `toPrecision` does for larger inputs —
// decimal places are derived from the value's own magnitude, so 0.05123,
// 1.235, and 123.5 all carry the same 4 significant digits despite very
// different decimal-place counts.
function toSigFigs(value: number, sigFigs: number): string {
  if (value === 0) return '0';
  // Guards against Infinity/NaN reaching .toFixed() as a raw JS value (which
  // stringifies to the literal "Infinity"/"NaN") — toNumber in
  // InsightsMetrics/types.ts only screens out NaN, not Infinity, so a
  // malformed backend value could otherwise surface here.
  if (!Number.isFinite(value)) return '0';
  const magnitude = Math.floor(Math.log10(Math.abs(value)));
  const decimals = Math.max(0, sigFigs - magnitude - 1);
  return value.toFixed(decimals);
}

// formatCost is 3 significant figures under $1000 (not a fixed decimal
// count) — a $0.0000234 call and a $234 one both read with the same
// precision — and falls back to a compact K/M number above that, where
// exact cents stop being the interesting part of the number.
export function formatCost(value: number): string {
  if (value >= 1000) {
    return `$${formatCompactNumber(value)}`;
  }
  return `$${toSigFigs(value, 3)}`;
}

// formatCostAxis is formatCost's axis-tick counterpart — trailing zeros
// collapse ($1.230 -> "$1.23", $2.000 -> "$2") since a row of axis labels
// reads better compact, whereas a tooltip's fixed significant-figure count
// keeps every value comparably precise.
export function formatCostAxis(value: number): string {
  if (value >= 1000) {
    return `$${formatCompactNumber(value)}`;
  }
  return `$${parseFloat(toSigFigs(value, 3))}`;
}

// headlineCaveat surfaces the ai_headline registry entry's unpriced-usage
// tracking (see pkg/applogic/dashboards/registry.go) as a human caveat, so
// the cost tile doesn't look artificially complete when some calls used a
// model the pricing table doesn't know about.
export function headlineCaveat(values: NamedValue[] | undefined): string | undefined {
  if (!values) return undefined;
  const byName = new Map(values.map((v) => [v.name, v.value]));
  const unpricedCalls = byName.get('unpriced_calls') ?? 0;
  if (unpricedCalls <= 0) return undefined;

  const unpricedTokens =
    (byName.get('unpriced_input_tokens') ?? 0) +
    (byName.get('unpriced_output_tokens') ?? 0);

  return `${formatCompactNumber(unpricedCalls)} call${unpricedCalls === 1 ? '' : 's'} (${formatCompactNumber(unpricedTokens)} tokens) used a model without pricing data and are excluded from cost.`;
}

export function formatMs(value: number): string {
  return formatMilliseconds(Math.round(value));
}

// formatSeconds is for charts that already convert ms values to seconds in
// their own data (e.g. via msPointsToSeconds, or RangePlot's own ms->s
// conversion) — 4 significant figures (not a fixed decimal count) since AI
// call latencies span anywhere from well under a second to tens of seconds.
export function formatSeconds(value: number): string {
  return `${toSigFigs(value, 4)}s`;
}

// formatSecondsAxis is formatSeconds' Y/X-axis-tick counterpart — trailing
// zeros collapse (1.230 -> "1.23s", 2.000 -> "2s") since a row of axis
// labels reads better compact, whereas a tooltip's fixed significant-figure
// count keeps every value comparably precise.
export function formatSecondsAxis(value: number): string {
  return `${parseFloat(toSigFigs(value, 4))}s`;
}

// msPointsToSeconds converts named values from milliseconds to seconds on
// every point — for charts (e.g. TrendChart) that plot raw numbers rather
// than formatting each value, so the unit conversion has to happen in the
// data itself.
export function msPointsToSeconds(
  points: InsightsMetricPoint[] | undefined,
  valueNames: string[],
): InsightsMetricPoint[] | undefined {
  if (!points) return points;
  const names = new Set(valueNames);
  return points.map((p) => ({
    ...p,
    values: (p.values ?? []).map((v) => (names.has(v.name) ? { ...v, value: v.value / 1000 } : v)),
  }));
}
