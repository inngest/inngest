import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/**
 * Combines the dense Queued and Sleeping results from the backend. Both are
 * gauges, so short runs of nulls remain chart gaps while runs lasting at least
 * three minutes are represented as zero and marked as inferred.
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
