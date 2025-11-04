import { useCallback, useRef, useState } from 'react';

import { ElementWrapper } from '../DetailsCard/Element';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { GroupSpan } from './GroupSpan';
import { Span } from './Span';
import { type Trace } from './types';
import { createSpanWidths, formatDuration, getSpanName } from './utils';

type Props = {
  className?: string;
  maxTime: Date;
  minTime: Date;
  trace: Trace;
  depth: number;
};

export function InlineSpans({ className, minTime, maxTime, trace, depth }: Props) {
  const [open, setOpen] = useState(false);
  const spanRef = useRef<HTMLDivElement | null>(null);
  const hoverTimeoutRef = useRef<NodeJS.Timeout>();
  const spanName = getSpanName(trace.name);

  const handleMouseEnter = useCallback(() => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }
    setOpen(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    hoverTimeoutRef.current = setTimeout(() => {
      setOpen(false);
    }, 100);
  }, []);

  const widths = createSpanWidths({
    ended: toMaybeDate(trace.endedAt)?.getTime() ?? null,
    max: maxTime.getTime(),
    min: minTime.getTime(),
    queued: new Date(trace.queuedAt).getTime(),
    started: toMaybeDate(trace.startedAt)?.getTime() ?? null,
  });

  // For steps with userland children, render the step itself to show proper background color
  const children = trace.childrenSpans || [];
  const hasUserlandChildren = depth === 1 && children.some((s) => s.isUserland);
  const spans = !trace.isRoot && children.length && !hasUserlandChildren ? children : [];
  const shouldRenderParentOverlay = spans.length > 0 && !trace.isUserland;

  return (
    <Tooltip open={open}>
      <div
        className={cn('flex h-7 grow items-center', className)}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        <div className="bg-canvasMuted h-0" style={{ flexGrow: widths.before }}></div>

        <div
          className="flex"
          style={{
            flexGrow: widths.queued + widths.running,
          }}
          ref={spanRef}
        >
          <TooltipTrigger className="flex w-full flex-row">
            {shouldRenderParentOverlay && (
              <GroupSpan
                depth={depth}
                status={trace.status}
                width={spanRef.current?.clientWidth ?? 0}
              />
            )}
            {spans.length ? (
              spans.map((span) => (
                <Span isInline key={span.spanID} maxTime={maxTime} minTime={minTime} span={span} />
              ))
            ) : (
              <Span isInline key={trace.spanID} maxTime={maxTime} minTime={minTime} span={trace} />
            )}
          </TooltipTrigger>
        </div>

        <div className="bg-canvasMuted h-0" style={{ flexGrow: widths.after }} />
      </div>
      <TooltipContent>
        <div>
          <Times isDelayVisible={spans.length === 0} name={spanName} span={trace} />
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function Times({
  isDelayVisible = true,
  name,
  span,
}: {
  isDelayVisible?: boolean;
  name: string;
  span: {
    queuedAt: string;
    startedAt: string | null;
    endedAt: string | null;
  };
}) {
  const queuedAt = new Date(span.queuedAt);
  const startedAt = toMaybeDate(span.startedAt);
  const endedAt = toMaybeDate(span.endedAt);

  const delay = (startedAt ?? new Date()).getTime() - queuedAt.getTime();
  const duration = startedAt ? (endedAt ?? new Date()).getTime() - startedAt.getTime() : 0;

  return (
    <>
      <p className="mb-2 font-bold">{name}</p>
      <div className="flex gap-16">
        <ElementWrapper className="[&>dt]:text-light w-fit" label="Duration">
          {duration > 0 ? formatDuration(duration) : '-'}
        </ElementWrapper>

        <ElementWrapper className="[&>dt]:text-light w-fit" label="Delay">
          {isDelayVisible && delay > 0 ? formatDuration(delay) : '-'}
        </ElementWrapper>
      </div>
    </>
  );
}
