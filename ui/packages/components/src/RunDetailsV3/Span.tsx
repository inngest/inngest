import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { createSpanWidths } from './utils';

type Props = {
  className?: string;
  isInline?: boolean;
  maxTime: Date;
  minTime: Date;
  trace: {
    endedAt: string | null;
    queuedAt: string;
    spanID: string;
    startedAt: string | null;
    status: string;
  };
};

export function Span({ className, isInline, maxTime, minTime, trace }: Props) {
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
      <div className="bg-contrast h-px" style={{ flexGrow: widths.before }}></div>

      {/* Queued part of the span */}
      {widths.queued > 0 && (
        <div
          className="bg-surfaceSubtle dark:bg-surfaceMuted h-2"
          style={{ flexGrow: widths.queued }}
        ></div>
      )}

      {/* Running part of the span */}
      {widths.running > 0 && (
        <div
          className={cn(
            'h-5 rounded transition-shadow hover:shadow-lg',
            getStatusBackgroundClass(trace.status),
            getStatusBorderClass(trace.status)
          )}
          style={{ flexGrow: widths.running }}
        ></div>
      )}

      {/* Gray line to the right of the span */}
      <div className="bg-contrast h-px" style={{ flexGrow: widths.after }}></div>
    </div>
  );
}
