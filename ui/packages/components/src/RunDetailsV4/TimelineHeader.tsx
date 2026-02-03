/**
 * TimelineHeader component - Time Brush for selecting a portion of the timeline.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders:
 * - A time brush with draggable handles at either end
 * - The selected range can be dragged as a whole
 * - Timing markers at 0%, 25%, 50%, 75%, 100% with duration labels
 */

import { getStatusBackgroundClass } from '../Status/statusClasses';
import { TimeBrush } from '../TimeBrush';
import { cn } from '../utils/classNames';
import { formatDuration } from './utils/formatting';
import { TIMELINE_CONSTANTS } from './utils/timing';

type Props = {
  /** Minimum time (run start) */
  minTime: Date;
  /** Maximum time (run end) */
  maxTime: Date;
  /** Width of the left panel as percentage (0-100) */
  leftWidth: number;
  /** Callback when selection changes (start and end as percentages 0-100) */
  onSelectionChange?: (start: number, end: number) => void;
  /** Run status for status-based coloring (e.g., COMPLETED, FAILED, CANCELLED) */
  status?: string;
};

const TIME_MARKERS = [0, 25, 50, 75, 100];

/**
 * Calculate duration labels for the time markers.
 */
function getMarkerDurations(minTime: Date, maxTime: Date): string[] {
  const totalMs = maxTime.getTime() - minTime.getTime();
  return TIME_MARKERS.map((percent) => formatDuration(Math.floor((totalMs * percent) / 100)));
}

/**
 * TimelineHeader displays a time brush for selecting a portion of the timeline.
 */
export function TimelineHeader({
  minTime,
  maxTime,
  leftWidth,
  onSelectionChange,
  status,
}: Props): JSX.Element {
  const durations = getMarkerDurations(minTime, maxTime);
  const barColorClass = status ? getStatusBackgroundClass(status) : 'bg-primary-moderate';

  return (
    <div className="mb-1 mt-5 flex w-full items-end">
      {/* Left panel spacer (no label - label is now in the Run bar) */}
      <div
        className="shrink-0 pr-2"
        style={{
          width: `${leftWidth}%`,
          paddingLeft: `${TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX}px`,
        }}
      />

      {/* Right panel with brush and markers */}
      <div className="relative flex-1">
        {/* Time markers - positioned above the bar */}
        <div className="relative w-full">
          {TIME_MARKERS.map((percent, i) => (
            <div
              key={`marker-${percent}`}
              className="absolute bottom-0 flex flex-col items-center"
              style={{
                left: `${percent}%`,
                transform:
                  percent === 0
                    ? 'translateX(0)'
                    : percent === 100
                    ? 'translateX(-100%)'
                    : 'translateX(-50%)',
              }}
            >
              {/* Duration label */}
              <span className="text-muted text-xs tabular-nums">{durations[i]}</span>
            </div>
          ))}
        </div>

        {/* Time brush */}
        <TimeBrush onSelectionChange={onSelectionChange} className="mt-1">
          {/* Main timeline bar (always full width) */}
          <div
            className={cn(
              'pointer-events-none absolute left-0 top-1/2 h-1 w-full -translate-y-1/2',
              barColorClass
            )}
          />
        </TimeBrush>

        {/* Vertical guide lines (subtle) */}
        <div className="pointer-events-none absolute bottom-0 left-0 right-0 top-0">
          {TIME_MARKERS.slice(1).map((percent) => (
            <div
              key={`line-${percent}`}
              className="bg-canvasSubtle absolute bottom-0 top-0 w-px opacity-50"
              style={{ left: `${percent}%` }}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
