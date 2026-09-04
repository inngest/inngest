/**
 * An elastic time axis.
 *
 * A realistic run is nine days elapsed and five seconds executing. Drawn
 * linearly, every step is sub-pixel and the view is a single bar with nothing in
 * it — see the `longgap` fixture, where a 208ms step and an 18ms step sit at the
 * far left of seven days.
 *
 * So the axis breaks wherever the run is idle, and the idle stretch is drawn as
 * a fixed narrow band rather than at its true width.
 *
 * THE INVARIANT THAT MAKES THIS SAFE: only the axis compresses. Every reported
 * duration stays wall-clock. A step that took 890ms says 890ms whatever the axis
 * does — and that is enforced structurally rather than by convention, because
 * this module only ever produces *positions*. Durations come from
 * `calculateDuration`, which knows nothing about any of this. Break that and the
 * view becomes untrustworthy in a way users feel but cannot articulate.
 *
 * A break must also be visibly a break. A compressed gap is not drawn as empty
 * space — the renderer marks it, because an axis that is not linear has to say
 * so.
 */

/** A stretch of time when something was actually happening. */
export type Interval = { startMs: number; endMs: number };

/** A stretch of idle time that the axis compresses. */
export type TimeGap = {
  startMs: number;
  endMs: number;
  /** Real elapsed time being compressed. Reported to the user, never scaled. */
  durationMs: number;
  /** Where the band is drawn, as percentages of the plot width. */
  startPercent: number;
  endPercent: number;
};

/** One piece of the piecewise-linear mapping from time to width. */
type Segment = {
  startMs: number;
  endMs: number;
  startPercent: number;
  endPercent: number;
};

export type TimeScale = {
  minMs: number;
  maxMs: number;
  /** Position of a timestamp, as a percentage of the plot width. */
  toPercent(ms: number): number;
  /** The compressed stretches, in time order. Empty when nothing was broken. */
  gaps: TimeGap[];
  /** Whether the axis is non-linear. When false this is an ordinary scale. */
  compressed: boolean;
};

export type TimeScaleOptions = {
  /**
   * A gap must be at least this long to be worth breaking, whatever the run's
   * total. Below a few seconds a break costs the reader more than it saves.
   */
  minGapMs?: number;
  /**
   * ...and must also be at least this fraction of the run. A 10s gap in a run
   * that took an hour is not what makes the view unreadable.
   */
  minGapFraction?: number;
  /** Width each compressed gap is given, as a fraction of the plot. */
  gapWidthFraction?: number;
};

const DEFAULTS = {
  minGapMs: 5_000,
  minGapFraction: 0.15,
  gapWidthFraction: 0.05,
} satisfies Required<TimeScaleOptions>;

/** Merge overlapping or touching intervals into a sorted, disjoint list. */
export function mergeIntervals(intervals: Interval[]): Interval[] {
  const sorted = intervals
    .filter((i) => Number.isFinite(i.startMs) && Number.isFinite(i.endMs))
    .map((i) => ({ startMs: i.startMs, endMs: Math.max(i.startMs, i.endMs) }))
    .sort((a, b) => a.startMs - b.startMs);

  const merged: Interval[] = [];
  for (const next of sorted) {
    const last = merged[merged.length - 1];
    if (last && next.startMs <= last.endMs) {
      last.endMs = Math.max(last.endMs, next.endMs);
    } else {
      merged.push({ ...next });
    }
  }
  return merged;
}

/** The ordinary linear axis, for runs with nothing worth breaking. */
export function linearScale(minMs: number, maxMs: number): TimeScale {
  const total = maxMs - minMs;
  return {
    minMs,
    maxMs,
    toPercent: (ms) => (total <= 0 ? 0 : ((ms - minMs) / total) * 100),
    gaps: [],
    compressed: false,
  };
}

/**
 * Build an axis that breaks wherever the run was idle.
 *
 * `busy` is every stretch when something was happening — typically one interval
 * per step. Anything between them is idle and a candidate for compression.
 */
