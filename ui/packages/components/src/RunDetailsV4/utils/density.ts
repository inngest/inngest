/**
 * The overview above the views: what loads first, and what answers "where is
 * the problem" before anyone scrolls or zooms.
 *
 * The acceptance test is specific — a failure cluster two-thirds through a
 * 500-step run should be findable in the first screenful with **no
 * interaction**. That rules out anything needing a hover or an expand, and it
 * rules out a summary: a count of failures does not tell you *where*, and where
 * is the question.
 *
 * This began as a stacked histogram and became a packed minimap, because the
 * histogram was made of the same marks in the same colours directly above the
 * trace's own bars — it read as more trace rather than as an overview of one.
 * See `packMinimap` below.
 *
 * Two properties are load-bearing either way:
 *
 *  - It shows the WHOLE run, always, however much is collapsed below it. The
 *    map must not shrink because the territory got folded.
 *  - It shares the timeline's axis, breaks included, so a mark really does sit
 *    above the row it describes.
 */ import type { TimelineBarData } from '../TimelineBar.types';
import type { TimeScale } from './timeScale';

/**
 * Deliberately fewer states than `CanvasStatus`. This is a five-pixel column
 * seen at a glance: the only distinctions it can carry are the ones a reader
 * can act on.
 */
export type DensityState = 'failed' | 'cancelled' | 'running' | 'waiting' | 'completed';

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

/**
 * A minimap of the whole run, packed into as few rows as possible.
 *
 * The histogram this replaced was made of the same marks, in the same colours,
 * directly above the trace's own bars — so it read as more trace rather than as
 * an overview of one, and added noise exactly where the eye lands first.
 *
 * Packing gives it a different job and a different shape. Each step goes in the
 * first row whose previous step has already finished, so sequential work
 * collapses to a single line and only genuine parallelism forces another. The
 * row count therefore *is* the concurrency profile: a forty-step run that never
 * did two things at once is one line, and a twelve-wide fan-out is visibly
 * twelve. That is a property neither the trace nor the canvas shows.
 *
 * Labels are the honest cost — several steps share a row, so none can be named.
 * This is a map, not the territory; the trace underneath has the names.
 */
export type MinimapMark = {
  id: string;
  row: number;
  startPercent: number;
  widthPercent: number;
  state: DensityState;
  /** For the tooltip, since a mark is too small to label. */
  name: string;
  /**
   * This step failed at least once and got there anyway.
   *
   * The minimap's job is making trouble findable, and a recovered step is
   * trouble that a final status of COMPLETED hides completely: on the fixture
   * named `retry`, the overview was one unbroken green bar. It keeps the
   * completed colour, because the run did complete, and stops receding.
   */
  recovered: boolean;
};

export type Minimap = {
  marks: MinimapMark[];
  /** How many trace rows the map mirrors. */
  rows: number;
};

/** A step drawn from more than one attempt is one that failed and came back. */
function wasRetried(bar: TimelineBarData): boolean {
  const attempts = bar.children?.filter((child) => /^Attempt \d+$/.test(child.name)) ?? [];
  return attempts.length > 1;
}

export function packMinimap(bars: TimelineBarData[], scale: TimeScale): Minimap {
  // One mark per TRACE ROW, in the trace's own order.
  //
  // This used to pack steps into as few lanes as possible, so the row count was
  // the run's concurrency profile — a genuinely useful property, and the wrong
  // one to put here. It meant the strip above the trace had a different row
  // layout from the trace, and two things that look alike while meaning
  // different things is worse than either alone. A reader glancing up from a
  // row could not find it.
  //
  // So it is a map of the thing it sits above: same rows, same order, same
  // x positions, shrunk to a fixed height. Concurrency is still visible — it is
  // rows overlapping in x, exactly as it is in the trace itself.
  const rows = bars.flatMap((bar) => (bar.isRoot ? bar.children ?? [] : [bar]));

  const marks: MinimapMark[] = rows.map((bar, row) => {
    const startMs = bar.startTime.getTime();
    const endMs = (bar.endTime ?? bar.startTime).getTime();
    const startPercent = scale.toPercent(startMs);

    return {
      id: bar.id,
      row,
      startPercent,
      // A floor, so a 1ms step in a two-second run is still a mark rather than
      // nothing. The minimap is for finding things, not for measuring them.
      widthPercent: Math.max(0.35, scale.toPercent(endMs) - startPercent),
      state: toState(bar.status),
      name: bar.name,
      recovered: wasRetried(bar),
    };
  });

  return { marks, rows: Math.max(1, rows.length) };
}
