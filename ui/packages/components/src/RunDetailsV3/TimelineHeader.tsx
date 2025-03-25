import { type Trace } from './types';
import { formatDuration } from './utils';

type TimelineHeaderProps = {
  trace: Trace;
  minTime: Date;
  maxTime: Date;
};

const xAxis = [25, 50, 75, 100];

const getEventDurations = (start: Date, end: Date, count: number): string[] => {
  const totalMs = end.getTime() - start.getTime();
  const increment = totalMs / count;

  return Array.from({ length: count }, (_, i) => formatDuration(Math.floor(increment * (i + 1))));
};

export const TimelineHeader = ({ trace, minTime, maxTime }: TimelineHeaderProps) => {
  if (!trace.isRoot || !minTime || !maxTime) {
    return null;
  }

  const durations = getEventDurations(minTime, maxTime, 4);

  return (
    <>
      <div className="text-subtle relative ml-[30%] mt-2 h-3 w-[70%] text-xs leading-none">
        {xAxis.map((x, i) => (
          <div
            key={`x-axis-label-${i}`}
            className="absolute h-7"
            style={{ left: `${x}%`, transform: 'translateX(-50%)' }}
          >
            {durations[i]}
          </div>
        ))}
      </div>

      <div className="pointer-events-none absolute bottom-0 right-0 top-6 w-[70%]">
        {xAxis.map((x, i) => (
          <div
            key={`x-axis-line-${i}`}
            className="bg-canvasSubtle absolute bottom-0 top-0 z-0 w-0.5 bg-opacity-80"
            style={{ left: `${x}%` }}
          />
        ))}
      </div>
    </>
  );
};
