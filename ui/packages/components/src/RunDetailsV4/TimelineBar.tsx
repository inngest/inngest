/**
 * TimelineBar component - Composable timeline row visualization.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders a single row in the timeline with:
 * - Left panel: name, optional icon, duration
 * - Right panel: visual bar positioned by percentage
 * - Optional expansion to show nested children
 */

import { memo, useMemo, useState, type CSSProperties } from 'react';
import {
  RiArrowRightLine,
  RiArrowRightSFill,
  RiBuilding2Line,
  RiCheckboxCircleFill,
  RiCloseCircleFill,
  RiFlashlightLine,
  RiFunctionLine,
  RiMailLine,
  RiSettings3Line,
  RiStopCircleFill,
  RiTimeLine,
} from '@remixicon/react';
import { format } from 'date-fns';

import { getStatusBackgroundClass, getStatusTextClass } from '../Status/statusClasses';
import { Tooltip, TooltipArrow, TooltipContent, TooltipTrigger } from '../Tooltip/Tooltip';
import { cn } from '../utils/classNames';
import type {
  BarHeight,
  BarIcon,
  BarPattern,
  BarSegment,
  BarStyle,
  BarStyleKey,
  TimelineBarProps,
} from './TimelineBar.types';
import { formatDuration } from './runDetailsUtils';
import { formatLabel } from './utils/formatting';
import { TIMELINE_CONSTANTS } from './utils/timing';

// ============================================================================
// Style Configurations
// ============================================================================

/**
 * Consolidated style configurations for all bar types.
 * Each entry contains visual style, status-based coloring flag, icon, height, and pattern.
 */
