import { cn } from '../utils/classNames';
import { formatMilliseconds } from '../utils/date';
import { StepStatus, isStepStatus } from './types';
import { createSpanWidths, toMaybeDate } from './utils';

type Props = {
  className?: string;
  isInline?: boolean;
  maxTime: Date;
  minTime: Date;
  trace: {
    endedAt: string | null;
    id: string;
    queuedAt: string;
    startedAt: string | null;
    status: string;
  };
};

export function Span({ className, isInline, maxTime, minTime, trace }: Props) {
  const widths = createSpanWidths({
    maxTime,
    minTime,
    trace: {
      endedAt: trace.endedAt ? new Date(trace.endedAt) : null,
      queuedAt: new Date(trace.queuedAt),
      startedAt: trace.startedAt ? new Date(trace.startedAt) : null,
    },
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

const statusColors: { [key in StepStatus | 'UNKNOWN']: string } = {
  [StepStatus.Cancelled]: 'bg-slate-400',
  [StepStatus.Failed]: 'bg-rose-600',
  [StepStatus.Queued]: 'bg-amber-500',
  [StepStatus.Running]: 'bg-sky-500',
  [StepStatus.Succeeded]: 'bg-teal-500',
  UNKNOWN: 'bg-slate-500',
};

function getStatusColor(status: string): string {
  if (isStepStatus(status)) {
    return statusColors[status];
  }
  return statusColors['UNKNOWN'];
}
