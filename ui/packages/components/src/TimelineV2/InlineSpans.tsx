import { Fragment } from 'react';

import { ElementWrapper } from '../DetailsCard/Element';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { Span } from './Span';

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
  let delay = 0;
  if (spans.length === 1) {
    const startedAt = toMaybeDate(spans[0]?.startedAt);
    delay = (startedAt ?? new Date()).getTime() - minTime.getTime();
  }

  let duration = 0;
  spans.forEach((span) => {
    const startedAt = toMaybeDate(span.startedAt)?.getTime();
    const endedAt = toMaybeDate(span.endedAt)?.getTime();
    if (startedAt && endedAt) {
      duration += endedAt - startedAt;
    }
  });

  return (
    <Tooltip>
      <TooltipTrigger className="h-fit grow">
        <div className={cn('flex h-8 grow items-center', className)}>
          <div className="bg-contrast h-px" style={{ flexGrow: widths.before }}></div>

          <div
            className="flex"
            style={{
              flexGrow: widths.queued + widths.running,
            }}
          >
            {spans.map((item) => {
              return (
                <Span isInline key={item.spanID} maxTime={maxTime} minTime={minTime} trace={item} />
              );
            })}
          </div>

          <div className="bg-contrast h-px" style={{ flexGrow: widths.after }}></div>
        </div>
      </TooltipTrigger>
      <TooltipContent>
        <div className="text-basis">
          {spans[0] && <Times isDelayVisible={spans.length === 1} name={name} span={spans[0]} />}

          {spans.length > 1 &&
            spans.map((span) => {
              return (
                <Fragment key={span.spanID}>
                  <hr className="my-2" />
                  <Times name={span.name} span={span} />
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

  let duration = 0;
  if (startedAt) {
    duration = (endedAt ?? new Date()).getTime() - startedAt.getTime();
  }

  return (
    <>
      <p className="mb-2 font-bold">{name}</p>
      <div className="flex gap-16">
        <ElementWrapper className="w-fit" label="Duration">
          {duration > 0 ? formatMilliseconds(duration) : '-'}
        </ElementWrapper>

        <ElementWrapper className="w-fit" label="Delay">
          {isDelayVisible && delay > 0 ? formatMilliseconds(delay) : '-'}
        </ElementWrapper>
      </div>
    </>
  );
}
