import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { SegmentedTimelineSpan } from './SegmentedTimelineSpan';
import { isStepRunSpan } from './timingBreakdown';
import type { Trace } from './types';
import { createSpanWidths } from './utils';

type Props = {
  className?: string;
  isInline?: boolean;
  maxTime: Date;
  minTime: Date;
  span: Trace;
};

export function Span({ className, isInline, maxTime, minTime, span: trace }: Props) {
  // US2: Use segmented timeline span for step.run spans (EXE-1217)
  if (isStepRunSpan(trace) && !trace.isUserland) {
    return (
      <SegmentedTimelineSpan
        className={className}
        isInline={isInline}
        maxTime={maxTime}
        minTime={minTime}
        span={trace}
      />
    );
  }

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

  return (
    <div
      className={cn('flex flex-grow items-center', className)}
      style={{
        flexGrow: totalWidth,
      }}
    >
      {/* Gray line to the left of the span */}
      <div className="bg-contrast h-px" style={{ flexGrow: widths.before }} />

      {/* Queued part of the span */}
      {widths.queued > 0 && (
        <div
          className="bg-surfaceSubtle dark:bg-surfaceMuted h-1"
          style={{ flexGrow: widths.queued }}
        />
      )}

      {/* Running part of the span */}
      {widths.running > 0 && (
        <div
          className={cn(
            'z-0 h-5 rounded-sm transition-shadow',
            trace.isUserland ? 'bg-quaternary-coolxSubtle' : getStatusBackgroundClass(trace.status),
            trace.isUserland ? 'border-quaternary-coolxSubtle' : getStatusBorderClass(trace.status)
          )}
          style={{ flexGrow: widths.running }}
        />
      )}

      {/* Gray line to the right of the span */}
      <div className="bg-contrast h-px" style={{ flexGrow: widths.after }}></div>
    </div>
  );
}
