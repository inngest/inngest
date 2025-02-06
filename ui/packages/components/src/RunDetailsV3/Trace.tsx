import { useEffect, useState } from 'react';
import type { Route } from 'next';
import { RiArrowUpSLine } from '@remixicon/react';

import type { Result } from '../types/functionRun';
import { toMaybeDate } from '../utils/date';
import { InlineSpans } from './InlineSpans';
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
  leftWidth: number;
  handleMouseDown: (e: React.MouseEvent) => void;
};

export function Trace({
  depth,
  getResult,
  maxTime,
  minTime,
  pathCreator,
  trace,
  runID,
  leftWidth,
  handleMouseDown,
}: Props) {
  const [expanded, setExpanded] = useState(true);
  const [result, setResult] = useState<Result>();
  const { selectStep } = useStepSelection();

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

  return (
    <>
      <div
        className="flex h-7 w-full cursor-pointer flex-row items-center justify-start gap-1"
        onClick={() => selectStep(depth ? { trace, runID, result, pathCreator } : undefined)}
      >
        <div
          className="flex flex-row items-center justify-start gap-1"
          style={{ width: `${leftWidth}%`, paddingLeft: `${depth * 40}px` }}
        >
          {(trace.childrenSpans?.length ?? 0) > 0 && (
            <>
              <div className="text-sm font-medium leading-tight">{trace.childrenSpans?.length}</div>
              <RiArrowUpSLine
                className={`w-3 shrink-0 cursor-pointer transition-transform duration-500 ${
                  expanded ? 'rotate-180' : ''
                }`}
                onClick={() => setExpanded(!expanded)}
              />
            </>
          )}

          <div className="text-basis text-sm font-normal leading-tight">{trace.name}</div>
        </div>

        <div
          className="border-muted h-7 w-2 cursor-col-resize border-r-[.5px]"
          onMouseDown={handleMouseDown}
        />

        <div style={{ width: `${100 - leftWidth}%` }}>
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
                leftWidth={leftWidth}
                handleMouseDown={handleMouseDown}
              />
            );
          })}
        </>
      )}
    </>
  );
}
