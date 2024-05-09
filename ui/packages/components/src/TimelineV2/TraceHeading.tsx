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
  let opCodeBadge;
  if (trace.stepOp && (trace.childrenSpans?.length ?? 0) > 0) {
    const isRetried = (trace.attempts ?? 0) > 1;

    opCodeBadge = (
      <span className="ml-2 flex h-fit">
        <Tooltip>
          <TooltipTrigger>
            <Badge
              className="bg-slate-200 px-1.5 font-mono"
              flatSide={isRetried ? 'right' : undefined}
              kind="solid"
            >
              <span>{trace.stepOp.toLowerCase()}</span>
            </Badge>
          </TooltipTrigger>
          <TooltipContent>Step method</TooltipContent>
        </Tooltip>

        {(trace.attempts ?? 0) > 1 && (
          <Tooltip>
            <TooltipTrigger>
              <Badge
                className="bg-amber-100 px-1.5 font-mono text-amber-700"
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
        <div className="flex gap-2">
          <span className="grow text-sm">{trace.name}</span>
          {opCodeBadge}
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
