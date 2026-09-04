/**
 * A thin full-width strip above the views, sharing their axis.
 *
 * This is what loads first, and its job is to answer "where is the problem"
 * before anyone scrolls or zooms. Think the severity histogram above a Google
 * Cloud Logs search: buckets across the run, each a small stacked bar of step
 * states, most serious at the bottom so a red smear is visible against the
 * baseline rather than floating somewhere in the middle of a column.
 *
 * It renders the whole run always, however much is collapsed below it, and it
 * takes the same `TimeScale` the rows take — including its breaks — so a column
 * really does sit above the rows it describes.
 */
import { type JSX } from 'react';

import { cn } from '../utils/classNames';
import { formatDuration } from './runDetailsUtils';
import { DENSITY_ORDER, densityPeak, type DensityBucket, type DensityState } from './utils/density';

/**
 * Neutral for the ordinary case and colour only where it carries meaning, so
 * the eye goes to failure first and to nothing else. All existing status
 * tokens; no new colour values.
 */
const STATE_CLASS: Record<DensityState, string> = {
  failed: 'bg-status-failed',
  cancelled: 'bg-status-cancelled',
  running: 'bg-status-running',
  waiting: 'bg-status-queued',
  completed: 'bg-status-completed',
};

const STATE_LABEL: Record<DensityState, string> = {
  failed: 'failed',
  cancelled: 'cancelled',
  running: 'running',
  waiting: 'waiting',
  completed: 'succeeded',
};

type Props = {
  buckets: DensityBucket[];
  /** Total height of the strip in pixels. */
  height?: number;
  /** Called with a bucket's time range when one is clicked. */
  onSelectRange?: (startMs: number, endMs: number) => void;
  /** Run start, so tooltips can report an offset rather than a wall clock. */
  minMs: number;
};

export function DensityStrip({
  buckets,
  height = 22,
  onSelectRange,
  minMs,
}: Props): JSX.Element | null {
  if (!buckets.length) return null;

  const peak = densityPeak(buckets);
  if (peak === 0) return null;

  return (
    <div
      data-testid="density-strip"
      className="pointer-events-none absolute inset-x-0 bottom-0"
      style={{ height }}
      aria-hidden={false}
      role="img"
      aria-label={`Step activity across the run, ${buckets.length} buckets`}
    >
      {buckets.map((bucket) => {
        if (bucket.total === 0) return null;

        const summary = DENSITY_ORDER.filter((state) => bucket.counts[state] > 0)
          .map((state) => `${bucket.counts[state]} ${STATE_LABEL[state]}`)
          .join(', ');

        return (
          <div
            key={bucket.startPercent}
            className={cn(
              'absolute bottom-0 flex flex-col-reverse',
              onSelectRange && 'pointer-events-auto cursor-pointer'
            )}
            style={{
              left: `${bucket.startPercent}%`,
              // A hairline between columns keeps them countable without a grid.
              width: `calc(${bucket.endPercent - bucket.startPercent}% - 1px)`,
              height: `${(bucket.total / peak) * 100}%`,
            }}
            onClick={onSelectRange ? () => onSelectRange(bucket.startMs, bucket.endMs) : undefined}
            title={`+${formatDuration(bucket.startMs - minMs)} — ${summary}`}
          >
            {/* Bottom-up, most serious first: failure sits on the baseline
                where it is easiest to see across a wide strip. */}
            {DENSITY_ORDER.map((state) =>
              bucket.counts[state] > 0 ? (
                <div
                  key={state}
                  className={STATE_CLASS[state]}
                  style={{ height: `${(bucket.counts[state] / bucket.total) * 100}%` }}
                />
              ) : null
            )}
          </div>
        );
      })}
    </div>
  );
}
