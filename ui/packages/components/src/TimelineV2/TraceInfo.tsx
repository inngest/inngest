import type { Route } from 'next';

import { Card } from '../Card';
import { InlineCode } from '../InlineCode';
import { Link } from '../Link';
import { Time } from '../Time';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { isStepInfoInvoke, isStepInfoSleep, isStepInfoWait, type Trace } from './types';

type Props = {
  className?: string;
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  trace: Trace;
};

export function TraceInfo({ className, pathCreator, trace }: Props) {
  const delayText = formatMilliseconds(
    (toMaybeDate(trace.startedAt) ?? new Date()).getTime() - new Date(trace.queuedAt).getTime()
  );

  let duration = 0;
  if (trace.childrenSpans && trace.childrenSpans.length > 0) {
    trace.childrenSpans.forEach((child, i) => {
      if (!child.startedAt) {
        return;
      }

      duration +=
        (toMaybeDate(child.endedAt) ?? new Date()).getTime() - new Date(child.startedAt).getTime();
    });
  } else if (trace.startedAt) {
    duration =
      (toMaybeDate(trace.endedAt) ?? new Date()).getTime() - new Date(trace.startedAt).getTime();
  }

  let durationText = '-';
  if (duration > 0) {
    durationText = formatMilliseconds(duration);
  }

  let stepKindInfo = null;

  if (isStepInfoInvoke(trace.stepInfo)) {
    const timeout = toMaybeDate(trace.stepInfo.timeout);
    stepKindInfo = (
      <>
        <Labeled label="Run">
          {trace.stepInfo.runID ? (
            <Link
              href={pathCreator.runPopout({ runID: trace.stepInfo.runID })}
              internalNavigation={false}
            >
              {trace.stepInfo.runID}
            </Link>
          ) : (
            '-'
          )}
        </Labeled>
        <Labeled label="Timeout">{timeout ? <Time value={timeout} /> : '-'}</Labeled>
        <Labeled label="Timed out">{booleanToString(trace.stepInfo.timedOut)}</Labeled>
      </>
    );
  } else if (isStepInfoSleep(trace.stepInfo)) {
    const sleepUntil = toMaybeDate(trace.stepInfo.sleepUntil);
    stepKindInfo = (
      <Labeled label="Sleep until">{sleepUntil ? <Time value={sleepUntil} /> : '-'}</Labeled>
    );
  } else if (isStepInfoWait(trace.stepInfo)) {
    const timeout = toMaybeDate(trace.stepInfo.timeout);
    stepKindInfo = (
      <>
        <Labeled label="Event name">{trace.stepInfo.eventName}</Labeled>
        <Labeled label="Timeout">{timeout ? <Time value={timeout} /> : '-'}</Labeled>
        <Labeled label="Timed out">{booleanToString(trace.stepInfo.timedOut)}</Labeled>
        <Labeled className="w-full" label="Match expression">
          {trace.stepInfo.expression ? <InlineCode value={trace.stepInfo.expression} /> : '-'}
        </Labeled>
      </>
    );
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

            {stepKindInfo}
          </dl>
        </Card.Content>
      </Card>
    </div>
  );
}

function Labeled({
  className,
  label,
  children,
}: React.PropsWithChildren<{ className?: string; label: string }>) {
  return (
    <div className={cn('w-64 text-sm', className)}>
      <dt className="text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}

function booleanToString(value: boolean) {
  return value ? 'True' : 'False';
}
