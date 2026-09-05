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
 * The steps of this run: the root's own children, and nothing deeper.
 *
 * Descending further finds attempts and invoked child runs, which are not steps
 * *of this run*. A retried step is one step that was tried twice, and the
 * minimap was drawing `retry` as "Attempt 0, Attempt 1, Attempt 0, Attempt 1"
 * — four marks for one step and one finalization.
 *
 * Platform rows are skipped for the same reason: Planning and Finalization are
 * machinery, and packing them into lanes made a strictly sequential run report
 * concurrency it never had.
 */
function leaves(bars: TimelineBarData[]): TimelineBarData[] {
  const out: TimelineBarData[] = [];
  for (const bar of bars) {
    if (bar.isRoot) {
      for (const child of bar.children ?? []) {
        if (!child.isPlatform) out.push(child);
      }
    } else if (!bar.isPlatform) {
      out.push(bar);
    }
  }
  return out;
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
  /** How many lanes were needed — the run's maximum concurrency. */
  rows: number;
};

/** A step drawn from more than one attempt is one that failed and came back. */
function wasRetried(bar: TimelineBarData): boolean {
  const attempts = bar.children?.filter((child) => /^Attempt \d+$/.test(child.name)) ?? [];
  return attempts.length > 1;
}

export function packMinimap(bars: TimelineBarData[], scale: TimeScale): Minimap {
  const steps = leaves(bars)
    .map((bar) => ({
      bar,
      startMs: bar.startTime.getTime(),
      endMs: (bar.endTime ?? bar.startTime).getTime(),
    }))
    .sort((a, b) => a.startMs - b.startMs);

  /** When each row last became free. */
  const rowFreeAt: number[] = [];
  const marks: MinimapMark[] = [];

  for (const step of steps) {
    let row = rowFreeAt.findIndex((freeAt) => freeAt <= step.startMs);
    if (row === -1) {
      row = rowFreeAt.length;
      rowFreeAt.push(0);
    }
    rowFreeAt[row] = step.endMs;

    const startPercent = scale.toPercent(step.startMs);
    marks.push({
      id: step.bar.id,
      row,
      startPercent,
      // A floor, so a 1ms step in a two-second run is still a mark rather than
      // nothing. The minimap is for finding things, not for measuring them.
      widthPercent: Math.max(0.35, scale.toPercent(step.endMs) - startPercent),
      state: toState(step.bar.status),
      name: step.bar.name,
      recovered: wasRetried(step.bar),
    });
  }

  return { marks, rows: Math.max(1, rowFreeAt.length) };
}
