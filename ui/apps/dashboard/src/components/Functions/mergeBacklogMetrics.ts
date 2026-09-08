import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/**
 * Converts the separate Queued and Sleeping results from the backend into
 * chart data with one row per timestamp. Both results are gauges whose numeric
 * values are plotted as lines, so the missing-data policy is applied to each:
 * short gaps remain absent for the chart to connect over, while gaps of at
 * least three minutes are represented as zero and marked as inferred.
 */
export function mergeBacklogMetrics(
  scheduled: Metric[],
  sleeping: Metric[],
  granularity: string,
  rangeStart: string,
  rangeEnd: string,
) {
  return mergeMetricRows(
    [
      { key: 'scheduled', data: scheduled, inferMissingAsZero: true },
      { key: 'sleeping', data: sleeping, inferMissingAsZero: true },
    ],
    granularity,
    rangeStart,
    rangeEnd,
  );
}
