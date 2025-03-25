import { useState } from 'react';
import type { Route } from 'next';
import { RiArrowRightSLine, RiExternalLinkLine } from '@remixicon/react';

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
} from '../DetailsCard/NewElement';
import { Link } from '../Link';
import type { Run as InitialRunData } from '../RunsPage/types';
import { AICell } from '../Table/Cell';
import type { Result } from '../types/functionRun';
import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { Actions } from './Actions';
import { formatDuration } from './utils';

type Props = {
  standalone: boolean;
  className?: string;
  pathCreator: {
    app: (params: { externalAppID: string }) => Route;
    function: (params: { functionSlug: string }) => Route;
    runPopout: (params: { runID: string }) => Route;
  };
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

export const RunInfo = ({ pathCreator, initialRunData, run, runID, standalone, result }: Props) => {
  const [expanded, setExpanded] = useState(true);
  const allowCancel = isLazyDone(run) && !Boolean(run.trace.endedAt);

  const aiOutput = result?.data ? parseAIOutput(result.data) : undefined;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none">
        <div className="text-basis flex items-center justify-start gap-2">
          <div
            className="flex  cursor-pointer items-center gap-2"
            onClick={() => setExpanded(!expanded)}
          >
            <RiArrowRightSLine
              className={`shrink-0 transition-transform duration-[250ms] ${
                expanded ? 'rotate-90' : ''
              }`}
            />
            {isLazyDone(run) ? (
              <span className="text-basis text-sm font-normal">{run.fn.name}</span>
            ) : (
              <SkeletonElement />
            )}
          </div>
          {!standalone && (
            <Link
              size="medium"
              href={pathCreator.runPopout({ runID })}
              iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
            />
          )}
        </div>

        <div className="flex items-center gap-2">
          <Actions
            runID={runID}
            fnID={isLazyDone(run) ? run.fn.id : undefined}
            allowCancel={allowCancel}
          />
        </div>
      </div>

      {expanded && (
        <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4">
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

              const queuedAt = toMaybeDate(run.trace.queuedAt);
              if (queuedAt) {
                durationText = formatDuration(
                  (toMaybeDate(run.trace.endedAt) ?? new Date()).getTime() - queuedAt.getTime()
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
        </div>
      )}
    </div>
  );
};
