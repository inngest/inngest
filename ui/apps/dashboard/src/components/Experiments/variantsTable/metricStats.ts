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

export function formatMetricValue(val: number): string {
  if (Number.isNaN(val)) return '-';
  if (Math.abs(val) >= 1000)
    return val.toLocaleString(undefined, { maximumFractionDigits: 1 });
  if (Number.isInteger(val)) return String(val);
  // Small floats: trim trailing zeros but keep up to 3 decimal places
  return parseFloat(val.toFixed(3)).toString();
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
