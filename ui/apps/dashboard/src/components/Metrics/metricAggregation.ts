import type { MetricsData, ScopedMetric } from '@/gql/graphql';

export const ZERO_ID = '00000000-0000-0000-0000-000000000000';

type MetricDataPoint = Pick<MetricsData, 'bucket' | 'value'>;
type ScopedMetricData = Pick<ScopedMetric, 'data' | 'id'>;

export const sumScopedMetricData = (
  metrics?: Array<ScopedMetricData>,
): Array<MetricDataPoint> => {
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

export type AggregatedScopedMetric = {
  id: string;
  tagName: null;
  tagValue: null;
  data: Array<MetricDataPoint>;
};

export const sumScopedMetricsByGroup = (
  metrics: Array<ScopedMetricData> | undefined,
  getGroupID: (metric: ScopedMetricData) => string | undefined,
): Array<AggregatedScopedMetric> => {
  const buckets: string[] = [];
  const bucketSet = new Set<string>();
  const groupOrder: string[] = [];
  const groupTotals = new Map<string, Map<string, number>>();

  for (const metric of metrics ?? []) {
    if (metric.id === ZERO_ID) {
      continue;
    }

    const groupID = getGroupID(metric);
    if (!groupID) {
      continue;
    }

    if (!groupTotals.has(groupID)) {
      groupTotals.set(groupID, new Map());
      groupOrder.push(groupID);
    }

    const totals = groupTotals.get(groupID);
    if (!totals) {
      continue;
    }

    for (const { bucket, value } of metric.data) {
      if (!bucketSet.has(bucket)) {
        bucketSet.add(bucket);
        buckets.push(bucket);
      }

      totals.set(bucket, (totals.get(bucket) ?? 0) + value);
    }
  }

  return groupOrder.map((id) => {
    const totals = groupTotals.get(id);

    return {
      id,
      tagName: null,
      tagValue: null,
      data: buckets.map((bucket) => ({
        bucket,
        value: totals?.get(bucket) ?? 0,
      })),
    };
  });
};

export const latestMetricDataValue = (
  data?: Array<MetricDataPoint>,
): number => {
  let latest: MetricDataPoint | undefined;

  for (const point of data ?? []) {
    if (!latest || Date.parse(point.bucket) > Date.parse(latest.bucket)) {
      latest = point;
    }
  }

  return latest?.value ?? 0;
};
