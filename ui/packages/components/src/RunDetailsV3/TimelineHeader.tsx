import { type Trace } from './types';

type TimelineHeaderProps = {
  trace: Trace;
  minTime: Date;
  maxTime: Date;
};

const xAxis = [25, 50, 75, 100];

const formatDuration = (ms: number): string => {
  const units = [
    { label: 'd', value: 86400000 }, // 24 * 60 * 60 * 1000
    { label: 'h', value: 3600000 }, // 60 * 60 * 1000
    { label: 'm', value: 60000 }, // 60 * 1000
    { label: 's', value: 1000 }, // 1000
    { label: 'ms', value: 1 },
  ];

  for (const { label, value } of units) {
    if (ms >= value) {
      const amount = Math.floor(ms / value);
      return `${amount}${label}`;
    }
  }

  return '0ms';
};

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
            style={{ left: `calc(${x}% - 0.5rem)` }}
          >
            {durations[i]}
          </div>
        ))}
      </div>

      <div className="pointer-events-none absolute bottom-0 right-0 top-6 w-[70%]">
        {xAxis.map((x, i) => (
          <div
            key={`x-axis-line-${i}`}
            className="bg-canvasMuted absolute bottom-0 top-0 z-50 w-[.5px]"
            style={{ left: `${x}%` }}
          />
        ))}
      </div>
    </>
  );
};
