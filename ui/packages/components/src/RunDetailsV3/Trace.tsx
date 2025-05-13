import { useEffect, useState } from 'react';
import type { Route } from 'next';
import { RiArrowRightSLine } from '@remixicon/react';

import type { Result } from '../types/functionRun';
import { toMaybeDate } from '../utils/date';
import { InlineSpans } from './InlineSpans';
import { TimelineHeader } from './TimelineHeader';
import { type Trace } from './types';
import { FINAL_SPAN_NAME, createSpanWidths, getSpanName, useStepSelection } from './utils';

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
  const { selectStep, selectedStep } = useStepSelection(runID);

  useEffect(() => {
    if (expanded && !result && trace.outputID) {
      getResult(trace.outputID).then((data) => {
        setResult(data);
      });
    }
  }, [expanded, result]);

  //
  // Don't show single finalization step for successful runs
  // unless they have children (e.g. failed attempts)
  const hasChildren =
    depth === 0 &&
    trace.childrenSpans?.length === 1 &&
    trace.childrenSpans[0]?.name === FINAL_SPAN_NAME &&
    (trace.childrenSpans[0]?.childrenSpans?.length ?? 0) == 0
      ? false
      : (trace.childrenSpans?.length ?? 0) > 0;

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
        onClick={() => selectStep({ trace, runID, result, pathCreator })}
      >
        <div
          className="flex w-[30%] flex-row items-center justify-start gap-1 overflow-hidden"
          style={{ paddingLeft: `${depth * 40}px` }}
        >
          {hasChildren && (
            <div
              className="border-subtle flex flex-row items-center justify-center rounded border p-0 pl-1"
              onClick={(e) => {
                e.stopPropagation();
                setExpanded(!expanded);
              }}
            >
              <div className="text-sm font-medium leading-tight">{trace.childrenSpans?.length}</div>
              <RiArrowRightSLine
                className={`text-subtle m-0 h-3.5 w-3.5 shrink-0 transition-transform duration-[250ms] ${
                  expanded ? 'rotate-90' : ''
                }`}
              />
            </div>
          )}

          <div
            className={`text-basis overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight ${
              !hasChildren && 'pl-1.5'
            }`}
          >
            {getSpanName(trace.name)}
          </div>
        </div>

        <div className="border-light/80 flex w-[70%] flex-row border-l-2">
          <InlineSpans maxTime={maxTime} minTime={minTime} trace={trace} />
        </div>
      </div>

      {expanded && hasChildren && (
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
