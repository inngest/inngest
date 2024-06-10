import { cn } from '../utils/classNames';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { Span } from './Span';

type Props = {
  className?: string;
  maxTime: Date;
  minTime: Date;
  spans: React.ComponentProps<typeof Span>['trace'][];
  widths: {
    before: number;
    queued: number;
    running: number;
    after: number;
  };
};

export function InlineSpans({ className, minTime, maxTime, spans, widths }: Props) {
  const startedAt = toMaybeDate(spans[0]?.startedAt);
  let durationText;
  if (startedAt) {
    const endedAt = toMaybeDate(spans[spans.length - 1]?.endedAt) ?? maxTime;
    durationText = formatMilliseconds(endedAt.getTime() - startedAt.getTime());
  }

  return (
    <div className={cn('flex h-fit grow items-center', className)}>
      <div className="h-px bg-slate-300" style={{ flexGrow: widths.before }}></div>

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

      <div className="h-px bg-slate-300" style={{ flexGrow: widths.after }}></div>
    </div>
  );
}
