import { Fragment, useCallback, useRef, useState } from 'react';

import { ElementWrapper } from '../DetailsCard/Element';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { cn } from '../utils/classNames';
import { toMaybeDate } from '../utils/date';
import { Span } from './Span';
import { formatDuration, getSpanName } from './utils';

type Props = {
  className?: string;
  maxTime: Date;
  minTime: Date;
  name: string;
  spans: (React.ComponentProps<typeof Span>['trace'] & { name: string })[];
  widths: {
    before: number;
    queued: number;
    running: number;
    after: number;
  };
};

export function InlineSpans({ className, minTime, maxTime, name, spans, widths }: Props) {
  const [open, setOpen] = useState(false);
  const hoverTimeoutRef = useRef<NodeJS.Timeout>();
  const spanName = getSpanName(name);

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
            {spans.map((item) => {
              return (
                <Span isInline key={item.spanID} maxTime={maxTime} minTime={minTime} trace={item} />
              );
            })}
          </TooltipTrigger>
        </div>

        <div className="bg-canvasMuted h-0" style={{ flexGrow: widths.after }}></div>
      </div>
      <TooltipContent>
        <div className="text-basis">
          {spans[0] && (
            <Times isDelayVisible={spans.length === 1} name={spanName} span={spans[0]} />
          )}

          {spans.length > 1 &&
            spans.map((span) => {
              return (
                <Fragment key={span.spanID}>
                  <hr className="my-2" />
                  <Times name={spanName} span={span} />
                  {span.spanID}
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
