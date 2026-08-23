import type { Metric } from './mergeMetricRows';
import { mergeMetricRows } from './mergeMetricRows';

/** Converts separate Queued and Sleeping results into one row per timestamp. */
export function mergeBacklogMetrics(
  scheduled: Metric[],
  sleeping: Metric[],
  granularity: string,
) {
  return mergeMetricRows(
    [
      { key: 'scheduled', data: scheduled },
      { key: 'sleeping', data: sleeping },
    ],
    granularity,
  );
}
