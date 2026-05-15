import { useRef, useState } from 'react';
import { RiArrowRightSLine } from '@remixicon/react';

import { InlineSpans } from '../RunDetailsV3/InlineSpans';
import { StepType } from '../RunDetailsV3/StepType';
import { TimelineHeader } from '../RunDetailsV3/TimelineHeader';
import { type Trace } from '../RunDetailsV3/types';
import { getSpanName, traceHasChildren, useStepSelection } from '../RunDetailsV3/utils';
import { overlayDebugRuns } from './utils';

type Props = {
  depth: number;
  minTime: Date;
  maxTime: Date;
  runTrace: Trace;
  runID: string;
  debugTraces?: Trace[];
};

const INDENT_WIDTH = 40;

export function DebugTrace({
  depth,
  maxTime,
  minTime,
  runTrace: originalTrace,
  runID,
  debugTraces,
}: Props) {
  const [expanded, setExpanded] = useState(true);
  const { selectStep, selectedStep } = useStepSelection({ runID });
  const expanderRef = useRef<HTMLDivElement>(null);

  const runTrace = debugTraces ? overlayDebugRuns(originalTrace, debugTraces) : originalTrace;

  const hasChildren = traceHasChildren(depth, runTrace);

  const spanName = runTrace.name === 'Run' ? 'Debug Run' : getSpanName(runTrace.name);

  return (
    <div className="relative flex w-full flex-col">
      <TimelineHeader trace={runTrace} minTime={minTime} maxTime={maxTime} />
      <div className="flex flex-col">
        <div
          className={`flex h-7 w-full cursor-pointer flex-row items-center justify-start gap-1 bg-opacity-50 py-0.5 pl-4 ${
            (!selectedStep && runTrace.isRoot) ||
            (selectedStep?.trace?.spanID === runTrace.spanID &&
              selectedStep?.trace?.name === runTrace.name)
              ? 'bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle hover:bg-opacity-60'
          } `}
          onClick={() => {
            selectStep({ trace: runTrace, runID });
          }}
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
                  top: runTrace.isRoot ? '2rem' : '1rem',
                  height: `calc(100% - ${runTrace.isRoot ? '2rem' : '1rem'})`,
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
                  {runTrace.childrenSpans?.length}
                </div>
                <RiArrowRightSLine
                  className={`text-subtle m-0 h-3.5 w-3.5 shrink-0 transition-transform duration-[250ms] ${
                    expanded ? 'rotate-90' : ''
                  }`}
                />
              </div>
            )}
            <StepType stepType={runTrace.stepType} />
            <div
              className={`text-basis overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight ${
                !hasChildren && 'pl-1.5'
              }`}
            >
              {spanName}
            </div>
          </div>

          <div className="border-light/80 flex w-[70%] flex-row border-l-2">
            <InlineSpans maxTime={maxTime} minTime={minTime} trace={runTrace} depth={depth} />
          </div>
        </div>
        {expanded && hasChildren && (
          <>
            {runTrace.childrenSpans?.map((child, i) => {
              return (
                <DebugTrace
                  key={`${child.name}-${i}`}
                  depth={depth + 1}
                  maxTime={maxTime}
                  minTime={minTime}
                  runTrace={child}
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
