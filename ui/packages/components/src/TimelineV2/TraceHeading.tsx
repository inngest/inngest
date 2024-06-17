import { RiArrowRightSLine } from '@remixicon/react';

import { Badge } from '../Badge';
import { Button } from '../Button';
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
};

export function TraceHeading({ isExpanded, isExpandable, onClickExpandToggle, trace }: Props) {
  const isAttempt = trace.stepOp === 'RUN' && (trace.childrenSpans?.length ?? 0) === 0;
  let opCodeBadge;
  if (trace.stepOp && !isAttempt) {
    const title = trace.stepOp.split('_')[0]?.toLowerCase();
    const isRetried = (trace.attempts ?? 0) > 1;

    opCodeBadge = (
      <span className="ml-2 flex h-fit">
        <Tooltip>
          <TooltipTrigger>
            <Badge
              className="border border-slate-400 bg-white px-1.5"
              flatSide={isRetried ? 'right' : undefined}
              kind="solid"
            >
              <span>{title}</span>
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Step method</TooltipContent>
        </Tooltip>

        {(trace.attempts ?? 0) > 1 && (
          <Tooltip>
            <TooltipTrigger>
              <Badge
                className="border-r-1 border-l-0 border-slate-400 bg-white px-1.5"
                flatSide="left"
                kind="solid"
              >
                <span>{trace.attempts}</span>
              </Badge>
            </TooltipTrigger>
            <TooltipContent>Attempt count</TooltipContent>
          </Tooltip>
        )}
      </span>
    );
  }

  return (
    <div className="flex w-72 gap-2">
      {isExpandable && (
        <Button
          btnAction={onClickExpandToggle}
          className="flex-none"
          size="small"
          appearance={isExpanded ? 'solid' : 'outlined'}
          icon={
            <RiArrowRightSLine
              className={cn(isExpanded && 'rotate-90 ', 'h-4 transition-transform duration-500')}
            />
          }
        />
      )}

      <div className="grow">
        <div className="flex">
          <span className="mt-1 h-fit self-start text-sm">{trace.name}</span>
          <div className="flex h-8 grow items-center">
            {opCodeBadge}
            <div className="ml-2 h-px grow bg-slate-100" />
          </div>
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
    <div className="text-xs text-slate-600">
      {text}: <Time value={value} />
    </div>
  );
}
