'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { ErrorCard } from '../Error/ErrorCard';
import type { Run as InitialRunData } from '../RunsPage/types';
import { StatusCell } from '../Table/Cell';
import { TriggerDetails } from '../TriggerDetails';
import { DragDivider } from '../icons/DragDivider';
import type { Result } from '../types/functionRun';
import { nullishToLazy } from '../utils/lazyLoad';
import { RunInfo } from './RunInfo';
import { StepInfo } from './StepInfo';
import { Tabs } from './Tabs';
import { Timeline } from './Timeline';
import { TopInfo } from './TopInfo';
import type { Trace } from './Trace';
import { Waiting } from './Waiting';
import { useStepSelection } from './utils';

type Props = {
  standalone: boolean;
  getResult: (outputID: string, preview?: boolean) => Promise<Result>;
  getRun: (runID: string, preview?: boolean) => Promise<Run>;
  initialRunData?: InitialRunData;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pollInterval?: number;
  runID: string;
  tracesPreviewEnabled?: boolean;
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
  trace: React.ComponentProps<typeof Trace>['trace'];
  hasAI: boolean;
};

const MIN_HEIGHT = 586;

export const RunDetailsV3 = (props: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const runInfoRef = useRef<HTMLDivElement>(null);
  const { getResult, getRun, getTrigger, runID, standalone } = props;
  const [pollInterval, setPollInterval] = useState(props.pollInterval);
  const [leftWidth, setLeftWidth] = useState(55);
  const [height, setHeight] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const { selectedStep } = useStepSelection(runID);

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
    //
    // left column height is dynamic and should determine right column height
    const h = leftColumnRef.current?.clientHeight ?? 0;
    setHeight(h > MIN_HEIGHT ? h : MIN_HEIGHT);
  }, [leftColumnRef.current?.clientHeight]);

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
    queryKey: ['run', runID, { preview: props.tracesPreviewEnabled }],
    queryFn: useCallback(() => {
      console.log('lollllyep', { 'props.tracesPreviewEnabled': props.tracesPreviewEnabled });

      return getRun(runID, props.tracesPreviewEnabled);
    }, [getRun, runID, props.tracesPreviewEnabled]),
    retry: 3,
    refetchInterval: pollInterval,
  });

  const outputID = runRes?.data?.trace.outputID;
  const resultRes = useQuery({
    enabled: Boolean(outputID),
    refetchInterval: pollInterval,
    queryKey: ['run-result', runID, { preview: props.tracesPreviewEnabled }],
    queryFn: useCallback(() => {
      if (!outputID) {
        // Unreachable
        throw new Error('missing outputID');
      }

      return getResult(outputID, props.tracesPreviewEnabled);
    }, [getResult, outputID, props.tracesPreviewEnabled]),
  });

  const run = runRes.data;
  if (run?.trace.endedAt && pollInterval) {
    //
    // Stop polling for ended runs, but still give it
    // a few seconds for any lingering userland traces.
    setTimeout(() => {
      setPollInterval(undefined);
    }, 6000);
  }

  // Do not show the error if queued and the error is no spans
  const noSpansFoundError = !!runRes.error?.toString().match(/no function run span found/gi);
  const waiting = props.initialRunData?.status === 'QUEUED' && noSpansFoundError;
  const showError = waiting ? false : runRes.error || resultRes.error;

  return (
    <>
      {standalone && run && (
        <div className="border-muted flex flex-row items-start justify-between border-b px-4 pb-4">
          <div className="flex flex-col gap-1">
            <StatusCell status={run.trace.status} />
            <p className="text-basis text-2xl font-medium">{run.fn.name}</p>
            <p className="text-subtle font-mono">{runID}</p>
          </div>
        </div>
      )}
      <div ref={containerRef} className="flex flex-row">
        <div ref={leftColumnRef} className="flex flex-col gap-2" style={{ width: `${leftWidth}%` }}>
          <div ref={runInfoRef} className="px-4">
            <RunInfo
              className="mb-4"
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
                id: 'trace',
                node: waiting ? (
                  <Waiting />
                ) : run ? (
                  <Timeline runID={runID} trace={run?.trace} />
                ) : null,
              },
            ]}
          />
        </div>

        <div className="relative cursor-col-resize" onMouseDown={handleMouseDown}>
          <div className="bg-canvasMuted absolute inset-0 z-[1] h-full w-px" />
          <div
            className="absolute z-[1] -translate-x-1/2"
            style={{
              top:
                (runInfoRef.current?.clientHeight ?? 0) +
                (height - (runInfoRef.current?.clientHeight ?? 0)) / 2,
            }}
          >
            <DragDivider className="bg-canvasBase" />
          </div>
        </div>

        <div
          className="border-muted flex flex-col justify-start"
          style={{ width: `${100 - leftWidth}%`, height: standalone ? '85vh' : height }}
        >
          {selectedStep && !selectedStep.trace.isRoot ? (
            <StepInfo
              selectedStep={selectedStep}
              getResult={getResult}
              pollInterval={pollInterval}
            />
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
