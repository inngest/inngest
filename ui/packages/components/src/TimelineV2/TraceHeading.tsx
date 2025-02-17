import { RiArrowRightSLine, RiSparkling2Fill } from '@remixicon/react';

import { Button } from '../Button';
import { Pill } from '../Pill';
import { Time } from '../Time';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { isFunctionRunStatus } from '../types/functionRun';
import { cn } from '../utils/classNames';

type Props = {
  isExpanded: boolean;
  isExpandable: boolean;
  onClickExpandToggle: () => void;
  trace: {
    attempts: number | null;
    childrenSpans?: unknown[];
    endedAt: string | null;
    name: string;
    queuedAt: string;
    startedAt: string | null;
    status: string;
    stepOp?: string | null;
  };
  isAI?: boolean;
};

export function TraceHeading({
  isExpanded,
  isExpandable,
  onClickExpandToggle,
  trace,
  isAI,
}: Props) {
  const isAttempt = trace.stepOp === 'RUN' && (trace.childrenSpans?.length ?? 0) === 0;
  let opCodeBadge;
  if (trace.stepOp && !isAttempt) {
    const title = trace.stepOp.split('_')[0]?.toLowerCase();
    const isRetried = (trace.attempts ?? 0) > 1;

    opCodeBadge = (
      <span className="flex h-fit">
        <Tooltip>
          <TooltipTrigger>
            <Pill
              className="border-muted text-subtle border px-1.5"
              flatSide={isRetried ? 'right' : undefined}
            >
              <span>{title}</span>
            </Pill>
          </TooltipTrigger>
          <TooltipContent>Step method</TooltipContent>
        </Tooltip>

        {(trace.attempts ?? 0) > 1 && (
          <Tooltip>
            <TooltipTrigger>
              <Pill
                className="border-r-1 border-muted text-subtle border-l-0 px-1.5"
                flatSide="left"
              >
                <span>{trace.attempts}</span>
              </Pill>
            </TooltipTrigger>
            <TooltipContent>Attempt count</TooltipContent>
          </Tooltip>
        )}
      </span>
    );
  }

  return (
    <div className="text-basis flex w-72 gap-2">
      {isExpandable && (
        <Button
          onClick={onClickExpandToggle}
          className={cn('flex-none', isExpanded && 'border border-transparent')}
          kind={isExpanded ? 'primary' : 'secondary'}
          appearance={isExpanded ? 'solid' : 'outlined'}
          icon={
            <RiArrowRightSLine
              className={cn(isExpanded && 'rotate-90 ', 'h-4 transition-transform duration-500')}
            />
          }
        />
      )}

      <div className="grow">
        <div className="flex items-center justify-start gap-2">
          {isAI && <RiSparkling2Fill className="text-primary-xIntense h-4 w-4" />}
          <span className={`mt-1 h-fit self-start text-sm ${isAI && 'text-primary-xIntense'}`}>
            {trace.name}
          </span>

          {!isAI && <div className="h-8">{opCodeBadge}</div>}
        </div>
        <TimeWithText trace={trace} />
      </div>
    </div>
  );
}

function TimeWithText({ trace }: { trace: Props['trace'] }) {
  let text: string;
  let value: Date;
  if (trace.endedAt) {
    text = 'Ended';

    if (isFunctionRunStatus(trace.status)) {
      if (trace.status === 'CANCELLED') {
        text = 'Cancelled';
      } else if (trace.status === 'COMPLETED') {
        text = 'Completed';
      } else if (trace.status === 'FAILED') {
        text = 'Failed';
      }
    }

    value = new Date(trace.endedAt);
  } else if (trace.startedAt) {
    text = 'Started';
    value = new Date(trace.startedAt);
  } else {
    text = 'Queued';
    value = new Date(trace.queuedAt);
  }

  return (
    <div className="text-basis text-xs">
      {text}: <Time value={value} />
    </div>
  );
}
