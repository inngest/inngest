import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/** Converts separate Running and concurrency-limit results into chart rows. */
export function mergeStepsRunningMetrics(
  running: Metric[],
  concurrencyLimit: Metric[],
  granularity: string,
) {
  return mergeMetricRows(
    [
      { key: 'running', data: running },
      { key: 'concurrencyLimit', data: concurrencyLimit, mapValue: Boolean },
    ],
    granularity,
  );
}
