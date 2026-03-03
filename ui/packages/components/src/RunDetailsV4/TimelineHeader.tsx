/**
 * TimelineHeader component - Time Brush for selecting a portion of the timeline.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders:
 * - A time brush with draggable handles at either end
 * - The selected range can be dragged as a whole
 * - Timing markers at 0%, 25%, 50%, 75%, 100% with duration labels
 */

import { useCallback, useLayoutEffect, useRef } from 'react';
import { RiContractUpDownLine, RiExpandUpDownLine } from '@remixicon/react';

import { Button } from '../Button';
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
  /** Current selection start as percentage (0-100), controlled by parent */
  selectionStart?: number;
  /** Current selection end as percentage (0-100), controlled by parent */
  selectionEnd?: number;
  /** Callback to expand all timeline rows */
  onExpandAll?: () => void;
  /** Callback to collapse all timeline rows */
  onCollapseAll?: () => void;
};

const TIME_MARKERS = [0, 25, 50, 75, 100];

/**
 * Calculate duration labels for the time markers.
 */
function getMarkerDurations(totalMs: number): string[] {
  return TIME_MARKERS.map((percent) => formatDuration(Math.floor((totalMs * percent) / 100)));
}

/**
 * Get the base CSS transform for a timestamp label at a given percentage.
 * Labels pin to edges when near 0% or 100%, otherwise center on the handle.
 */
function getLabelTransform(percent: number): string {
  if (percent < 5) return 'translateX(0)';
  if (percent > 95) return 'translateX(-100%)';
  return 'translateX(-50%)';
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
  selectionStart: selStart = 0,
  selectionEnd: selEnd = 100,
  onExpandAll,
  onCollapseAll,
}: Props): JSX.Element {
  const totalMs = maxTime.getTime() - minTime.getTime();
  const durations = getMarkerDurations(totalMs);
  const barColorClass = status ? getStatusBackgroundClass(status) : 'bg-primary-moderate';

  const isDefault = selStart === 0 && selEnd === 100;

  const leftLabelRef = useRef<HTMLDivElement>(null);
  const rightLabelRef = useRef<HTMLDivElement>(null);
  const rightPanelRef = useRef<HTMLDivElement>(null);

  // Prevent timestamp labels from overlapping by measuring after render
  // and pushing them apart before paint.
  useLayoutEffect(() => {
    if (isDefault) return;
    const leftEl = leftLabelRef.current;
    const rightEl = rightLabelRef.current;
    const containerEl = rightPanelRef.current;
    if (!leftEl || !rightEl || !containerEl) return;

    // Reset to base transforms so we can measure natural positions
    const baseLeft = getLabelTransform(selStart);
    const baseRight = getLabelTransform(selEnd);
    leftEl.style.transform = baseLeft;
    rightEl.style.transform = baseRight;

    const containerRect = containerEl.getBoundingClientRect();
    if (containerRect.width === 0) return; // no layout (e.g., jsdom)

    const leftRect = leftEl.getBoundingClientRect();
    const rightRect = rightEl.getBoundingClientRect();
    const gap = 4; // minimum pixel gap between labels

    const overlap = leftRect.right + gap - rightRect.left;
    if (overlap <= 0) return; // no overlap, base transforms are fine

    // Split the push evenly, but respect container edges
    let leftPush = overlap / 2;
    let rightPush = overlap / 2;

    // Left label can't extend past container left edge
    const leftAvail = leftRect.left - containerRect.left;
    if (leftPush > leftAvail) {
      leftPush = Math.max(0, leftAvail);
      rightPush = overlap - leftPush;
    }

    // Right label can't extend past container right edge
    const rightAvail = containerRect.right - rightRect.right;
    if (rightPush > rightAvail) {
      rightPush = Math.max(0, rightAvail);
      leftPush = overlap - rightPush;
    }

    leftEl.style.transform = `${baseLeft} translateX(-${Math.round(leftPush)}px)`;
    rightEl.style.transform = `${baseRight} translateX(${Math.round(rightPush)}px)`;
  }, [selStart, selEnd, isDefault, totalMs]);

  const handleSelectionChange = useCallback(
    (start: number, end: number) => {
      onSelectionChange?.(start, end);
    },
    [onSelectionChange]
  );

  return (
    <div className="mb-1 flex w-full items-end pt-5">
      {/* Left panel with expand/collapse controls */}
      <div
        className="flex shrink-0 items-center gap-0.5 pr-2"
        style={{
          width: `${leftWidth}%`,
          paddingLeft: `${TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX}px`,
        }}
      >
        {onExpandAll && (
          <Button
            size="small"
            appearance="ghost"
            icon={<RiExpandUpDownLine className="h-3.5 w-3.5" />}
            title="Expand all"
            tooltip="Expand all"
            aria-label="Expand all"
            onClick={onExpandAll}
          />
        )}
        {onCollapseAll && (
          <Button
            size="small"
            appearance="ghost"
            icon={<RiContractUpDownLine className="h-3.5 w-3.5" />}
            title="Collapse all"
            tooltip="Collapse all"
            aria-label="Collapse all"
            onClick={onCollapseAll}
          />
        )}
      </div>

      {/* Right panel with brush and markers */}
      <div ref={rightPanelRef} className="relative flex-1">
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
              <span className="text-muted whitespace-nowrap text-xs tabular-nums">
                {durations[i]}
              </span>
            </div>
          ))}
        </div>

        {/* Timestamp labels above drag handles */}
        {!isDefault && (
          <>
            <div
              ref={leftLabelRef}
              data-testid="timestamp-label-left"
              className="bg-canvasBase border-subtle text-basis pointer-events-none absolute z-20 whitespace-nowrap rounded border px-1 py-0.5 text-xs tabular-nums"
              style={{
                left: `${selStart}%`,
                transform: getLabelTransform(selStart),
                top: '-18px',
              }}
            >
              {formatDuration(Math.floor((totalMs * selStart) / 100))}
            </div>
            <div
              ref={rightLabelRef}
              data-testid="timestamp-label-right"
              className="bg-canvasBase border-subtle text-basis pointer-events-none absolute z-20 whitespace-nowrap rounded border px-1 py-0.5 text-xs tabular-nums"
              style={{
                left: `${selEnd}%`,
                transform: getLabelTransform(selEnd),
                top: '-18px',
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
