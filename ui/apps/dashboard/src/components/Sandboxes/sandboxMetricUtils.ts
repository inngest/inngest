import { SandboxMetric } from '@/gql/graphql';

type MetricPoint = {
  time: string;
  value?: number | null;
};

type SandboxMetricSeries = {
  metric: SandboxMetric;
  data: MetricPoint[];
};

export function sandboxMetricStats(
  series: SandboxMetricSeries[],
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
  const byMetric = new Map(series.map((item) => [item.metric, item.data]));
  const cpuRate = latestCounterRate(
    sumMatchingCounters(
      byMetric.get(SandboxMetric.CpuUserTime) ?? [],
      byMetric.get(SandboxMetric.CpuSystemTime) ?? [],
    ),
    1 / 1_000_000,
  );
  const memoryCurrent = latestValue(
    byMetric.get(SandboxMetric.MemoryCurrent) ?? [],
  );
  const memoryPeak = latestValue(byMetric.get(SandboxMetric.MemoryPeak) ?? []);

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
  const rightByTime = new Map(
    right.filter(hasMetricValue).map((point) => [point.time, point.value]),
  );

  return left.flatMap((point) => {
    const rightValue = rightByTime.get(point.time);
    if (!hasMetricValue(point) || rightValue === undefined) return [];
    return [{ time: point.time, value: point.value + rightValue }];
  });
}

function latestCounterRate(
  source: MetricPoint[],
  scale: number,
): number | null {
  const points = source
    .filter(hasMetricValue)
    .sort((a, b) => Date.parse(a.time) - Date.parse(b.time));
  let latest: number | null = null;

  for (let i = 1; i < points.length; i++) {
    const previous = points[i - 1];
    const current = points[i];
    const elapsedSeconds =
      (Date.parse(current.time) - Date.parse(previous.time)) / 1000;
    if (elapsedSeconds <= 0 || current.value < previous.value) continue;
    latest = ((current.value - previous.value) / elapsedSeconds) * scale;
  }

  return latest;
}

function latestValue(points: MetricPoint[]): number | null {
  const latest = points
    .filter(hasMetricValue)
    .sort((left, right) => Date.parse(right.time) - Date.parse(left.time))[0];
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
