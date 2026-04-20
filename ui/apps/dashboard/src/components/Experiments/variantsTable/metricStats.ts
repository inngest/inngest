import type {
  ExperimentVariantMetrics,
  VariantMetric,
} from '@inngest/components/Experiments';

import { findBestAndWorst } from '@/lib/experiments/score';

export type MetricStats = {
  bestAvg: number;
  worstAvg: number;
  bestVariant: string;
  worstVariant: string;
};

export type MetricRow = ExperimentVariantMetrics & {
  metricsByKey: Map<string, VariantMetric>;
};

/** Numeric round to 2 decimals. Shared with display formatting so rounded
 * values written back to state match what the UI shows. */
export function roundMetricValue(val: number): number {
  return Number(val.toFixed(2));
}

export function formatMetricValue(val: number): string {
  if (Number.isNaN(val)) return '-';
  // Round to 2 decimals, trim trailing zeros, and keep locale-aware
  // thousands separators via toLocaleString.
  return roundMetricValue(val).toLocaleString(undefined, {
    maximumFractionDigits: 2,
  });
}

export function computeMetricStats(
  rows: MetricRow[],
  metricKey: string,
  invert: boolean,
): MetricStats | null {
  const entries: { name: string; avg: number }[] = [];
  for (const row of rows) {
    const m = row.metricsByKey.get(metricKey);
    if (m) entries.push({ name: row.variantName, avg: m.avg });
  }
  const pair = findBestAndWorst(entries, (e) => e.avg, invert);
  if (!pair) return null;
  return {
    bestAvg: pair.best.avg,
    worstAvg: pair.worst.avg,
    bestVariant: pair.best.name,
    worstVariant: pair.worst.name,
  };
}
