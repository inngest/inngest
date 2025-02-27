import { useEffect, useState } from 'react';
import type { Route } from 'next';
import { RiArrowRightSLine } from '@remixicon/react';

import type { Result } from '../types/functionRun';
import { toMaybeDate } from '../utils/date';
import { InlineSpans } from './InlineSpans';
import { TimelineHeader } from './TimelineHeader';
import { type Trace } from './types';
import { createSpanWidths, useStepSelection } from './utils';

type Props = {
  depth: number;
  getResult: (outputID: string) => Promise<Result>;
  minTime: Date;
  maxTime: Date;
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  trace: Trace;
  runID: string;
};

export function Trace({ depth, getResult, maxTime, minTime, pathCreator, trace, runID }: Props) {
  const [expanded, setExpanded] = useState(true);
  const [result, setResult] = useState<Result>();
  const { selectStep, selectedStep } = useStepSelection();

  useEffect(() => {
    if (expanded && !result && trace.outputID) {
      getResult(trace.outputID).then((data) => {
        setResult(data);
      });
    }
  }, [expanded, result]);

  const widths = createSpanWidths({
    ended: toMaybeDate(trace.endedAt)?.getTime() ?? null,
    max: maxTime.getTime(),
    min: minTime.getTime(),
    queued: new Date(trace.queuedAt).getTime(),
    started: toMaybeDate(trace.startedAt)?.getTime() ?? null,
  });

  const spans =
    !trace.isRoot && trace.childrenSpans && trace.childrenSpans.length > 0
      ? trace.childrenSpans
      : [trace];

  const hasChildren = (trace.childrenSpans?.length ?? 0) > 0;

  return (
    <div className="relative flex w-full flex-col">
      <TimelineHeader trace={trace} minTime={minTime} maxTime={maxTime} />

      <div
        className={`flex h-7 w-full cursor-pointer flex-row items-center justify-start gap-1 bg-opacity-50 py-0.5 pl-4 ${
          (!selectedStep && trace.isRoot) ||
          (selectedStep?.trace?.spanID === trace.spanID && selectedStep?.trace?.name === trace.name)
            ? 'bg-secondary-3xSubtle'
            : 'hover:bg-canvasSubtle'
        } `}
        onClick={() => selectStep(depth ? { trace, runID, result, pathCreator } : undefined)}
      >
        <div
          className="flex w-[30%] flex-row items-center justify-start gap-1 overflow-hidden"
          style={{ paddingLeft: `${depth * 40}px` }}
        >
          {hasChildren && (
            <div
              className="flex shrink-0 flex-row items-center justify-start gap-1"
              onClick={(e) => {
                e.stopPropagation();
                setExpanded(!expanded);
              }}
            >
              <div className="text-sm font-medium leading-tight">{trace.childrenSpans?.length}</div>
              <RiArrowRightSLine
                className={`shrink-0 transition-transform duration-500 ${
                  expanded ? 'rotate-90' : ''
                }`}
              />
            </div>
          )}

          <div className="text-basis overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight">
            {trace.name}
          </div>
        </div>

        <div className="border-light/80 flex w-[70%] flex-row border-l-2">
          <InlineSpans
            maxTime={maxTime}
            minTime={minTime}
            name={trace.name}
            spans={spans}
            widths={widths}
          />
        </div>
      </div>

      {expanded && (
        <>
          {trace.childrenSpans?.map((child, i) => {
            return (
              <Trace
                key={`${child.name}-${i}`}
                depth={depth + 1}
                getResult={getResult}
                maxTime={maxTime}
                minTime={minTime}
                pathCreator={pathCreator}
                trace={child}
                runID={runID}
              />
            );
          })}
        </>
      )}
    </div>
  );
}
