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

/** Combines dense metric series that share the same ordered buckets. */
export function mergeMetricRows(
  series: MetricSeries[],
  granularity: string,
  rangeStart: string,
  rangeEnd: string,
) {
  const amount = Number.parseInt(granularity, 10);
  const bucketWidth = amount * (granularity.endsWith('h') ? 60 : 1) * 60_000;
  const inferredByKey = new Map(
    series
      .filter(({ inferMissingAsZero }) => inferMissingAsZero)
      .map(({ key, data }) => [
        key,
        new Set(
          findSustainedMissingBuckets(data, bucketWidth, rangeStart, rangeEnd),
        ),
      ]),
  );

  // The backend contract guarantees that every response has the same dense,
  // ordered buckets. Choosing the longest response is only defensive so one
  // missing response does not hide another; this does not align sparse series.
  const buckets = series.reduce<Metric[]>((longest, { data }) => {
    return data.length > longest.length ? data : longest;
  }, []);

  return buckets.map(({ bucket }, index): MetricRow => {
    const timestamp = new Date(bucket).getTime();
    const metric: MetricRow = { name: bucket, values: {} };

    for (const { key, data, mapValue } of series) {
      const value = data[index]?.value;
      if (value === undefined) continue;

      if (inferredByKey.get(key)?.has(timestamp)) {
        metric.values[key] = 0;
        metric.inferred = [...(metric.inferred ?? []), key];
      } else {
        // Preserve null as a chart gap and don't pass it to mappers; for
        // example, Boolean(null) would conflate absence with reported false.
        metric.values[key] =
          value === null ? null : (mapValue?.(value) ?? value);
      }
    }

    return metric;
  });
}
