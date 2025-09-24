import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { usePathCreator } from '@inngest/components/SharedContext/usePathCreator';
import { RiGitForkLine, RiPauseLine, RiStopLine } from '@remixicon/react';
import { toast } from 'sonner';

import { Button } from '../Button';
import { RerunModal } from '../Rerun/RerunModal';
import { StepInfo } from '../RunDetailsV3/StepInfo';
import { Timeline } from '../RunDetailsV3/Timeline';
import { useStepSelection } from '../RunDetailsV3/utils';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { useGetDebugRun } from '../SharedContext/useGetDebugRun';
import { useGetRunTrace } from '../SharedContext/useGetRunTrace';
import { useRerun } from '../SharedContext/useRerun';
import { Skeleton } from '../Skeleton';
import { StatusDot } from '../Status/StatusDot';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { useSearchParam } from '../hooks/useSearchParam';
import { DragDivider } from '../icons/DragDivider';
import { StepOver } from '../icons/debug/StepOver';
import { DebugRun } from './DebugRun';
import { History } from './History';
import { Play } from './Play';

const DEBUG_RUN_REFETCH_INTERVAL = 1000;
const RUN_REFETCH_INTERVAL = 1000;

export const Debugger = ({ functionSlug }: { functionSlug: string }) => {
  const router = useRouter();
  const { pathCreator } = usePathCreator();
  const [runID] = useSearchParam('runID');
  const [rerunModalOpen, setRerunModalOpen] = useState(false);
  const [debugRunID] = useSearchParam('debugRunID');
  const [debugSessionID] = useSearchParam('debugSessionID');
  const [runDone, setRunDone] = useState(false);
  const { selectedStep } = useStepSelection({
    debugRunID,
    runID,
  });

  const { booleanFlag } = useBooleanFlag();
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false
  );

  const { rerun } = useRerun();

  const { data: runTraceData, loading: runTraceLoading } = useGetRunTrace({
    runID,
    refetchInterval: runDone || (pollingFlagReady && pollingDisabled) ? 0 : RUN_REFETCH_INTERVAL,
  });

  const { data: debugRunData, loading } = useGetDebugRun({
    functionSlug,
    debugRunID,
    runID,
    refetchInterval: pollingFlagReady && pollingDisabled ? 0 : DEBUG_RUN_REFETCH_INTERVAL,
  });

  useEffect(() => {
    if (runTraceData?.status === 'COMPLETED') {
      setRunDone(true);
    }
  }, [runTraceData?.status]);

  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [leftWidth, setLeftWidth] = useState(50);

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

  const handleRerun = async () => {
    if (runID) {
      const result = await rerun({
        runID,
        debugRunID,
        debugSessionID,
      });

      if (result.error) {
        console.error('error rerunning function', result.error);
        toast.error(`Error rerunning function, see console for more details.`);
        return;
      }

      if (result.data?.newRunID) {
        router.push(
          pathCreator.debugger({
            functionSlug,
            runID: result.data?.newRunID,
            debugSessionID: result.data?.newRunID,
          })
        );
      }
    }
  };

  return (
    <>
      <div className="mx-4 my-8 flex flex-row items-center justify-between">
        <div className="flex flex-col gap-2">
          <div className="text-xl">{functionSlug}</div>

          <div className="flex flex-row items-center gap-x-2 text-sm">
            <RiGitForkLine className="text-muted h-6 w-6" />
            <div>Forked from:</div>
            {runTraceData?.status && <StatusDot status={runTraceData.status} className="h-3 w-3" />}
            <div>{runID && <Link href={pathCreator.runPopout({ runID })}>{runID}</Link>}</div>
          </div>
        </div>

        <Tooltip>
          <TooltipTrigger>
            <Button
              kind="primary"
              appearance="outlined"
              size="medium"
              label="Rerun function"
              onClick={handleRerun}
            />
          </TooltipTrigger>
          <TooltipContent className="whitespace-pre-line">
            Reruns function and start a new debug session
          </TooltipContent>
        </Tooltip>
      </div>

      <div className="flex h-full w-full flex-row" ref={containerRef}>
        <div ref={leftColumnRef} style={{ width: `${leftWidth}%` }}>
          <div className="flex flex-col gap-2">
            <div className="border-muted  h-12 w-full border-b border-t px-4">
              <div className="flex flex-row items-center gap-x-2">
                <Tooltip>
                  <TooltipTrigger>
                    {runTraceData?.status === 'RUNNING' ? (
                      <RiPauseLine className="text-subtle hover:bg-canvasSubtle h-8 w-8 cursor-not-allowed rounded-md p-1" />
                    ) : (
                      <Play
                        functionSlug={functionSlug}
                        runID={runID}
                        debugRunID={debugRunID}
                        debugSessionID={debugSessionID}
                      />
                    )}
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">
                    {runTraceData?.status === 'RUNNING' ? 'Pause coming soon!' : 'Play'}
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger>
                    <StepOver className="text-subtle hover:bg-canvasSubtle h-8 w-8 cursor-not-allowed rounded-md p-1 opacity-50" />
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">
                    Step Over coming soon!
                  </TooltipContent>
                </Tooltip>

                <div className="bg-canvasMuted my-2 h-8 w-px" />
                <Tooltip>
                  <TooltipTrigger>
                    <RiStopLine className="text-subtle hover:bg-canvasSubtle h-8 w-8 cursor-not-allowed rounded-md p-1 opacity-50" />
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">Stop coming soon!</TooltipContent>
                </Tooltip>
              </div>
            </div>
            <div>
              {loading || runTraceLoading ? (
                <Skeleton className="h-24 w-full" />
              ) : (
                <DebugRun
                  debugTraces={debugRunData?.debugTraces}
                  runTrace={runTraceData}
                  runID={runID}
                />
              )}
            </div>
          </div>
        </div>
        <div className="relative cursor-col-resize" onMouseDown={handleMouseDown}>
          <div className="bg-canvasMuted absolute inset-0 z-[1] h-full w-px" />
          <div
            className="absolute z-[1] -translate-x-1/2"
            style={{
              top: (containerRef.current?.clientHeight ?? 0) / 4,
            }}
          >
            <DragDivider className="bg-canvasBase" />
          </div>
        </div>
        <div style={{ width: `${100 - leftWidth}%` }}>
          <div className="flex flex-col items-start justify-start">
            <div className="border-muted flex h-12 w-full flex-row items-center justify-end border-b border-t px-4">
              <Button
                kind="secondary"
                appearance="outlined"
                size="medium"
                label="Edit and rerun from step"
                disabled={!selectedStep?.trace.stepID}
                onClick={() => setRerunModalOpen(true)}
              />
              {runID && selectedStep?.trace.stepID && (
                <RerunModal
                  open={rerunModalOpen}
                  setOpen={setRerunModalOpen}
                  runID={runID}
                  stepID={selectedStep?.trace.stepID}
                  debugRunID={debugRunID}
                  debugSessionID={debugSessionID}
                  redirect={false}
                  //
                  // TODO: fetch step result
                  input={''}
                />
              )}
            </div>

            <History functionSlug={functionSlug} debugSessionID={debugSessionID} runID={runID} />
            {selectedStep && (
              <StepInfo
                selectedStep={selectedStep}
                pollInterval={1000}
                tracesPreviewEnabled={true}
              />
            )}
          </div>
        </div>
      </div>
    </>
  );
};
