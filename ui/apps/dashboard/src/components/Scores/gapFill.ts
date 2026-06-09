import type { ScoreBucket, ScoreKind } from './types';

const emptyBucket = (kind: ScoreKind, bucketStart: string): ScoreBucket => ({
  bucketStart,
  p50: null,
  p90: null,
  p99: null,
  trueCount: kind === 'BOOLEAN' ? 0 : null,
  falseCount: kind === 'BOOLEAN' ? 0 : null,
});

// Synthesize buckets for the requested window at the server's resolved width so
// the chart's X axis stays continuous. Server omits empty buckets.
export function gapFill({
  buckets,
  kind,
  bucketSeconds,
  from,
  to,
}: {
  buckets: ScoreBucket[];
  kind: ScoreKind;
  bucketSeconds: number;
  from: Date;
  to: Date;
}): ScoreBucket[] {
  if (bucketSeconds <= 0) return buckets;

  const stepMs = bucketSeconds * 1000;
  // Server buckets aren't necessarily epoch-aligned, so derive the grid phase
  // from the first returned bucket; otherwise real buckets would miss the
  // synthesized grid points and get dropped. With no buckets, phase 0
  // (epoch alignment) is as good as any.
  const phase =
    buckets.length > 0
      ? new Date(buckets[0].bucketStart).getTime() % stepMs
      : 0;
  const startMs =
    Math.floor((from.getTime() - phase) / stepMs) * stepMs + phase;
  const endMs = to.getTime();

  const present = new Map<number, ScoreBucket>();
  for (const b of buckets) {
    present.set(new Date(b.bucketStart).getTime(), b);
  }

  const out: ScoreBucket[] = [];
  for (let t = startMs; t <= endMs; t += stepMs) {
    const existing = present.get(t);
    out.push(existing ?? emptyBucket(kind, new Date(t).toISOString()));
  }
  return out;
}
