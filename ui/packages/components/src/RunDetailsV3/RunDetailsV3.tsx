'use client';

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { ErrorCard } from '../Error/ErrorCard';
import type { Run as InitialRunData } from '../RunsPage/types';
import { useGetRun } from '../SharedContext/useGetRun';
import { useGetTraceResult } from '../SharedContext/useGetTraceResult';
import { StatusCell } from '../Table/Cell';
import { TriggerDetails } from '../TriggerDetails';
import { DragDivider } from '../icons/DragDivider';
import { nullishToLazy } from '../utils/lazyLoad';
import { RunInfo } from './RunInfo';
import { StepInfo } from './StepInfo';
import { Tabs } from './Tabs';
import { Timeline } from './Timeline';
import { TopInfo } from './TopInfo';
import { Waiting } from './Waiting';
import { useStepSelection } from './utils';

type Props = {
  standalone: boolean;
  initialRunData?: InitialRunData;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pollInterval?: number;
  runID: string;
  tracesPreviewEnabled: boolean;
};

const MIN_HEIGHT = 586;
const NO_SPANS_OR_TRACE_ERROR = /no function run span found|trace run not found/gi;

//
// Do not show the error if queued and can't find the trace or spans (backend timing issue)
const isWaiting = (status?: string, runError?: Error | null, traceResultError?: Error | null) => {
  if (status && status !== 'QUEUED') {
    return false;
  }

  return (
    !!runError?.toString().match(NO_SPANS_OR_TRACE_ERROR) ||
    !!traceResultError?.toString().match(NO_SPANS_OR_TRACE_ERROR)
  );
};

export const RunDetailsV3 = ({
  getTrigger,
  runID,
  standalone,
  tracesPreviewEnabled,
  pollInterval: initialPollInterval,
  initialRunData,
}: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const runInfoRef = useRef<HTMLDivElement>(null);
  const [pollInterval, setPollInterval] = useState(
    initialPollInterval ? initialPollInterval : standalone ? 1000 : undefined
  );

  const [leftWidth, setLeftWidth] = useState(55);
  const [height, setHeight] = useState(MIN_HEIGHT);
  const [isDragging, setIsDragging] = useState(false);
  const [windowHeight, setWindowHeight] = useState(0);
  const { selectedStep } = useStepSelection(runID);
  const { getTraceResult } = useGetTraceResult();

  const { getRun, error: runError } = useGetRun();

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
    setWindowHeight(window.innerHeight);

    const handleResize = () => {
      setWindowHeight(window.innerHeight);
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useLayoutEffect(() => {
    if (!leftColumnRef.current) {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      const h = leftColumnRef.current?.clientHeight ?? 0;
      setHeight(h > MIN_HEIGHT ? h : MIN_HEIGHT);
    });

    resizeObserver.observe(leftColumnRef.current);

    return () => {
      resizeObserver.disconnect();
    };
  }, []);

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
    queryKey: ['run', runID, { preview: tracesPreviewEnabled }],
    queryFn: useCallback(() => {
      return getRun({ runID, preview: tracesPreviewEnabled });
    }, [getRun, runID, tracesPreviewEnabled]),
    retry: 3,
    refetchInterval: pollInterval,
  });

  const outputID = runRes?.data?.data?.trace?.outputID;
  const resultRes = useQuery({
    enabled: Boolean(outputID),
    refetchInterval: pollInterval,
    queryKey: ['run-result', runID, { preview: tracesPreviewEnabled }],
    queryFn: useCallback(() => {
      if (!outputID) {
        // Unreachable
        throw new Error('missing outputID');
      }
      return getTraceResult({ traceID: outputID, preview: tracesPreviewEnabled });
    }, [getTraceResult, outputID, tracesPreviewEnabled]),
  });

  const run = runRes.data?.data;
  if (run?.trace.endedAt && pollInterval) {
    //
    // Stop polling for ended runs, but still give it
    // a few seconds for any lingering userland traces.
    setTimeout(() => {
      setPollInterval(undefined);
    }, 6000);
  }

  const waiting = isWaiting(
    initialRunData?.status || runRes?.data?.data?.trace?.status,
    runError || runRes.error,
    resultRes.error
  );
  const showError = waiting ? false : runRes.error || resultRes.error;

  //
  // works around a variety of layout and scroll issues with our two column layout
  const dynamicHeight = standalone ? '85vh' : height < windowHeight * 0.85 ? height : '85vh';

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
              initialRunData={initialRunData}
              run={nullishToLazy(run)}
              runID={runID}
              standalone={standalone}
              result={resultRes?.data}
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
          className="border-muted sticky top-0 flex flex-col justify-start overflow-y-auto"
          style={{
            width: `${100 - leftWidth}%`,
            height: dynamicHeight,
            alignSelf: 'flex-start',
          }}
        >
          {selectedStep && !selectedStep.trace.isRoot ? (
            <StepInfo
              selectedStep={selectedStep}
              pollInterval={pollInterval}
              tracesPreviewEnabled={tracesPreviewEnabled}
            />
          ) : (
            <TopInfo
              slug={run?.fn.slug}
              getTrigger={getTrigger}
              runID={runID}
              result={resultRes?.data}
            />
          )}
        </div>
      </div>
    </>
  );
};
