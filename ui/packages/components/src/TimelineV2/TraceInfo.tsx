import type { Route } from 'next';

import { AITrace } from '../AI/AITrace';
import type { ExperimentalAI } from '../AI/utils';
import { Card } from '../Card';
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { RunResult } from '../RunResult';
import { Time } from '../Time';
import type { Result } from '../types/functionRun';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { isStepInfoInvoke, isStepInfoSleep, isStepInfoWait, type Trace } from './types';

type Props = {
  className?: string;
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  trace: Trace;
  runID: string;
  result?: Result;
  aiOutput?: ExperimentalAI;
};

export function TraceInfo({ className, pathCreator, trace, result, runID, aiOutput }: Props) {
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
        <ElementWrapper label="Run">
          {trace.stepInfo.runID ? (
            <LinkElement href={pathCreator.runPopout({ runID: trace.stepInfo.runID })}>
              {trace.stepInfo.runID}
            </LinkElement>
          ) : (
            '-'
          )}
        </ElementWrapper>
        <ElementWrapper label="Timeout">
          {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
        </ElementWrapper>
        <ElementWrapper label="Timed out">
          <TextElement>{maybeBooleanToString(trace.stepInfo.timedOut)}</TextElement>
        </ElementWrapper>
      </>
    );
  } else if (isStepInfoSleep(trace.stepInfo)) {
    const sleepUntil = toMaybeDate(trace.stepInfo.sleepUntil);
    stepKindInfo = (
      <ElementWrapper label="Sleep until">
        {sleepUntil ? <Time value={sleepUntil} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
    );
  } else if (isStepInfoWait(trace.stepInfo)) {
    const timeout = toMaybeDate(trace.stepInfo.timeout);
    stepKindInfo = (
      <>
        <ElementWrapper label="Event name">
          <TextElement>{trace.stepInfo.eventName}</TextElement>
        </ElementWrapper>
        <ElementWrapper label="Timeout">
          {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
        </ElementWrapper>
        <ElementWrapper label="Timed out">
          <TextElement>{maybeBooleanToString(trace.stepInfo.timedOut)}</TextElement>
        </ElementWrapper>
        <ElementWrapper className="w-full" label="Match expression">
          {trace.stepInfo.expression ? (
            <CodeElement value={trace.stepInfo.expression} />
          ) : (
            <TextElement>-</TextElement>
          )}
        </ElementWrapper>
      </>
    );
  }

  return (
    <div className={cn('flex', className)}>
      <Card>
        <Card.Header className="h-11 flex-row items-center gap-2">Step information</Card.Header>

        <Card.Content>
          <dl className="flex flex-wrap gap-4">
            <ElementWrapper label="Queued at">
              <TimeElement date={new Date(trace.queuedAt)} />
            </ElementWrapper>

            <ElementWrapper label="Started at">
              {trace.startedAt ? (
                <TimeElement date={new Date(trace.startedAt)} />
              ) : (
                <TextElement>-</TextElement>
              )}
            </ElementWrapper>

            <ElementWrapper label="Ended at">
              {trace.endedAt ? (
                <TimeElement date={new Date(trace.endedAt)} />
              ) : (
                <TextElement>-</TextElement>
              )}
            </ElementWrapper>

            <ElementWrapper label="Delay">
              <TextElement>{delayText}</TextElement>
            </ElementWrapper>

            <ElementWrapper label="Duration">
              <TextElement>{durationText}</TextElement>
            </ElementWrapper>

            {stepKindInfo}

            {aiOutput && <AITrace aiOutput={aiOutput} />}
          </dl>
        </Card.Content>
        {result && (
          <RunResult
            className="border-subtle border-t"
            result={result}
            runID={runID}
            stepID={trace.stepID ?? undefined}
          />
        )}
      </Card>
    </div>
  );
}

function maybeBooleanToString(value: boolean | null): string | null {
  if (value === null) {
    return null;
  }
  return value ? 'True' : 'False';
}
