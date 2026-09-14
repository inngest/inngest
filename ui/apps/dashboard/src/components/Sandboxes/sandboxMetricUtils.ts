type MetricPoint = {
  bucket: string;
  value?: number | null;
};

type MetricSeries = {
  data: MetricPoint[];
};

export type SandboxMetricSeries = {
  cpuUserTime?: MetricSeries | null;
  cpuSystemTime?: MetricSeries | null;
  memoryCurrent?: MetricSeries | null;
  memoryPeak?: MetricSeries | null;
};

export function sandboxMetricStats(
  series: SandboxMetricSeries,
  vcpu: number,
  memoryBytes: number,
): {
  cpu: { current: number; percent: number } | null;
  memory: {
    current: number;
    peak: number | null;
    percent: number;
    peakPercent: number | null;
  } | null;
} {
  const cpuRate = latestCounterRate(
    sumMatchingCounters(
      series.cpuUserTime?.data ?? [],
      series.cpuSystemTime?.data ?? [],
    ),
    1 / 1_000_000,
  );
  const memoryCurrent = latestValue(series.memoryCurrent?.data ?? []);
  const memoryPeak = latestValue(series.memoryPeak?.data ?? []);

  return {
    cpu:
      cpuRate === null
        ? null
        : {
            current: cpuRate,
            percent: percentage(cpuRate, vcpu),
          },
    memory:
      memoryCurrent === null
        ? null
        : {
            current: memoryCurrent,
            peak: memoryPeak,
            percent: percentage(memoryCurrent, memoryBytes),
            peakPercent:
              memoryPeak === null ? null : percentage(memoryPeak, memoryBytes),
          },
  };
}

function sumMatchingCounters(
  left: MetricPoint[],
  right: MetricPoint[],
): MetricPoint[] {
  const rightByBucket = new Map(
    right.filter(hasMetricValue).map((point) => [point.bucket, point.value]),
  );

  return left.flatMap((point) => {
    const rightValue = rightByBucket.get(point.bucket);
    if (!hasMetricValue(point) || rightValue === undefined) return [];
    return [{ bucket: point.bucket, value: point.value + rightValue }];
  });
}

function latestCounterRate(
  source: MetricPoint[],
  scale: number,
): number | null {
  const points = source
    .filter(hasMetricValue)
    .sort((a, b) => Date.parse(a.bucket) - Date.parse(b.bucket));
  let latest: number | null = null;

  for (let i = 1; i < points.length; i++) {
    const previous = points[i - 1];
    const current = points[i];
    const elapsedSeconds =
      (Date.parse(current.bucket) - Date.parse(previous.bucket)) / 1000;
    if (elapsedSeconds <= 0 || current.value < previous.value) continue;
    latest = ((current.value - previous.value) / elapsedSeconds) * scale;
  }

  return latest;
}

function latestValue(points: MetricPoint[]): number | null {
  const latest = points
    .filter(hasMetricValue)
    .sort(
      (left, right) => Date.parse(right.bucket) - Date.parse(left.bucket),
    )[0];
  return latest?.value ?? null;
}

function hasMetricValue(
  point: MetricPoint,
): point is MetricPoint & { value: number } {
  return typeof point.value === 'number' && Number.isFinite(point.value);
}

function percentage(value: number, limit: number): number {
  if (limit <= 0) return 0;
  return Math.min(Math.max((value / limit) * 100, 0), 100);
}
