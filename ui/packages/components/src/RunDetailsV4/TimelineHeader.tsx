/**
 * TimelineHeader component - Time Brush for selecting a portion of the timeline.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders:
 * - A time brush with draggable handles at either end
 * - The selected range can be dragged as a whole
 * - Timing markers at 0%, 25%, 50%, 75%, 100% with duration labels
 */

import { useCallback, useState } from 'react';

import { getStatusBackgroundClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';
import { TimeBrush } from './TimeBrush';
import { formatDuration } from './runDetailsUtils';
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
function getMarkerDurations(totalMs: number): string[] {
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
  const totalMs = maxTime.getTime() - minTime.getTime();
  const durations = getMarkerDurations(totalMs);
  const barColorClass = status ? getStatusBackgroundClass(status) : 'bg-primary-moderate';

  const [selStart, setSelStart] = useState(0);
  const [selEnd, setSelEnd] = useState(100);
  const isDefault = selStart === 0 && selEnd === 100;

  const handleSelectionChange = useCallback(
    (start: number, end: number) => {
      setSelStart(start);
      setSelEnd(end);
      onSelectionChange?.(start, end);
    },
    [onSelectionChange]
  );

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

        {/* Timestamp labels above drag handles */}
        {!isDefault && (
          <>
            <div
              data-testid="timestamp-label-left"
              className="text-basis pointer-events-none absolute z-20 text-xs tabular-nums"
              style={{
                left: `${selStart}%`,
                transform:
                  selStart < 5
                    ? 'translateX(0)'
                    : selStart > 95
                    ? 'translateX(-100%)'
                    : 'translateX(-50%)',
                top: '-4px',
              }}
            >
              {formatDuration(Math.floor((totalMs * selStart) / 100))}
            </div>
            <div
              data-testid="timestamp-label-right"
              className="text-basis pointer-events-none absolute z-20 text-xs tabular-nums"
              style={{
                left: `${selEnd}%`,
                transform:
                  selEnd < 5
                    ? 'translateX(0)'
                    : selEnd > 95
                    ? 'translateX(-100%)'
                    : 'translateX(-50%)',
                top: '-4px',
              }}
            >
              {formatDuration(Math.floor((totalMs * selEnd) / 100))}
            </div>
          </>
        )}

        {/* Time brush */}
        <TimeBrush onSelectionChange={handleSelectionChange} className="mt-1">
          {isDefault ? (
            <div
              data-testid="timeline-bar-default"
              className={cn(
                'pointer-events-none absolute left-0 top-1/2 h-1 w-full -translate-y-1/2',
                barColorClass
              )}
            />
          ) : (
            <>
              <div
                data-testid="bar-segment-left"
                className="bg-canvasMuted pointer-events-none absolute top-1/2 h-1 -translate-y-1/2"
                style={{ left: 0, width: `${selStart}%` }}
              />
              <div
                data-testid="bar-segment-middle"
                className={cn(
                  'pointer-events-none absolute top-1/2 h-1 -translate-y-1/2',
                  barColorClass
                )}
                style={{ left: `${selStart}%`, width: `${selEnd - selStart}%` }}
              />
              <div
                data-testid="bar-segment-right"
                className="bg-canvasMuted pointer-events-none absolute top-1/2 h-1 -translate-y-1/2"
                style={{ left: `${selEnd}%`, width: `${100 - selEnd}%` }}
              />
            </>
          )}
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
