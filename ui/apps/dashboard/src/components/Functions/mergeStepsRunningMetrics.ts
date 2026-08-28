import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/**
 * Converts the separate Running and concurrency-limit results from the backend
 * into chart data with one row per timestamp. Running is a gauge whose numeric
 * values are plotted as a line, so short gaps remain absent and gaps of at
 * least three minutes are represented as inferred zeroes. Concurrency Limit is
 * instead a marker that changes the chart background; its numeric value is
 * converted to a boolean at the matching timestamp, and missing markers are
 * not inferred because absence means the limit was not hit.
 */
export function mergeStepsRunningMetrics(
  running: Metric[],
  concurrencyLimit: Metric[],
  granularity: string,
  rangeStart: string,
  rangeEnd: string,
) {
  return mergeMetricRows(
    [
      { key: 'running', data: running, inferMissingAsZero: true },
      { key: 'concurrencyLimit', data: concurrencyLimit, mapValue: Boolean },
    ],
    granularity,
    rangeStart,
    rangeEnd,
  );
}
