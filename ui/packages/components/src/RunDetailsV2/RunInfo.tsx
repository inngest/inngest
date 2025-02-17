import type { Route } from 'next';
import { RiArrowRightUpLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { parseAIOutput } from '../AI/utils';
import { CancelRunButton } from '../CancelRunButton';
import { Card } from '../Card';
import {
  ElementWrapper,
  IDElement,
  LinkElement,
  OptimisticElementWrapper,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { Link } from '../Link';
import { RerunButton } from '../RerunButtonV2';
import { RunResult } from '../RunResult';
import type { Run as InitialRunData } from '../RunsPage/types';
import { AICell } from '../Table/Cell';
import type { Result } from '../types/functionRun';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';

type Props = {
  standalone: boolean;
  cancelRun: () => Promise<unknown>;
  className?: string;
  pathCreator: {
    app: (params: { externalAppID: string }) => Route;
    function: (params: { functionSlug: string }) => Route;
    runPopout: (params: { runID: string }) => Route;
  };
  rerun: (args: { fnID: string; runID: string }) => Promise<unknown>;
  initialRunData?: InitialRunData;
  run: Lazy<Run>;
  runID: string;
  result?: Result;
};

type Run = {
  app: {
    externalID: string;
    name: string;
  };
  fn: {
    id: string;
    name: string;
    slug: string;
  };
  id: string;
  trace: {
    childrenSpans?: unknown[];
    endedAt: string | null;
    queuedAt: string;
    startedAt: string | null;
    status: string;
    stepID?: string | null;
  };
  hasAI: boolean;
};

export function RunInfo({
  cancelRun,
  className,
  pathCreator,
  rerun,
  initialRunData,
  run,
  runID,
  standalone,
  result,
}: Props) {
  let allowCancel = false;
  let isSuccess = false;
  let stepID = null;

  if (isLazyDone(run)) {
    allowCancel = !Boolean(run.trace.endedAt);
    isSuccess = run.trace.status === 'COMPLETED';
    stepID = run.trace.stepID;
  }

  const aiOutput = result?.data ? parseAIOutput(result.data) : undefined;

  return (
    <div className={cn('flex flex-col gap-5', className)}>
      <Card>
        <Card.Header className="h-11 flex-row items-center gap-2">
          <div className="text-basis flex grow items-center gap-2">
            Run details{' '}
            {!standalone && (
              <Link
                size="medium"
                href={pathCreator.runPopout({ runID })}
                iconAfter={<RiArrowRightUpLine className="h-4 w-4 shrink-0" />}
              />
            )}
          </div>

          <CancelRunButton disabled={!allowCancel} onClick={cancelRun} />
          <RerunButton
            disabled={!isLazyDone(run)}
            onClick={async () => {
              if (!isLazyDone(run)) {
                return;
              }
              await rerun({ fnID: run.fn.id, runID });
            }}
          />
        </Card.Header>

        <Card.Content>
          <div>
            <dl className="flex flex-wrap gap-4">
              <ElementWrapper label="Run ID">
                <IDElement>{runID}</IDElement>
              </ElementWrapper>

              <OptimisticElementWrapper
                label="App"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) => <>{initialRun.app.name}</>}
              >
                {(run: Run) => {
                  return (
                    <LinkElement href={pathCreator.app({ externalAppID: run.app.externalID })}>
                      {run.app.name}
                    </LinkElement>
                  );
                }}
              </OptimisticElementWrapper>

              <OptimisticElementWrapper
                label="Function"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) => <>{initialRun.function.name}</>}
              >
                {(run: Run) => {
                  return (
                    <LinkElement href={pathCreator.function({ functionSlug: run.fn.slug })}>
                      {run.hasAI ? <AICell>{run.fn.name}</AICell> : run.fn.name}
                    </LinkElement>
                  );
                }}
              </OptimisticElementWrapper>

              <OptimisticElementWrapper
                label="Duration"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) => <TextElement>-</TextElement>}
              >
                {(run: Run) => {
                  let durationText = '-';

                  const startedAt = toMaybeDate(run.trace.startedAt);
                  if (startedAt) {
                    durationText = formatMilliseconds(
                      (toMaybeDate(run.trace.endedAt) ?? new Date()).getTime() - startedAt.getTime()
                    );
                  }

                  return <TextElement>{durationText}</TextElement>;
                }}
              </OptimisticElementWrapper>

              <OptimisticElementWrapper
                label="Queued at"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) =>
                  initialRun.queuedAt ? (
                    <TimeElement date={new Date(initialRun.queuedAt)} />
                  ) : (
                    <TextElement>-</TextElement>
                  )
                }
              >
                {(run: Run) => {
                  return <TimeElement date={new Date(run.trace.queuedAt)} />;
                }}
              </OptimisticElementWrapper>

              <OptimisticElementWrapper
                label="Started at"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) =>
                  initialRun?.status === 'QUEUED' ? <TextElement>-</TextElement> : null
                }
              >
                {(run: Run) => {
                  const startedAt = toMaybeDate(run.trace.startedAt);
                  if (!startedAt) {
                    return <TextElement>-</TextElement>;
                  }
                  return <TimeElement date={startedAt} />;
                }}
              </OptimisticElementWrapper>

              <OptimisticElementWrapper
                label="Ended at"
                lazy={run}
                initial={initialRunData}
                optimisticChildren={(initialRun: InitialRunData) =>
                  initialRun?.status === 'QUEUED' ? <TextElement>-</TextElement> : null
                }
              >
                {(run: Run) => {
                  const endedAt = toMaybeDate(run.trace.endedAt);
                  if (!endedAt) {
                    return <TextElement>-</TextElement>;
                  }
                  return <TimeElement date={endedAt} />;
                }}
              </OptimisticElementWrapper>
              {aiOutput && <AITrace aiOutput={aiOutput} />}
            </dl>
          </div>
        </Card.Content>
        {result ? (
          <RunResult
            className="border-muted border-t"
            result={result}
            runID={runID}
            stepID={stepID}
            isSuccess={isSuccess}
          />
        ) : (
          !isLazyDone(run) &&
          (initialRunData?.status === 'QUEUED' ? (
            <div className="border-muted bg-canvas border-t">
              <div className="border-l-status-queued flex items-center justify-start border-l-4">
                <span className="relative ml-4 flex h-2.5 w-2.5">
                  <span className="bg-status-queued absolute inline-flex h-full w-full animate-ping rounded-full opacity-75"></span>
                  <span className="bg-status-queued relative inline-flex h-2.5 w-2.5 rounded-full"></span>
                </span>
                <p className="text-subtle max-h-24 text-ellipsis break-words py-2.5 pl-3 text-sm">
                  Queued run awaiting start...
                </p>
              </div>
            </div>
          ) : null)
        )}
      </Card>
    </div>
  );
}
