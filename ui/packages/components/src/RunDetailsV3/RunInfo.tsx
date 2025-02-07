import { useState } from 'react';
import type { Route } from 'next';
import { RiArrowRightUpLine, RiArrowUpSLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { parseAIOutput } from '../AI/utils';
import {
  ElementWrapper,
  IDElement,
  LinkElement,
  OptimisticElementWrapper,
  SkeletonElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { Link } from '../Link';
import type { Run as InitialRunData } from '../RunsPage/types';
import { AICell } from '../Table/Cell';
import type { Result } from '../types/functionRun';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { ActionsMenu } from './ActionMenu';

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

export const RunInfo = ({
  cancelRun,
  pathCreator,
  rerun,
  initialRunData,
  run,
  runID,
  standalone,
  result,
}: Props) => {
  const [expanded, setExpanded] = useState(true);
  let allowCancel = false;

  if (isLazyDone(run)) {
    allowCancel = !Boolean(run.trace.endedAt);
  }

  const aiOutput = result?.data ? parseAIOutput(result.data) : undefined;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none">
        <div className="text-basis flex items-center justify-start gap-2">
          <RiArrowUpSLine
            className={`cursor-pointer transition-transform duration-500 ${
              expanded ? 'rotate-180' : ''
            }`}
            onClick={() => setExpanded(!expanded)}
          />
          {isLazyDone(run) ? (
            <span className="text-basis text-sm font-normal">{run.fn.name}</span>
          ) : (
            <SkeletonElement />
          )}
          {!standalone && (
            <Link
              size="medium"
              href={pathCreator.runPopout({ runID })}
              iconAfter={<RiArrowRightUpLine className="h-4 w-4 shrink-0" />}
            />
          )}
        </div>

        <div className="flex items-center gap-2">
          <ActionsMenu
            cancel={cancelRun}
            reRun={async () => {
              if (!isLazyDone(run)) {
                return;
              }
              await rerun({ fnID: run.fn.id, runID });
            }}
            allowCancel={allowCancel}
          />
        </div>
      </div>

      {expanded && (
        <div>
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
        </div>
      )}
    </div>
  );
};
