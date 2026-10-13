import { SandboxStatus } from '@/gql/graphql';

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

/**
 * Summarizes live usage. Each bucket holds its latest sample, so the newest
 * bucket is still filling until its interval ends; CPU rates only use
 * complete buckets, whose samples sit one bucket width apart.
 */
export function sandboxMetricStats(
  series: SandboxMetricSeries,
  vcpu: number,
  memoryBytes: number,
  range: { until: Date; bucketMS: number },
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
    ).filter(
      (point) =>
        Date.parse(point.bucket) + range.bucketMS <= range.until.getTime(),
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

// Sandbox metric points live in ClickHouse's compute_metric_points table,
// which expires rows after 30 days.
export const SANDBOX_METRICS_RETENTION_MS = 30 * 24 * 60 * 60 * 1_000;

// How far past a sandbox's end we look for its final samples.
const LIFETIME_TAIL_MS = 60 * 1_000;

export type SandboxMetricsWindow =
  | { kind: 'live' }
  | { kind: 'pending' }
  | { kind: 'neverStarted' }
  | { kind: 'expired'; startedAt: Date; endedAt: Date }
  | {
      kind: 'lifetime';
      startedAt: Date;
      endedAt: Date;
      from: Date;
      until: Date;
    };

/**
 * Picks what the resource cards show for a sandbox: live usage while it runs,
 * usage over its whole lifetime once it has ended, or an explanation when no
 * usage can exist.
 */
export function sandboxMetricsWindow(
  sandbox: {
    status: SandboxStatus;
    startedAt?: string | null;
    endedAt?: string | null;
    failedAt?: string | null;
    launchUnknownAt?: string | null;
  },
  now: Date,
): SandboxMetricsWindow {
  switch (sandbox.status) {
    case SandboxStatus.Running:
    case SandboxStatus.Stopping:
      return { kind: 'live' };
    case SandboxStatus.Pending:
      return { kind: 'pending' };
  }

  if (!sandbox.startedAt) return { kind: 'neverStarted' };

  const startedAt = new Date(sandbox.startedAt);
  const end = sandbox.endedAt ?? sandbox.failedAt ?? sandbox.launchUnknownAt;
  const endedAt = new Date(
    Math.max(end ? Date.parse(end) : now.getTime(), startedAt.getTime()),
  );

  if (now.getTime() - endedAt.getTime() > SANDBOX_METRICS_RETENTION_MS) {
    return { kind: 'expired', startedAt, endedAt };
  }

  const until = new Date(
    Math.max(
      Math.min(endedAt.getTime() + LIFETIME_TAIL_MS, now.getTime()),
      startedAt.getTime() + 1_000,
    ),
  );
  return { kind: 'lifetime', startedAt, endedAt, from: startedAt, until };
}

/**
 * Summarizes usage over a sandbox's whole run. CPU counters are cumulative
 * from launch, so the final reading divided by runtime is the average vCPU.
 * memory.peak is a high-water mark, so its largest reading is the lifetime
 * peak.
 */
export function sandboxLifetimeStats(
  series: SandboxMetricSeries,
  vcpu: number,
  memoryBytes: number,
  runtimeSeconds: number,
): {
  cpu: { average: number; percent: number } | null;
  memory: { peak: number; percent: number } | null;
} {
  const cpuTotal = maxValue(
    sumMatchingCounters(
      series.cpuUserTime?.data ?? [],
      series.cpuSystemTime?.data ?? [],
    ),
  );
  const memoryPeak =
    maxValue(series.memoryPeak?.data ?? []) ??
    maxValue(series.memoryCurrent?.data ?? []);
  const cpuAverage =
    cpuTotal === null || runtimeSeconds <= 0
      ? null
      : cpuTotal / 1_000_000 / runtimeSeconds;

  return {
    cpu:
      cpuAverage === null
        ? null
        : { average: cpuAverage, percent: percentage(cpuAverage, vcpu) },
    memory:
      memoryPeak === null
        ? null
        : { peak: memoryPeak, percent: percentage(memoryPeak, memoryBytes) },
  };
}

function maxValue(points: MetricPoint[]): number | null {
  const values = points.filter(hasMetricValue).map((point) => point.value);
  return values.length ? Math.max(...values) : null;
}
