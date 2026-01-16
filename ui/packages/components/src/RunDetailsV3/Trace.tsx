import { useRef, useState } from 'react';
import { RiArrowRightSLine } from '@remixicon/react';

import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip/Tooltip';
import { InlineSpans } from './InlineSpans';
import { StepType } from './StepType';
import { TimelineHeader } from './TimelineHeader';
import { type Trace } from './types';
import { getSpanName, traceHasChildren, useStepSelection } from './utils';

type Props = {
  depth: number;
  minTime: Date;
  maxTime: Date;
  trace: Trace;
  runID: string;
  leftWidth: number;
  onResizeStart: () => void;
};

const INDENT_WIDTH = 40;

export function Trace({ depth, maxTime, minTime, trace, runID, leftWidth, onResizeStart }: Props) {
  const [expanded, setExpanded] = useState(true);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const { selectStep, selectedStep } = useStepSelection({ runID });
  const expanderRef = useRef<HTMLDivElement>(null);
  const spanNameRef = useRef<HTMLDivElement>(null);

  const hasChildren = traceHasChildren(depth, trace);
  const spanName = getSpanName(trace.name);

  const handleTooltipOpenChange = (open: boolean) => {
    if (!open) {
      setTooltipOpen(false);
      return;
    }

    const element = spanNameRef.current;
    if (element && element.scrollWidth > element.clientWidth) {
      setTooltipOpen(true);
    }
  };

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
          onClick={() => {
            selectStep({ trace, runID });
          }}
        >
          <div
            className="flex flex-row items-center justify-start gap-1 overflow-hidden"
            style={{ width: `${leftWidth}%`, paddingLeft: `${depth * INDENT_WIDTH}px` }}
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
            <Tooltip open={tooltipOpen} onOpenChange={handleTooltipOpenChange}>
              <TooltipTrigger asChild>
                <div
                  ref={spanNameRef}
                  className={`text-basis min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight ${
                    !hasChildren && 'pl-1.5'
                  }`}
                >
                  {spanName}
                </div>
              </TooltipTrigger>
              <TooltipContent
                side="top"
                className="flex min-h-8 items-center px-4 text-xs leading-[18px]"
              >
                {spanName}
              </TooltipContent>
            </Tooltip>
          </div>

          <div
            className="border-light/80 relative flex flex-row border-l-2"
            style={{ width: `${100 - leftWidth}%` }}
          >
            <div
              className="absolute -left-1.5 top-0 z-10 h-7 w-3 cursor-col-resize"
              onMouseDown={(e) => {
                e.stopPropagation();
                onResizeStart();
              }}
            />
            <InlineSpans maxTime={maxTime} minTime={minTime} trace={trace} depth={depth} />
          </div>
        </div>
        {expanded && hasChildren && (
          <>
            {trace.childrenSpans?.map((child, i) => (
              <Trace
                key={`${child.name}-${i}`}
                depth={depth + 1}
                maxTime={maxTime}
                minTime={minTime}
                trace={child}
                runID={runID}
                leftWidth={leftWidth}
                onResizeStart={onResizeStart}
              />
            ))}
          </>
        )}
      </div>
    </div>
  );
}
