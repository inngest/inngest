import { isFunctionRunStatus, type FunctionRunStatus } from '../types/functionRun';
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
      <div className="h-0.5 bg-slate-200" style={{ flexGrow: widths.before }}></div>

      {/* Queued part of the span */}
      <div className="h-2 bg-slate-500" style={{ flexGrow: widths.queued }}></div>

      {/* Running part of the span */}
      <div
        className={cn('h-5 rounded', getStatusColor(trace.status))}
        style={{ flexGrow: widths.running }}
      ></div>

      {/* Gray line to the right of the span */}
      <div className="h-0.5 bg-slate-200" style={{ flexGrow: widths.after }}></div>
    </div>
  );
}

const statusColors: { [key in FunctionRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'bg-slate-400',
  COMPLETED: 'bg-teal-500',
  FAILED: 'bg-rose-600',
  QUEUED: 'bg-amber-500',
  RUNNING: 'bg-sky-500',
  UNKNOWN: 'bg-slate-500',
};

function getStatusColor(status: string): string {
  if (isFunctionRunStatus(status)) {
    return statusColors[status];
  }
  return statusColors['UNKNOWN'];
}