export function buildTimeScale(
  minMs: number,
  maxMs: number,
  busy: Interval[],
  options: TimeScaleOptions = {}
): TimeScale {
  const { minGapMs, minGapFraction, gapWidthFraction } = { ...DEFAULTS, ...options };

  const total = maxMs - minMs;
  if (total <= 0) return linearScale(minMs, maxMs);

  // Idle stretches are the holes between the merged busy intervals, clipped to
  // the run's own bounds.
  const merged = mergeIntervals(busy).filter((i) => i.endMs > minMs && i.startMs < maxMs);

  const idle: Interval[] = [];
  let cursor = minMs;
  for (const interval of merged) {
    if (interval.startMs > cursor) idle.push({ startMs: cursor, endMs: interval.startMs });
    cursor = Math.max(cursor, interval.endMs);
  }
  if (cursor < maxMs) idle.push({ startMs: cursor, endMs: maxMs });

  const breaks = idle.filter((gap) => {
    const duration = gap.endMs - gap.startMs;
    return duration >= minGapMs && duration / total >= minGapFraction;
  });

  if (!breaks.length) return linearScale(minMs, maxMs);

  // The gaps take a fixed share between them; everything else divides the rest
  // in proportion to its real duration, so the parts of the run that are not
  // compressed remain linear relative to each other.
  const gapShare = Math.min(0.6, gapWidthFraction * breaks.length);
  const compressedMs = breaks.reduce((n, g) => n + (g.endMs - g.startMs), 0);
  const liveMs = total - compressedMs;
  const livePercent = (1 - gapShare) * 100;
  const perGapPercent = (gapShare * 100) / breaks.length;

  const segments: Segment[] = [];
  const gaps: TimeGap[] = [];

  let atMs = minMs;
  let atPercent = 0;

  for (const gap of breaks) {
    if (gap.startMs > atMs) {
      const width = liveMs > 0 ? ((gap.startMs - atMs) / liveMs) * livePercent : 0;
      segments.push({
        startMs: atMs,
        endMs: gap.startMs,
        startPercent: atPercent,
        endPercent: atPercent + width,
      });
      atPercent += width;
      atMs = gap.startMs;
    }

    segments.push({
      startMs: gap.startMs,
      endMs: gap.endMs,
      startPercent: atPercent,
      endPercent: atPercent + perGapPercent,
    });
    gaps.push({
      startMs: gap.startMs,
      endMs: gap.endMs,
      durationMs: gap.endMs - gap.startMs,
      startPercent: atPercent,
      endPercent: atPercent + perGapPercent,
    });

    atPercent += perGapPercent;
    atMs = gap.endMs;
  }

  if (atMs < maxMs) {
    segments.push({
      startMs: atMs,
      endMs: maxMs,
      startPercent: atPercent,
      endPercent: 100,
    });
  }

  const toPercent = (ms: number): number => {
    if (ms <= minMs) return 0;
    if (ms >= maxMs) return 100;

    for (const segment of segments) {
      if (ms >= segment.startMs && ms <= segment.endMs) {
        const span = segment.endMs - segment.startMs;
        if (span <= 0) return segment.startPercent;
        const ratio = (ms - segment.startMs) / span;
        return segment.startPercent + ratio * (segment.endPercent - segment.startPercent);
      }
    }
    return 100;
  };

  return { minMs, maxMs, toPercent, gaps, compressed: true };
}

/**
 * Tick positions for the axis.
 *
 * Evenly spaced *in the drawn width*, not in time — the whole point is that
 * those are different — so each tick reports the real time at its position and
 * the labels are not evenly spaced values. Ticks that would land inside a
 * compressed gap are dropped: a label there would be reporting a time from a
 * stretch the axis is explicitly not drawing.
 */
export function scaleTicks(scale: TimeScale, count = 5): Array<{ percent: number; ms: number }> {
  const ticks: Array<{ percent: number; ms: number }> = [];

  for (let i = 0; i < count; i++) {
    const percent = (i / (count - 1)) * 100;
    const ms = percentToMs(scale, percent);
    const insideGap = scale.gaps.some(
      (gap) => percent > gap.startPercent + 0.01 && percent < gap.endPercent - 0.01
    );
    if (!insideGap) ticks.push({ percent, ms });
  }

  return ticks;
}

/** The inverse of `toPercent`, for placing tick labels. */
export function percentToMs(scale: TimeScale, percent: number): number {
  if (!scale.compressed) {
    return scale.minMs + ((scale.maxMs - scale.minMs) * percent) / 100;
  }

  // Walk the same piecewise mapping backwards. Cheap: there are only ever a
  // handful of segments.
  let lo = scale.minMs;
  let hi = scale.maxMs;
  for (let i = 0; i < 40; i++) {
    const mid = (lo + hi) / 2;
    if (scale.toPercent(mid) < percent) lo = mid;
    else hi = mid;
  }
  return (lo + hi) / 2;
}
