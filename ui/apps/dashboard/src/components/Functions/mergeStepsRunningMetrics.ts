import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/**
 * Combines the dense Running and concurrency-limit results from the backend.
 * Short runs of null Running values remain chart gaps, while runs lasting at
 * least three minutes are represented as inferred zeroes. Concurrency Limit is
 * a marker that changes the chart background; its observed numeric value is
 * converted to a boolean, while null remains absent.
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
