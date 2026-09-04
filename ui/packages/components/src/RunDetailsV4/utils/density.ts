/**
 * The density strip: what loads first, and what answers "where is the problem"
 * before anyone scrolls or zooms.
 *
 * The acceptance test is specific — a failure cluster two-thirds through a
 * 500-step run should be a red smear visible in the first screenful with **no
 * interaction**. That rules out anything that needs hovering, expanding or
 * reading, and it rules out a summary: a count of failures does not tell you
 * *where*, and where is the question.
 *
 * Two properties follow from that and are worth stating because they are easy
 * to break later:
 *
 *  - It renders the WHOLE run, always, however much is collapsed below it. The
 *    strip is the map; collapsing is a property of the territory.
 *  - It shares the timeline's axis, including its breaks, so a column above a
 *    row really is that row.
 */
import type { TimelineBarData } from '../TimelineBar.types';
import type { TimeScale } from './timeScale';

/**
 * Deliberately fewer states than `CanvasStatus`. This is a five-pixel column
 * seen at a glance: the only distinctions it can carry are the ones a reader
 * can act on.
 */
export type DensityState = 'failed' | 'cancelled' | 'running' | 'waiting' | 'completed';

export type DensityBucket = {
  startMs: number;
  endMs: number;
  /** Position in the plot, from the same scale the rows use. */
  startPercent: number;
  endPercent: number;
  counts: Record<DensityState, number>;
  total: number;
};

const EMPTY: Record<DensityState, number> = {
  failed: 0,
  cancelled: 0,
  running: 0,
  waiting: 0,
  completed: 0,
};

/** Most serious first — the order the column stacks in, bottom-up. */
export const DENSITY_ORDER: DensityState[] = [
  'failed',
  'cancelled',
  'running',
  'waiting',
  'completed',
];

function toState(status: string | undefined): DensityState {
  switch ((status ?? '').toUpperCase()) {
    case 'FAILED':
      return 'failed';
    case 'CANCELLED':
      return 'cancelled';
    case 'RUNNING':
    case 'QUEUED':
      return 'running';
    case 'WAITING':
      return 'waiting';
    default:
      return 'completed';
  }
}

/** Every real step in the run, however the views below are collapsing it. */
function leaves(bars: TimelineBarData[], out: TimelineBarData[] = []): TimelineBarData[] {
  for (const bar of bars) {
    if (bar.children?.length) leaves(bar.children, out);
    else if (!bar.isRoot) out.push(bar);
  }
  return out;
}

/**
 * Bucket a run into `count` columns.
 *
 * A step is counted in every bucket it overlaps, not just the one it started
 * in. A slow failing step should smear across the width it actually occupied —
 * that width is the signal.
 */
export function densityBuckets(
  bars: TimelineBarData[],
  scale: TimeScale,
  count = 120
): DensityBucket[] {
  const { minMs, maxMs } = scale;
  if (maxMs <= minMs || count < 1) return [];

  const buckets: DensityBucket[] = [];
  for (let i = 0; i < count; i++) {
    const startPercent = (i / count) * 100;
    const endPercent = ((i + 1) / count) * 100;
    buckets.push({
      // Bucket edges are in *drawn* space and converted back to time, so a
      // compressed stretch occupies the few columns it is drawn in rather than
      // most of them.
      startMs: percentToMsLinear(scale, startPercent),
      endMs: percentToMsLinear(scale, endPercent),
      startPercent,
      endPercent,
      counts: { ...EMPTY },
      total: 0,
    });
  }

  for (const bar of leaves(bars)) {
    const state = toState(bar.status);
    const startPercent = scale.toPercent(bar.startTime.getTime());
    const endPercent = scale.toPercent((bar.endTime ?? bar.startTime).getTime());

    const first = Math.max(0, Math.min(count - 1, Math.floor((startPercent / 100) * count)));
    const last = Math.max(first, Math.min(count - 1, Math.floor((endPercent / 100) * count)));

    for (let i = first; i <= last; i++) {
      const bucket = buckets[i]!;
      bucket.counts[state] += 1;
      bucket.total += 1;
    }
  }

  return buckets;
}

/**
 * Inverse of the scale, good enough for a bucket edge.
 *
 * `percentToMs` in `timeScale` binary-searches for precision; here the value is
 * only ever shown in a tooltip, and doing it 120 times per render is not worth
 * 40 iterations each.
 */
function percentToMsLinear(scale: TimeScale, percent: number): number {
  if (!scale.compressed) {
    return scale.minMs + ((scale.maxMs - scale.minMs) * percent) / 100;
  }
  let lo = scale.minMs;
  let hi = scale.maxMs;
  for (let i = 0; i < 12; i++) {
    const mid = (lo + hi) / 2;
    if (scale.toPercent(mid) < percent) lo = mid;
    else hi = mid;
  }
  return (lo + hi) / 2;
}

/** The tallest column, so the strip can scale to what is actually there. */
export function densityPeak(buckets: DensityBucket[]): number {
  return buckets.reduce((n, b) => Math.max(n, b.total), 0);
}
