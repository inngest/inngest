import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathCreator } from '@inngest/components/SharedContext/usePathCreator';
import { RiGitForkLine, RiPauseLine, RiStopLine } from '@remixicon/react';

import { Button } from '../Button';
import { Timeline } from '../RunDetailsV3/Timeline';
import { useGetRun } from '../SharedContext/useGetRun';
import { Skeleton } from '../Skeleton';
import { StatusDot } from '../Status/StatusDot';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { useSearchParam } from '../hooks/useSearchParam';
import { DragDivider } from '../icons/DragDivider';
import { StepOver } from '../icons/debug/StepOver';
import { History } from './History';
import { Play } from './Play';

export const Debugger = ({ functionSlug }: { functionSlug: string }) => {
  const { pathCreator } = usePathCreator();
  const [runID] = useSearchParam('runID');
  const { data: runData, loading: runLoading } = useGetRun({
    runID,
  });

  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [leftWidth, setLeftWidth] = useState(50);
  const [running, setRunning] = useState(false);

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

  return (
    <>
      <div className="mx-4 my-8 flex flex-row items-center justify-between">
        <div className="flex flex-col gap-2">
          <div className="text-xl">{functionSlug}</div>

          <div className="flex flex-row items-center gap-x-2 text-sm">
            <RiGitForkLine className="text-muted h-6 w-6" />
            <div>Forked from:</div>
            {runData?.trace?.status && (
              <StatusDot status={runData?.trace?.status} className="h-3 w-3" />
            )}
            <div>{runID && <Link href={pathCreator.runPopout({ runID })}>{runID}</Link>}</div>
          </div>
        </div>

        <Button kind="primary" appearance="outlined" size="medium" label="Rerun function" />
      </div>

      <div className="flex h-full w-full flex-row" ref={containerRef}>
        <div ref={leftColumnRef} style={{ width: `${leftWidth}%` }}>
          <div className="flex flex-col gap-2">
            <div className="border-muted  h-12 w-full border-b border-t px-4">
              <div className="flex flex-row items-center gap-x-2">
                <Tooltip>
                  <TooltipTrigger>
                    {running ? (
                      <RiPauseLine className="text-muted hover:bg-canvasSubtle h-6 w-6 cursor-pointer rounded-md p-1" />
                    ) : (
                      <Play runID={runID} />
                    )}
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">
                    {running ? 'Pause' : 'Play'}
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger>
                    <StepOver className="text-muted hover:bg-canvasSubtle h-6 w-6 cursor-pointer rounded-md p-1" />
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">Step Over</TooltipContent>
                </Tooltip>

                <div className="bg-canvasMuted my-2 h-8 w-px" />
                <Tooltip>
                  <TooltipTrigger>
                    <RiStopLine className="text-muted hover:bg-canvasSubtle h-6 w-6 cursor-pointer rounded-md p-1" />
                  </TooltipTrigger>
                  <TooltipContent className="whitespace-pre-line">Stop</TooltipContent>
                </Tooltip>
              </div>
            </div>
            <div>
              {runLoading ? (
                <Skeleton className="h-24 w-full" />
              ) : runID && runData ? (
                <Timeline runID={runID} trace={runData?.trace} />
              ) : null}
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
          <div className="flex flex-col items-start justify-start gap-2">
            <div className="border-muted flex h-12 w-full flex-row items-center justify-end border-b border-t px-4">
              <Button
                kind="secondary"
                appearance="outlined"
                size="medium"
                label="Edit and rerun from step"
                disabled={true}
              />
            </div>

            <History />
          </div>
        </div>
      </div>
    </>
  );
};
