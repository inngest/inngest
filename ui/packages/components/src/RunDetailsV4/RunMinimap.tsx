/**
 * A minimap of the whole run, sitting inside the scrubber.
 *
 * This replaced a stacked histogram, which failed for a reason worth recording:
 * it was made of the same marks, in the same colours, directly above the trace's
 * own bars, so it read as *more trace* rather than as an overview of one. Two
 * things that look alike and mean different things is worse than either alone,
 * and it sat exactly where the eye lands first.
 *
 * A minimap has a different job and now has a different shape. Steps are packed
 * into as few rows as possible — sequential work shares one line, and only
 * genuine parallelism forces another — so the **row count is the concurrency
 * profile**, which neither the trace nor the canvas shows. Marks are hairline
 * thin, unlabelled, and visibly a different kind of object from the bars below.
 *
 * It always draws the whole run, however much is collapsed underneath: the map
 * must not shrink because the territory got folded.
 */
import { type JSX } from 'react';

import { cn } from '../utils/classNames';
import { formatDuration } from './runDetailsUtils';
import type { DensityState, Minimap } from './utils/density';

/**
 * Colour only where it carries meaning. Anything that succeeded is deliberately
 * quiet: at this size the job is to make failure findable, not to render a
 * second copy of the trace's palette.
 */
const STATE_CLASS: Record<DensityState, string> = {
  failed: 'bg-status-failed',
  cancelled: 'bg-status-cancelled',
  running: 'bg-status-running',
  waiting: 'bg-status-queued',
  completed: 'bg-status-completed',
};

/**
 * The minimap is always the same height, whatever the run.
 *
 * A strip that grew with concurrency shoved the whole trace down the page on
 * wide runs and made the header jump between fixtures — the overview is a fixed
 * piece of furniture, so it gets a fixed size. Lanes get thinner to fit instead.
 */
export const MINIMAP_HEIGHT_PX = 16;

/** Thin enough that a mark cannot be mistaken for a bar in the trace. */
const MAX_ROW_PX = 3;

/**
 * Lane geometry for a given number of lanes.
 *
 * Lanes never get CHUNKIER than 3px — a fat minimap starts to look like the
 * trace again, which is the thing it exists not to look like — they only get
 * thinner, down to a 1px hairline, and the gap between them is the first thing
 * surrendered when space runs short.
 */
function laneGeometry(rows: number): { rowPx: number; gapPx: number } {
  const gapPx = rows <= 8 ? 1 : 0;
  const available = MINIMAP_HEIGHT_PX - Math.max(0, rows - 1) * gapPx;
  const rowPx = Math.max(1, Math.min(MAX_ROW_PX, Math.floor(available / rows)));
  return { rowPx, gapPx };
}

type Props = {
  minimap: Minimap;
  /** Run start, so a tooltip can report an offset rather than a wall clock. */
  minMs: number;
  /** Span hovered anywhere in the run; the matching mark lifts. */
  hoveredStepId?: string;
};

export function RunMinimap({ minimap, minMs, hoveredStepId }: Props): JSX.Element | null {
  if (!minimap.marks.length) return null;

  const { rowPx, gapPx } = laneGeometry(minimap.rows);

  return (
    <div
      data-testid="run-minimap"
      className="pointer-events-none absolute inset-x-0 bottom-0"
      style={{ height: MINIMAP_HEIGHT_PX }}
      role="img"
      aria-label={`Overview of the run: ${minimap.marks.length} steps across ${
        minimap.rows
      } concurrent ${minimap.rows === 1 ? 'lane' : 'lanes'}`}
    >
      {minimap.marks.map((mark) => (
        <div
          key={mark.id}
          className={cn(
            'absolute rounded-[1px]',
            STATE_CLASS[mark.state],
            // Succeeded work recedes; anything else keeps its full weight, so a
            // failure is the only thing that draws the eye.
            mark.state === 'completed' ? 'opacity-45' : 'opacity-95',
            hoveredStepId === mark.id &&
              '!opacity-100 ring-1 ring-[rgb(var(--color-border-contrast))]'
          )}
          style={{
            left: `${mark.startPercent}%`,
            width: `${mark.widthPercent}%`,
            top: mark.row * (rowPx + gapPx),
            height: rowPx,
          }}
          title={mark.name}
        />
      ))}
    </div>
  );
}

/** "12 steps · 3 at once · 1.3s" — what the map is, in one line. */
export function minimapSummary(minimap: Minimap, totalMs: number): string {
  const steps = `${minimap.marks.length} step${minimap.marks.length === 1 ? '' : 's'}`;
  const lanes = minimap.rows > 1 ? ` · up to ${minimap.rows} at once` : '';
  return `${steps}${lanes} · ${formatDuration(totalMs)}`;
}
