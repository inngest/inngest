import { useRef, useState } from 'react';
import { RiArrowRightSLine } from '@remixicon/react';

import { InlineSpans } from './InlineSpans';
import { StepType } from './StepType';
import { TimelineHeader } from './TimelineHeader';
import { type Trace } from './types';
import { FINAL_SPAN_DISPLAY, FINAL_SPAN_NAME, getSpanName, useStepSelection } from './utils';

type Props = {
  depth: number;
  minTime: Date;
  maxTime: Date;
  trace: Trace;
  runID: string;
};

const INDENT_WIDTH = 40;

export function Trace({ depth, maxTime, minTime, trace, runID }: Props) {
  const [expanded, setExpanded] = useState(true);
  const { selectStep, selectedStep } = useStepSelection(runID);
  const expanderRef = useRef<HTMLDivElement>(null);

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
  const spanName = getSpanName(trace.name);
  return (
    <div className="relative flex w-full flex-col">
      <TimelineHeader trace={trace} minTime={minTime} maxTime={maxTime} />
      <div className="flex flex-col">
        <div
          className={`flex h-7 w-full cursor-pointer flex-row items-center justify-start gap-1 bg-opacity-50 py-0.5 pl-4 ${
            (!selectedStep && trace.isRoot) ||
            (selectedStep?.trace?.spanID === trace.spanID &&
              selectedStep?.trace?.name === trace.name)
              ? 'bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle hover:bg-opacity-60'
          } `}
          onClick={() => selectStep({ trace, runID })}
        >
          <div
            className="flex w-[30%] flex-row items-center justify-start gap-1 overflow-hidden"
            style={{ paddingLeft: `${depth * INDENT_WIDTH}px` }}
          >
            {hasChildren && expanded && (
              <div
                className={'border-subtle absolute z-10 w-px border-r'}
                style={{
                  //
                  // Use placeholder width of 28 (single digit expander) to reduce flickering
                  left: `${depth * INDENT_WIDTH + (expanderRef.current?.clientWidth ?? 28) + 17}px`,
                  top: trace.isRoot ? '2rem' : '1rem',
                  height: `calc(100% - ${trace.isRoot ? '2rem' : '1rem'})`,
                }}
              />
            )}

            {hasChildren && (
              <div
                className="border-subtle flex flex-row items-center justify-center rounded border p-0 pl-1"
                onClick={(e) => {
                  e.stopPropagation();
                  setExpanded(!expanded);
                }}
                ref={expanderRef}
              >
                <div className="text-sm font-medium leading-tight">
                  {trace.childrenSpans?.length}
                </div>
                <RiArrowRightSLine
                  className={`text-subtle m-0 h-3.5 w-3.5 shrink-0 transition-transform duration-[250ms] ${
                    expanded ? 'rotate-90' : ''
                  }`}
                />
              </div>
            )}
            <StepType stepType={trace.stepType} />
            <div
              className={`text-basis overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight ${
                !hasChildren && 'pl-1.5'
              }`}
            >
              {spanName}
            </div>
          </div>

          <div className="border-light/80 flex w-[70%] flex-row border-l-2">
            <InlineSpans maxTime={maxTime} minTime={minTime} trace={trace} depth={depth} />
          </div>
        </div>
        {expanded && hasChildren && (
          <>
            {trace.childrenSpans?.map((child, i) => {
              return (
                <Trace
                  key={`${child.name}-${i}`}
                  depth={depth + 1}
                  maxTime={maxTime}
                  minTime={minTime}
                  trace={child}
                  runID={runID}
                />
              );
            })}
          </>
        )}
      </div>
    </div>
  );
}
