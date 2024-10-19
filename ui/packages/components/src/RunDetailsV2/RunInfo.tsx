import type { Route } from 'next';

import { CancelRunButton } from '../CancelRunButton';
import { Card } from '../Card';
import {
  ElementWrapper,
  IDElement,
  LazyElementWrapper,
  LinkElement,
  OptimisticElementWrapper,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { Link } from '../Link';
import { RerunButton } from '../RerunButtonV2';
import { RunResult } from '../RunResult';
import type { Run as InitialRunData } from '../RunsPage/types';
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
  initialRunData: InitialRunData;
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
  };
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
  if (isLazyDone(run)) {
    allowCancel = !Boolean(run.trace.endedAt);
    isSuccess = run.trace.status === 'COMPLETED';
  }

  return (
    <div className={cn('flex flex-col gap-5', className)}>
      <Card>
        <Card.Header className="h-11 flex-row items-center gap-2">
          <div className="text-basis flex grow items-center gap-2">
            Run details {!standalone && <Link href={pathCreator.runPopout({ runID })} />}
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
                    <LinkElement
                      internalNavigation
                      href={pathCreator.app({ externalAppID: run.app.externalID })}
                      showIcon={false}
                    >
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
                    <LinkElement
                      internalNavigation
                      href={pathCreator.function({ functionSlug: run.fn.slug })}
                      showIcon={false}
                    >
                      {run.fn.name}
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
                  initialRun.queuedAt ?? <TimeElement date={new Date(initialRun.queuedAt)} />
                }
              >
                {(run: Run) => {
                  return <TimeElement date={new Date(run.trace.queuedAt)} />;
                }}
              </OptimisticElementWrapper>

              <LazyElementWrapper label="Started at" lazy={run}>
                {(run: Run) => {
                  const startedAt = toMaybeDate(run.trace.startedAt);
                  if (!startedAt) {
                    return <TextElement>-</TextElement>;
                  }
                  return <TimeElement date={startedAt} />;
                }}
              </LazyElementWrapper>

              <LazyElementWrapper label="Ended at" lazy={run}>
                {(run: Run) => {
                  const endedAt = toMaybeDate(run.trace.endedAt);
                  if (!endedAt) {
                    return <TextElement>-</TextElement>;
                  }
                  return <TimeElement date={endedAt} />;
                }}
              </LazyElementWrapper>
            </dl>
          </div>
        </Card.Content>
        {!result &&
          !isLazyDone(run) &&
          (initialRunData.status === 'QUEUED' ? (
            <div className="border-muted bg-canvas border-t">
              <div className="border-l-status-queued flex items-center justify-between border-l-4">
                <p className="text-subtle max-h-24 text-ellipsis break-words px-5 py-2.5 text-sm">
                  Awaiting run start...
                </p>
              </div>
            </div>
          ) : (
            ''
          ))}
        {result && (
          <RunResult className="border-muted border-t" result={result} isSuccess={isSuccess} />
        )}
      </Card>
    </div>
  );
}
