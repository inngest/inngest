/**
 * RunDetailsV4 - Run details page using the composable TimelineBar.
 * Feature: 001-composable-timeline-bar
 *
 * This component mirrors RunDetailsV3's interface but uses the V4 Timeline
 * component for the timeline visualization.
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import { ErrorCard } from '../Error/ErrorCard';
import type { Run as InitialRunData } from '../RunsPage/types';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { useGetRun } from '../SharedContext/useGetRun';
import { useGetTraceResult } from '../SharedContext/useGetTraceResult';
import { StatusCell } from '../Table/Cell';
import { TriggerDetails } from '../TriggerDetails';
import { DragDivider } from '../icons/DragDivider';
import { isLazyDone, nullishToLazy } from '../utils/lazyLoad';
// Import V4 components (decoupled from V3)
import { RunInfo } from './RunInfo';
import { StepInfo } from './StepInfo';
import { Tabs } from './Tabs';
// Import V4 Timeline
import { Timeline } from './Timeline';
import { TopInfo } from './TopInfo';
import { Waiting } from './Waiting';
import { traceWalk, useDynamicRunData, useStepSelection } from './runDetailsUtils';
import type { Trace } from './types';
import { traceToTimelineData } from './utils/traceConversion';

// Residual poll interval for userland traces
const RESIDUAL_POLL_INTERVAL = 6000;

type Props = {
  standalone: boolean;
  initialRunData?: InitialRunData;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pollInterval?: number;
  runID: string;
  orgName?: string;
};

const MIN_HEIGHT = 586;
const NO_SPANS_OR_TRACE_ERROR = /no function run span found|trace run not found/gi;

/**
 * Check if we're in a waiting state (queued but no trace data yet)
 */
const isWaiting = (status?: string, runError?: Error | null, traceResultError?: Error | null) => {
  if (status && status !== 'QUEUED') {
    return false;
  }

  return (
    !!runError?.toString().match(NO_SPANS_OR_TRACE_ERROR) ||
    !!traceResultError?.toString().match(NO_SPANS_OR_TRACE_ERROR)
  );
};

/**
 * V4 Timeline wrapper that converts V3 Trace data to V4 format
 * and handles step selection to update the right panel.
 */
function TimelineV4Wrapper({
  runID,
  trace,
  orgName,
}: {
  runID: string;
  trace: Trace;
  orgName?: string;
}) {
  const { selectStep } = useStepSelection({ runID });

  // Build a map of spanID -> Trace for looking up traces when clicked
  const traceMap = useMemo(() => {
    const map = new Map<string, Trace>();
    traceWalk(trace, (t) => {
      map.set(t.spanID, t);
    });
    return map;
  }, [trace]);

  // Convert V3 trace to V4 TimelineData
  const timelineData = useMemo(
    () => traceToTimelineData(trace, { runID, orgName }),
    [trace, runID, orgName]
  );

  // Handle step selection - look up the trace and emit to global selection
  const handleSelectStep = useCallback(
    (stepId: string) => {
      const selectedTrace = traceMap.get(stepId);
      if (selectedTrace) {
        selectStep({ trace: selectedTrace, runID });
      }
    },
    [traceMap, selectStep, runID]
  );

  return <Timeline data={timelineData} onSelectStep={handleSelectStep} />;
}

export const RunDetailsV4 = ({
  getTrigger,
  runID,
  standalone,
  pollInterval: initialPollInterval,
  initialRunData,
  orgName,
}: Props) => {
  const { booleanFlag } = useBooleanFlag();
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false
  );
  const { value: tracesPreviewEnabled } = booleanFlag('traces-preview', true, true);
  const { updateDynamicRunData } = useDynamicRunData({ runID });

  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const runInfoRef = useRef<HTMLDivElement>(null);
  const [pollInterval, setPollInterval] = useState(initialPollInterval);

  const [leftWidth, setLeftWidth] = useState(55);
  const [height, setHeight] = useState(MIN_HEIGHT);
  const [isDragging, setIsDragging] = useState(false);
  const [windowHeight, setWindowHeight] = useState(0);
  const { selectedStep } = useStepSelection({ runID });

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

  const {
    data: runData,
    error: runError,
    refetch: refetchRun,
  } = useGetRun({
    runID,
    preview: tracesPreviewEnabled,
    refetchInterval: pollInterval,
  });

  const outputID = runData?.trace?.outputID;
  const {
    data: resultData,
    error: resultError,
    refetch: refetchResult,
  } = useGetTraceResult({
    traceID: outputID,
    preview: tracesPreviewEnabled,
    refetchInterval: pollInterval,
    enabled: Boolean(outputID),
  });

  useEffect(() => {
    if (pollingFlagReady && pollingDisabled) {
      setPollInterval(undefined);
    }
  }, [pollingFlagReady, pollingDisabled]);

  // Stop polling for ended runs, but still give it a few seconds
  // for any lingering userland traces to arrive.
  useEffect(() => {
    if (!runData?.trace.endedAt || !pollInterval) {
      return;
    }

    const timeoutId = setTimeout(() => {
      setPollInterval(undefined);
    }, RESIDUAL_POLL_INTERVAL);

    return () => clearTimeout(timeoutId);
  }, [runData?.trace.endedAt, pollInterval]);

  useEffect(() => {
    if (!runData?.status || runData?.status === initialRunData?.status) {
      return;
    }

    updateDynamicRunData({
      runID,
      status: runData.status,
      endedAt: runData.trace.endedAt ?? undefined,
    });
  }, [
    runData?.trace.endedAt,
    runData?.status,
    initialRunData?.status,
    updateDynamicRunData,
    runID,
  ]);

  const waiting = isWaiting(initialRunData?.status || runData?.status, runError, resultError);
  const showError = waiting ? false : runError || resultError;

  // Works around a variety of layout and scroll issues with two column layout
  const dynamicHeight = standalone ? '85vh' : height < windowHeight * 0.85 ? height : '85vh';

  // Check if trace data is ready for V4 Timeline
  const traceReady = runData?.trace && isLazyDone(runData.trace);

  return (
    <>
      {standalone && runData && (
        <div className="border-muted flex flex-row items-start justify-between border-b px-4 pb-4">
          <div className="flex flex-col gap-1">
            <StatusCell status={runData.status} />
            <p className="text-basis text-2xl font-medium">{runData.fn.name}</p>
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
              run={nullishToLazy(runData)}
              runID={runID}
              standalone={standalone}
              result={resultData}
            />
            {showError && (
              <ErrorCard
                error={runError || resultError}
                reset={runError ? () => refetchRun() : () => refetchResult()}
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
                ) : traceReady ? (
                  <TimelineV4Wrapper runID={runID} trace={runData.trace} orgName={orgName} />
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
              slug={runData?.fn.slug}
              getTrigger={getTrigger}
              runID={runID}
              result={resultData}
              trace={runData?.trace}
            />
          )}
        </div>
      </div>
    </>
  );
};
