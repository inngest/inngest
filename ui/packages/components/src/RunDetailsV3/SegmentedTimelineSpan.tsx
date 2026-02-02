import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { TIMING_COLORS } from './timingBreakdown';
import type { Trace } from './types';
import { createSpanWidths } from './utils';

type Props = {
  className?: string;
  isInline?: boolean;
  maxTime: Date;
  minTime: Date;
  span: Trace;
};

/**
 * SegmentedTimelineSpan renders a step.run span in the timeline with
 * color-coded segments showing the proportion of time spent in each
 * timing category (INNGEST queue vs SERVER execution).
 *
 * US2: Identify Timing Categories at a Glance
 */
export function SegmentedTimelineSpan({
  className,
  isInline,
  maxTime,
  minTime,
  span: trace,
}: Props) {
  const widths = createSpanWidths({
    ended: toMaybeDate(trace.endedAt)?.getTime() ?? null,
    max: maxTime.getTime(),
    min: minTime.getTime(),
    queued: new Date(trace.queuedAt).getTime(),
    started: toMaybeDate(trace.startedAt)?.getTime() ?? null,
  });

  if (isInline) {
    widths.after = 0;
    widths.before = 0;
  }

  const totalWidth = widths.before + widths.queued + widths.running + widths.after;

  // Calculate timing proportions for the segmented bar
  const queuedAt = new Date(trace.queuedAt).getTime();
  const startedAt = toMaybeDate(trace.startedAt)?.getTime() ?? null;
  const endedAt = toMaybeDate(trace.endedAt)?.getTime() ?? Date.now();

  // Queue delay and execution time
  const queueDelay = startedAt ? Math.max(0, startedAt - queuedAt) : 0;
  const executionTime = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalDuration = queueDelay + executionTime;

  // Calculate percentages for the segmented bar
  const inngestPercent = totalDuration > 0 ? (queueDelay / totalDuration) * 100 : 0;
  const serverPercent = totalDuration > 0 ? (executionTime / totalDuration) * 100 : 100;

  return (
    <div
      className={cn('flex flex-grow items-center', className)}
      style={{
        flexGrow: totalWidth,
      }}
    >
      {/* Gray line to the left of the span */}
      <div className="bg-contrast h-px" style={{ flexGrow: widths.before }} />

      {/* Queued part - thin bar for waiting in queue (before execution starts) */}
      {widths.queued > 0 && !startedAt && (
        <div
          className="bg-surfaceSubtle dark:bg-surfaceMuted h-1"
          style={{ flexGrow: widths.queued }}
        />
      )}

      {/* Segmented running bar - shows inngest (queue delay) + server (execution) */}
      {(widths.running > 0 || (startedAt && widths.queued > 0)) && (
        <div
          className="z-0 flex h-5 overflow-hidden rounded-sm"
          style={{ flexGrow: widths.queued + widths.running }}
        >
          {/* Inngest queue segment (gray) */}
          {inngestPercent > 0 && (
            <div
              className={cn(TIMING_COLORS.inngest.base, 'transition-all duration-150')}
              style={{
                width: `${inngestPercent}%`,
                minWidth: inngestPercent > 0 ? '2px' : '0px',
              }}
            />
          )}

          {/* Server execution segment (green) */}
          {serverPercent > 0 && (
            <div
              className={cn(TIMING_COLORS.customer_server.base, 'transition-all duration-150')}
              style={{
                width: `${serverPercent}%`,
                minWidth: serverPercent > 0 ? '2px' : '0px',
              }}
            />
          )}
        </div>
      )}

      {/* Gray line to the right of the span */}
      <div className="bg-contrast h-px" style={{ flexGrow: widths.after }} />
    </div>
  );
}
