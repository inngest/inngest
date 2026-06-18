import type { MetricsData, ScopedMetric } from '@/gql/graphql';

export const ZERO_ID = '00000000-0000-0000-0000-000000000000';

export const sumScopedMetricData = (
  metrics?: Array<Pick<ScopedMetric, 'data' | 'id'>>,
): Array<Pick<MetricsData, 'bucket' | 'value'>> => {
  const totals = new Map<string, number>();
  const buckets: string[] = [];

  for (const metric of metrics ?? []) {
    if (metric.id === ZERO_ID) {
      continue;
    }

    for (const { bucket, value } of metric.data) {
      if (!totals.has(bucket)) {
        buckets.push(bucket);
      }

      totals.set(bucket, (totals.get(bucket) ?? 0) + value);
    }
  }

  return buckets.map((bucket) => ({
    bucket,
    value: totals.get(bucket) ?? 0,
  }));
};
