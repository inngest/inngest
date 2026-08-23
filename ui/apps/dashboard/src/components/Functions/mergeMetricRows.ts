export type Metric = {
  bucket: string;
  value: number;
};

export type MetricSeries = {
  key: string;
  data: Metric[];
  mapValue?: (value: number) => number | boolean;
};

type MetricRow = {
  name: string;
  values: Record<string, number | boolean | undefined>;
};

/** Builds one chart row per timestamp from independently reported series. */
export function mergeMetricRows(
  series: MetricSeries[],
  granularity: string,
) {
  const byBucket = new Map<number, MetricRow>();

  for (const { key, data, mapValue } of series) {
    for (const { bucket, value } of data) {
      const timestamp = new Date(bucket).getTime();
      const metric = byBucket.get(timestamp) ?? { name: bucket, values: {} };
      metric.values[key] = mapValue ? mapValue(value) : value;
      byBucket.set(timestamp, metric);
    }
  }

  const observed = Array.from(byBucket.values()).sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  if (observed.length < 2) {
    return observed;
  }

  const amount = Number.parseInt(granularity, 10);
  const bucketWidth = amount * (granularity.endsWith('h') ? 60 : 1) * 60_000;
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
