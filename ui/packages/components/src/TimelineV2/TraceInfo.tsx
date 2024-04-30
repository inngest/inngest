import { Card } from '../Card';
import { Time } from '../Time';
import { cn } from '../utils/classNames';
import { formatMilliseconds } from '../utils/date';
import type { Trace } from './types';
import { toMaybeDate } from './utils';

type Props = {
  className?: string;
  trace: Trace;
};

export function TraceInfo({ className, trace }: Props) {
  const delayText = formatMilliseconds(
    (toMaybeDate(trace.startedAt) ?? new Date()).getTime() - new Date(trace.queuedAt).getTime()
  );

  let duration = 0;
  (trace.childrenSpans ?? []).forEach((child, i) => {
    if (!child.startedAt) {
      return;
    }

    duration +=
      (toMaybeDate(child.endedAt) ?? new Date()).getTime() - new Date(child.startedAt).getTime();
  });

  let durationText = '-';
  if (duration > 0) {
    durationText = formatMilliseconds(duration);
  }

  return (
    <div className={cn('flex bg-white', className)}>
      <Card>
        <Card.Content>
          <dl className="flex flex-wrap gap-2">
            <Labeled label="Queued at">
              <Time value={new Date(trace.queuedAt)} />
            </Labeled>

            <Labeled label="Started at">
              {trace.startedAt ? <Time value={new Date(trace.startedAt)} /> : '-'}
            </Labeled>

            <Labeled label="Ended at">
              {trace.endedAt ? <Time value={new Date(trace.endedAt)} /> : '-'}
            </Labeled>

            <Labeled label="Delay">{delayText}</Labeled>

            <Labeled label="Duration">{durationText}</Labeled>
          </dl>
        </Card.Content>
      </Card>
    </div>
  );
}

function Labeled({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <div className="w-64 text-sm">
      <dt className="text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}
