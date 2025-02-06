'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { ErrorCard } from '../RunDetailsV2/ErrorCard';
import type { Run as InitialRunData } from '../RunsPage/types';
import { StatusCell } from '../Table/Cell';
import { Trace as OldTrace } from '../TimelineV2';
import { TriggerDetails } from '../TriggerDetails';
import type { Result } from '../types/functionRun';
import { nullishToLazy } from '../utils/lazyLoad';
import { RunInfo } from './RunInfo';
import { StepInfo } from './StepInfo';
import { Tabs } from './Tabs';
import { Timeline } from './Timeline';
import { TopInfo } from './TopInfo';
import { Workflow } from './Workflow';
import { useStepSelection } from './utils';

type Props = {
  standalone: boolean;
  cancelRun: (runID: string) => Promise<unknown>;
  getResult: (outputID: string) => Promise<Result>;
  getRun: (runID: string) => Promise<Run>;
  initialRunData?: InitialRunData;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pathCreator: React.ComponentProps<typeof RunInfo>['pathCreator'];
  pollInterval?: number;
  rerun: React.ComponentProps<typeof RunInfo>['rerun'];
  runID: string;
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
  trace: React.ComponentProps<typeof OldTrace>['trace'];
  hasAI: boolean;
};

export const RunDetailsV3 = (props: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const { getResult, getRun, getTrigger, pathCreator, rerun, runID, standalone } = props;
  const [pollInterval, setPollInterval] = useState(props.pollInterval);
  const [leftWidth, setLeftWidth] = useState(55);
  const [isDragging, setIsDragging] = useState(false);
  const { selectedStep } = useStepSelection();

  const handleMouseDown = useCallback(() => {
    setIsDragging(true);
  }, []);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging) {
        return;
      }

      const container = containerRef.current;
      if (!container) {
        return;
      }

      const containerRect = container.getBoundingClientRect();
      const newWidth = ((e.clientX - containerRect.left) / containerRect.width) * 100;
      setLeftWidth(Math.min(Math.max(newWidth, 20), 80));
    },
    [isDragging]
  );

  useEffect(() => {
    if (isDragging) {
      document.body.style.userSelect = 'none';
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
    }
    return () => {
      document.body.style.userSelect = '';
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  const runRes = useQuery({
    queryKey: ['run', runID],
    queryFn: useCallback(() => {
      return getRun(runID);
    }, [getRun, runID]),
    retry: 3,
    refetchInterval: pollInterval,
  });

  const outputID = runRes?.data?.trace.outputID;
  const resultRes = useQuery({
    enabled: Boolean(outputID),
    queryKey: ['run-result', runID],
    queryFn: useCallback(() => {
      if (!outputID) {
        // Unreachable
        throw new Error('missing outputID');
      }

      return getResult(outputID);
    }, [getResult, outputID]),
  });

  const cancelRun = useCallback(async () => {
    return await props.cancelRun(runID);
  }, [props.cancelRun]);

  const run = runRes.data;
  if (run?.trace.endedAt && pollInterval) {
    // Stop polling since ended runs are immutable
    setPollInterval(undefined);
  }

  // Do not show the error if queued and the error is no spans
  const isNoSpansFoundError = !!runRes.error?.toString().match(/no function run span found/gi);
  const showError =
    props.initialRunData?.status === 'QUEUED' && isNoSpansFoundError
      ? false
      : runRes.error || resultRes.error;

  return (
    <>
      {standalone && run && (
        <div className="border-muted mb-2 flex flex-col gap-1 border-b px-4 pb-4">
          <StatusCell status={run.trace.status} />
          <p className="text-basis text-2xl font-medium">{run.fn.name}</p>
          <p className="text-subtle font-mono">{runID}</p>
        </div>
      )}
      <div ref={containerRef} className="flex h-full flex-row">
        <div className="flex flex-col gap-2" style={{ width: `${leftWidth}%` }}>
          <div className="px-4">
            <RunInfo
              cancelRun={cancelRun}
              className="mb-4"
              pathCreator={pathCreator}
              rerun={rerun}
              initialRunData={props.initialRunData}
              run={nullishToLazy(run)}
              runID={runID}
              standalone={standalone}
              result={resultRes.data}
            />
            {showError && (
              <ErrorCard
                error={runRes.error || resultRes.error}
                reset={runRes.error ? () => runRes.refetch() : () => resultRes.refetch()}
              />
            )}
          </div>
          <Tabs
            tabs={[
              {
                label: 'Trace',
                node: run && (
                  <Timeline
                    getResult={getResult}
                    pathCreator={pathCreator}
                    runID={runID}
                    trace={run?.trace}
                  />
                ),
              },
              { label: 'Workflow', node: <Workflow /> },
            ]}
          />
        </div>

        <div
          className="border-muted w-2 cursor-col-resize border-r-[.5px]"
          onMouseDown={handleMouseDown}
        />

        <div className="border-muted flex h-full flex-col" style={{ width: `${100 - leftWidth}%` }}>
          {selectedStep ? (
            <StepInfo selectedStep={selectedStep} />
          ) : (
            <TopInfo
              slug={run?.fn.slug}
              getTrigger={getTrigger}
              runID={runID}
              result={resultRes.data}
            />
          )}
        </div>
      </div>
    </>
  );
};
