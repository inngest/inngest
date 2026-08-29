import { findSustainedMissingBuckets } from './findSustainedMissingBuckets';

export type Metric = {
  bucket: string;
  value: number | null;
};

export type MetricSeries = {
  key: string;
  data: Metric[];
  mapValue?: (value: number) => number | boolean;
  inferMissingAsZero?: boolean;
};

type MetricRow = {
  name: string;
  values: Record<string, number | boolean | null>;
  inferred?: string[];
};

/** Builds one chart row per timestamp from independently reported series. */
export function mergeMetricRows(
  series: MetricSeries[],
  granularity: string,
  rangeStart: string,
  rangeEnd: string,
) {
  const byBucket = new Map<number, MetricRow>();

  for (const { key, data, mapValue } of series) {
    for (const { bucket, value } of data) {
      const timestamp = new Date(bucket).getTime();
      const metric = byBucket.get(timestamp) ?? { name: bucket, values: {} };

      // Preserve null as a chart gap and don't pass it to mappers; for example,
      // Boolean(null) would conflate absence with a reported false value.
      metric.values[key] = value === null ? null : (mapValue?.(value) ?? value);
      byBucket.set(timestamp, metric);
    }
  }

  const amount = Number.parseInt(granularity, 10);
  const bucketWidth = amount * (granularity.endsWith('h') ? 60 : 1) * 60_000;

  for (const { key, data, inferMissingAsZero } of series) {
    if (!inferMissingAsZero) continue;

    for (const timestamp of findSustainedMissingBuckets(
      data,
      bucketWidth,
      rangeStart,
      rangeEnd,
    )) {
      const metric = byBucket.get(timestamp) ?? {
        name: new Date(timestamp).toISOString(),
        values: {},
      };
      metric.values[key] = 0;
      metric.inferred = [...(metric.inferred ?? []), key];
      byBucket.set(timestamp, metric);
    }
  }

  const observed = Array.from(byBucket.values()).sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  if (observed.length < 2) {
    return observed;
  }

  const firstBucket = new Date(observed[0].name).getTime();
  const lastBucket = new Date(observed.at(-1)!.name).getTime();
  const metrics: MetricRow[] = [];

  for (
    let timestamp = firstBucket;
    timestamp <= lastBucket;
    timestamp += bucketWidth
  ) {
    metrics.push(
      byBucket.get(timestamp) ?? {
        name: new Date(timestamp).toISOString(),
        values: {},
      },
    );
  }

  return metrics;
}