const BAR_STYLES: Record<BarStyleKey, BarStyle> = {
  root: {
    barColor: 'bg-status-completed',
    statusBased: true,
  },
  'step.run': {
    barColor: 'bg-status-completed',
    statusBased: true,
  },
  'step.sleep': {
    barColor: 'bg-slate-400',
  },
  'step.waitForEvent': {
    barColor: 'bg-slate-400',
  },
  'step.invoke': {
    barColor: 'bg-slate-400',
  },
  'timing.inngest': {
    barColor: 'bg-slate-300',
    barHeight: 'tall',
    durationColor: 'text-basis',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.server': {
    barColor: 'bg-status-completed',
    barHeight: 'tall',
    durationColor: 'text-basis',
    labelFormat: 'default',
    pattern: 'barber-pole',
    statusBased: true,
    textColor: 'text-light',
  },
  'timing.connecting': {
    barColor: 'bg-transparent',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    barHeight: 'short',
  },
  // HTTP timing phases (children of SERVER bar)
  'timing.http.dns': {
    barColor: 'bg-sky-400',
    barHeight: 'short',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.tcp': {
    barColor: 'bg-cyan-400',
    barHeight: 'short',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.tls': {
    barColor: 'bg-teal-400',
    barHeight: 'short',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.server': {
    barColor: 'bg-emerald-400',
    barHeight: 'short',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.transfer': {
    barColor: 'bg-green-400',
    barHeight: 'short',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  default: {
    barColor: 'bg-slate-400',
    statusBased: true,
  },
};

/**
 * CSS pattern definitions for bar fills.
 * Barber-pole uses semi-transparent white stripes to work on any background color.
 */
const BAR_PATTERNS: Record<BarPattern, CSSProperties> = {
  solid: {},
  'barber-pole': {
    backgroundImage: `repeating-linear-gradient(
      -45deg,
      transparent,
      transparent 6px,
      rgba(255, 255, 255, 0.15) 6px,
      rgba(255, 255, 255, 0.15) 8px
    )`,
  },
  dotted: {
    border: '2px dotted rgb(var(--color-primary-subtle))',
    borderRadius: '2px',
  },
};

/**
 * Get the complete style configuration for a bar style key.
 * Falls back to 'default' if the key is not found.
 */
function getBarStyle(styleKey: BarStyleKey): BarStyle {
  return BAR_STYLES[styleKey] ?? BAR_STYLES.default;
}

/**
 * Get the bar color class, using status-based coloring when appropriate.
 */
function getBarColor(styleKey: BarStyleKey, status?: string): string {
  const barStyle = getBarStyle(styleKey);

  if (status && barStyle.statusBased) {
    return getStatusBackgroundClass(status);
  }

  return barStyle.barColor;
}

/**
 * Get the icon for a root bar based on run status.
 */
function getRootIcon(styleKey: BarStyleKey, status?: string): BarIcon | undefined {
  if (styleKey !== 'root') return undefined;
  switch (status) {
    case 'FAILED':
      return 'close-circle';
    case 'CANCELLED':
      return 'stop-circle';
    default:
      return 'checkbox';
  }
}

/**
 * Get the CSS pattern for a bar pattern type.
 */
function getBarPattern(pattern?: BarPattern): CSSProperties {
  return pattern ? BAR_PATTERNS[pattern] : BAR_PATTERNS.solid;
}

/**
 * Transform bar positions based on view offsets (for zooming).
 * Takes the original bar position and clips/scales it to the visible view window.
 *
 * @param startPercent - Original start position (0-100)
 * @param widthPercent - Original width (0-100)
 * @param viewStartOffset - Start of visible window (0-100, default 0)
 * @param viewEndOffset - End of visible window (0-100, default 100)
 * @returns Transformed start and width for the visible portion, or null if completely outside
 */
function transformBarPosition(
  startPercent: number,
  widthPercent: number,
  viewStartOffset: number = 0,
  viewEndOffset: number = 100
): { startPercent: number; widthPercent: number } | null {
  const barEnd = startPercent + widthPercent;
  const viewWidth = viewEndOffset - viewStartOffset;

  // If view width is 0 or negative, return null
  if (viewWidth <= 0) return null;

  // Check if bar is completely outside the view window
  if (barEnd <= viewStartOffset || startPercent >= viewEndOffset) {
    return null;
  }

  // Clip the bar to the view window
  const clippedStart = Math.max(startPercent, viewStartOffset);
  const clippedEnd = Math.min(barEnd, viewEndOffset);
  const clippedWidth = clippedEnd - clippedStart;

  // Transform to the 0-100 scale of the visible window
  const transformedStart = ((clippedStart - viewStartOffset) / viewWidth) * 100;
  const transformedWidth = (clippedWidth / viewWidth) * 100;

  return {
    startPercent: transformedStart,
    widthPercent: transformedWidth,
  };
}

// ============================================================================
// Icon Mapping
// ============================================================================

const ICON_MAP: Record<BarIcon, React.ComponentType<{ className?: string }>> = {
  gear: RiSettings3Line,
  building: RiBuilding2Line,
  lightning: RiFlashlightLine,
  function: RiFunctionLine,
  clock: RiTimeLine,
  mail: RiMailLine,
  arrow: RiArrowRightLine,
  checkbox: RiCheckboxCircleFill,
  'close-circle': RiCloseCircleFill,
  'stop-circle': RiStopCircleFill,
  none: () => null,
};

/** Icons that should derive their color from run status */
const STATUS_ICONS = new Set<BarIcon>(['checkbox', 'close-circle', 'stop-circle']);

// ============================================================================
// Sub-components
// ============================================================================

/**
 * Renders the icon for a bar based on style or explicit prop.
 */
function BarIconComponent({
  icon,
  className,
  status,
}: {
  icon?: BarIcon;
  className?: string;
  status?: string;
}) {
  if (!icon || icon === 'none') return null;
  const IconComponent = ICON_MAP[icon];
  // Status icons (checkbox, close-circle, stop-circle) derive color from run status
  const statusColor = STATUS_ICONS.has(icon) && status ? getStatusTextClass(status) : undefined;
  return IconComponent ? (
    <IconComponent
      className={cn('h-3.5 w-3.5 shrink-0', className, statusColor)}
      data-testid="bar-icon"
    />
  ) : null;
}

/**
 * Renders the expand/collapse toggle button as a solid triangle.
 */
function ExpandToggle({ expanded, onCollapse }: { expanded: boolean; onCollapse?: () => void }) {
  return (
    <div
      role={expanded ? 'button' : undefined}
      aria-label={expanded ? 'Collapse' : 'Expand'}
      className={cn('flex items-center justify-center p-0', expanded && 'cursor-pointer')}
      onClick={
        expanded
          ? (e) => {
              e.stopPropagation();
              onCollapse?.();
            }
          : undefined
      }
    >
      <RiArrowRightSFill
        className={cn('h-4 w-4 transition-transform', expanded && 'rotate-90')}
        style={{ transitionDuration: `${TIMELINE_CONSTANTS.TRANSITION_MS}ms` }}
      />
    </div>
  );
}

/**
 * Tooltip content for a timeline bar, showing duration/delay and start/end timestamps.
 */
function BarTooltipContent({
  name,
  startTime,
  endTime,
  delayMs,
}: {
  name: string;
  startTime: Date;
  endTime: Date | null;
  delayMs?: number;
}) {
  const startTimestamp = format(startTime, 'yyyy-MM-dd HH:mm:ss.SSS');
  const endTimestamp = endTime ? format(endTime, 'yyyy-MM-dd HH:mm:ss.SSS') : null;

  const durationMs = endTime ? endTime.getTime() - startTime.getTime() : 0;

  return (
    <div className="whitespace-nowrap px-1 py-0.5 text-xs">
      <p className="text-basis mb-1.5 font-medium">{name}</p>
      <div className="flex flex-col gap-1">
        <div className="border-subtle flex flex-col gap-1 border-b pb-1.5">
          <div className="flex justify-between gap-6">
            <span className="text-light font-medium">Duration</span>
            <span className="text-basis tabular-nums">
              {durationMs > 0 ? formatDuration(durationMs) : '-'}
            </span>
          </div>
          {delayMs != null && (
            <div className="flex justify-between gap-6">
              <span className="text-light font-medium">Delay</span>
              <span className="text-basis tabular-nums">
                {delayMs > 0 ? formatDuration(delayMs) : '-'}
              </span>
            </div>
          )}
        </div>
        <div className="mt-0.5 flex justify-between gap-6">
          <span className="text-light font-medium">Start</span>
          <span className="text-basis tabular-nums">{startTimestamp}</span>
        </div>
        <div className="flex justify-between gap-6">
          <span className="text-light font-medium">End</span>
          {endTimestamp !== null ? (
            <span className="text-basis tabular-nums">{endTimestamp}</span>
          ) : (
            <span className="text-light italic">In progress</span>
          )}
        </div>
      </div>
    </div>
  );
}

const BAR_HEIGHT_CLASSES: Record<BarHeight, string> = { short: 'h-2', tall: 'h-4' };

/**
 * Renders the visual bar in the right panel.
 * For compound bars with segments, each segment is independently transformed
 * based on view offsets to only show the visible portion.
 */
const VisualBar = memo(function VisualBar({
  startPercent,
  widthPercent,
  style,
  segments,
  expanded,
  originalBarStart,
  originalBarWidth,
  viewStartOffset = 0,
  viewEndOffset = 100,
  status,
}: {
  startPercent: number;
  widthPercent: number;
  style: TimelineBarProps['style'];
  segments?: BarSegment[];
  expanded?: boolean;
  /** Original bar start before transform (for segment calculation) */
  originalBarStart?: number;
  /** Original bar width before transform (for segment calculation) */
  originalBarWidth?: number;
  /** View start offset for segment filtering */
  viewStartOffset?: number;
  /** View end offset for segment filtering */
  viewEndOffset?: number;
  /** Run status for status-based coloring */
  status?: string;
}) {
  const barStyle = getBarStyle(style);
  const pattern = getBarPattern(barStyle.pattern);
  const heightClass = BAR_HEIGHT_CLASSES[barStyle.barHeight ?? 'tall'];
  const opacityStyle = expanded ? { opacity: 0 } : {};
  const barColor = getBarColor(style, status);

  // Memoize segment transformation to avoid recalculating on every render
  const transformedSegments = useMemo(() => {
    if (!segments || segments.length === 0) return [];

    return segments
      .map((segment) => {
        // Convert segment position from bar-relative to timeline-absolute
        const barStart = originalBarStart ?? 0;
        const barWidth = originalBarWidth ?? 100;
        const segmentAbsoluteStart = barStart + (segment.startPercent / 100) * barWidth;
        const segmentAbsoluteWidth = (segment.widthPercent / 100) * barWidth;

        // Transform to view coordinates
        const transformed = transformBarPosition(
          segmentAbsoluteStart,
          segmentAbsoluteWidth,
          viewStartOffset,
          viewEndOffset
        );

        if (!transformed) return null;

        return {
          ...segment,
          transformedStart: transformed.startPercent,
          transformedWidth: transformed.widthPercent,
        };
      })
      .filter(Boolean);
  }, [segments, originalBarStart, originalBarWidth, viewStartOffset, viewEndOffset]);

  // Render compound bar with segments if provided
  if (segments && segments.length > 0) {
    // If no segments are visible, don't render the container
    if (transformedSegments.length === 0) return null;

    return (
      <div
        data-testid="timeline-bar-visual"
        className="absolute h-full"
        style={{
          left: '0%',
          width: '100%',
          ...opacityStyle,
        }}
      >
        {transformedSegments.map((segment) => {
          if (!segment) return null;
          const segmentStyle = getBarStyle(segment.style);
          const segmentPattern = getBarPattern(segmentStyle.pattern);
          const segmentHeightClass = BAR_HEIGHT_CLASSES[segmentStyle.barHeight ?? 'tall'];
          const segmentColor = getBarColor(segment.style, segment.status);
          return (
            <div
              key={segment.id}
              className={cn('absolute top-1/2 -translate-y-1/2', segmentHeightClass, segmentColor)}
              style={{
                left: `${segment.transformedStart}%`,
                width: `${segment.transformedWidth}%`,
                minWidth: `${TIMELINE_CONSTANTS.MIN_BAR_WIDTH_PX}px`,
                ...segmentPattern,
              }}
            />
          );
        })}
      </div>
    );
  }

  // Render simple bar
  return (
    <div
      data-testid="timeline-bar-visual"
      className={cn('absolute top-1/2 -translate-y-1/2', heightClass, barColor)}
      style={{
        left: `${startPercent}%`,
        width: `${widthPercent}%`,
        minWidth: `${TIMELINE_CONSTANTS.MIN_BAR_WIDTH_PX}px`,
        ...pattern,
        ...opacityStyle,
      }}
    />
  );
});

// ============================================================================
// Main Component
// ============================================================================

/**
 * TimelineBar component renders a single row in the timeline visualization.
 *
 * Features:
 * - Configurable positioning via startPercent/widthPercent
 * - Style-based visual appearance
 * - Optional expand/collapse for nested children
 * - Optional icon display
 * - Depth-based indentation
 * - Selection highlighting
 */
export function TimelineBar({
  name,
  duration,
  icon,
  startPercent,
  widthPercent,
  depth,
  leftWidth,
  style,
  segments,
  expandable,
  expanded,
  onToggle,
  onClick,
  selected,
  children,
  orgName,
  status,
  viewStartOffset = 0,
  viewEndOffset = 100,
  startTime,
  endTime,
  delayMs,
}: TimelineBarProps): JSX.Element {
  const barStyle = getBarStyle(style);
  const effectiveIcon = icon ?? barStyle.icon ?? getRootIcon(style, status);

  // Format the display name based on style
  let displayName = name;
  if (barStyle.labelFormat) {
    displayName = formatLabel(name, barStyle.labelFormat);
  }

  // For SERVER timing, show org name or fallback
  if (style === 'timing.server') {
    displayName = orgName ? `${orgName} server` : 'Your server';
  }

  // Calculate indentation (base padding + depth-based indent)
  const indentPx =
    TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX + depth * TIMELINE_CONSTANTS.INDENT_WIDTH_PX;

  // Transform bar position based on view offsets
  const transformed = useMemo(
    () => transformBarPosition(startPercent, widthPercent, viewStartOffset, viewEndOffset),
    [startPercent, widthPercent, viewStartOffset, viewEndOffset]
  );

  // Tooltip state — controlled so hover target (full right panel) is separate from anchor (bar position)
  const showTooltip = !!startTime;
  const [tooltipOpen, setTooltipOpen] = useState(false);

  return (
    <div data-testid="timeline-bar-container" className="relative">
      {/* Main row */}
      <div
        data-testid="timeline-bar-row"
        className="relative isolate flex h-7 cursor-pointer items-center"
        onClick={() => {
          onClick?.();
          if (expandable && !expanded) {
            onToggle?.();
          }
        }}
        style={{ height: `${TIMELINE_CONSTANTS.ROW_HEIGHT_PX}px` }}
      >
        {/* Selection / hover highlight - extends from indent to full width */}
        {(selected || tooltipOpen) && (
          <div
            className={cn(
              'pointer-events-none absolute inset-y-0 right-0 -z-10',
              selected ? 'bg-secondary-3xSubtle' : 'bg-canvasSubtle'
            )}
            style={{
              left: `${indentPx - 4}px`,
            }}
          />
        )}
        {/* Left panel - name, icon, controls */}
        <div
          data-testid="timeline-bar-left"
          className="flex h-full shrink-0 items-center gap-1.5 overflow-hidden pr-4"
          style={{
            width: `${leftWidth}%`,
            paddingLeft: `${indentPx}px`,
          }}
        >
          {/* Expand toggle */}
          {expandable && <ExpandToggle expanded={expanded ?? false} onCollapse={onToggle} />}

          {/* Icon */}
          <BarIconComponent icon={effectiveIcon} className="text-subtle ml-px" status={status} />

          {/* Name */}
          <span
            className={cn(
              'min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-xs font-normal leading-tight',
              barStyle.textColor ?? 'text-basis',
              !expandable && !effectiveIcon && 'pl-1.5'
            )}
          >
            {displayName}
          </span>

          {/* Duration */}
          <span
            className={cn(
              'shrink-0 text-xs font-medium tabular-nums',
              barStyle.durationColor ?? barStyle.textColor ?? 'text-basis'
            )}
          >
            {formatDuration(duration)}
          </span>
        </div>

        {/* Right panel - visual bar with optional hover tooltip */}
        <div
          data-testid="timeline-bar-right"
          className="relative h-full flex-1"
          style={{ width: `${100 - leftWidth}%` }}
          onMouseEnter={showTooltip ? () => setTooltipOpen(true) : undefined}
          onMouseLeave={showTooltip ? () => setTooltipOpen(false) : undefined}
        >
          {/* Center line */}
          <div className="bg-canvasMuted absolute left-0 right-0 top-1/2 h-px -translate-y-1/2" />
          {/* Bar container, centered vertically */}
          <div className="absolute inset-y-0 flex w-full items-center">
            {transformed && (
              <>
                <VisualBar
                  startPercent={transformed.startPercent}
                  widthPercent={transformed.widthPercent}
                  style={style}
                  segments={segments}
                  expanded={!!(expandable && expanded)}
                  originalBarStart={startPercent}
                  originalBarWidth={widthPercent}
                  viewStartOffset={viewStartOffset}
                  viewEndOffset={viewEndOffset}
                  status={status}
                />
                {showTooltip && (
                  <Tooltip open={tooltipOpen}>
                    <TooltipTrigger asChild>
                      <div
                        className="pointer-events-none absolute inset-y-0"
                        style={{
                          left: `${transformed.startPercent}%`,
                          width: `${transformed.widthPercent}%`,
                          minWidth: '4px',
                        }}
                      />
                    </TooltipTrigger>
                    <TooltipContent
                      side="top"
                      hasArrow={false}
                      className="bg-canvasBase text-basis border-muted max-w-none border shadow-lg"
                    >
                      <BarTooltipContent
                        name={displayName}
                        startTime={startTime!}
                        endTime={endTime ?? null}
                        delayMs={delayMs}
                      />
                      <TooltipArrow className="fill-canvasBase" />
                    </TooltipContent>
                  </Tooltip>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Vertical guide line from arrow to bottom of expanded area */}
      {expanded && (
        <div
          className="bg-canvasMuted absolute w-px"
          style={{
            left: `${indentPx + 8}px`,
            top: `${TIMELINE_CONSTANTS.ROW_HEIGHT_PX}px`,
            bottom: 0,
          }}
        />
      )}

      {/* Children (expanded content) */}
      {expanded && children}
    </div>
  );
}
