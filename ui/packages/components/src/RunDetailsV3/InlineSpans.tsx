import { Fragment, useCallback, useRef, useState } from 'react';

import { ElementWrapper } from '../DetailsCard/Element';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { Span } from './Span';
import { UserlandSpan } from './UserlandSpan';
import { type Trace } from './types';
import { createSpanWidths, formatDuration, getSpanName } from './utils';

type Props = {
  className?: string;
  maxTime: Date;
  minTime: Date;
  trace: Trace;
};

export function InlineSpans({ className, minTime, maxTime, trace }: Props) {
  const [open, setOpen] = useState(false);
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

  //
  // when a span has step (not userland) children then we construct the span from them
  const stepChildren = trace.childrenSpans?.filter((s) => !s.isUserland) || [];
  const spans = !trace.isRoot && stepChildren.length ? stepChildren : [];

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
        >
          <TooltipTrigger className="flex w-full flex-row">
            {spans.length ? (
              spans.map((span) => {
                return (
                  <Span
                    isInline
                    key={span.spanID}
                    maxTime={maxTime}
                    minTime={minTime}
                    span={span}
                  />
                );
              })
            ) : (
              <Span isInline key={trace.spanID} maxTime={maxTime} minTime={minTime} span={trace} />
            )}
          </TooltipTrigger>
        </div>

        <div className="bg-canvasMuted h-0" style={{ flexGrow: widths.after }} />
      </div>
      <TooltipContent>
        <div className="text-basis">
          <Times isDelayVisible={spans.length === 0} name={spanName} span={trace} />
          {trace.isUserland && trace.userlandAttrs && (
            <UserlandSpan userlandAttrs={trace.userlandAttrs} />
          )}
          {spans.map((span) => {
            return (
              <Fragment key={span.spanID}>
                <hr className="my-2" />
                <Times isDelayVisible={true} name={getSpanName(span.name)} span={span} />
              </Fragment>
            );
          })}
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
        <ElementWrapper className="w-fit" label="Duration">
          {duration > 0 ? formatDuration(duration) : '-'}
        </ElementWrapper>

        <ElementWrapper className="w-fit" label="Delay">
          {isDelayVisible && delay > 0 ? formatDuration(delay) : '-'}
        </ElementWrapper>
      </div>
    </>
  );
}
