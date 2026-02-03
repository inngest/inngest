/**
 * TimelineBar component - Composable timeline row visualization.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders a single row in the timeline with:
 * - Left panel: name, optional icon, duration
 * - Right panel: visual bar positioned by percentage
 * - Optional expansion to show nested children
 */

import { useMemo, type CSSProperties } from 'react';
import {
  RiArrowRightLine,
  RiBuilding2Line,
  RiCheckboxCircleFill,
  RiFlashlightLine,
  RiFunctionLine,
  RiMailLine,
  RiSettings3Line,
  RiTimeLine,
} from '@remixicon/react';

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
import { formatDuration, formatLabel } from './utils/formatting';
import { TIMELINE_CONSTANTS } from './utils/timing';

// ============================================================================
// Style Configurations
// ============================================================================

/**
 * Style configurations for all bar types.
 * Colors use status-based semantics from the Status system for consistency.
 */
const BAR_STYLES: Record<BarStyleKey, BarStyle> = {
  // Root run bar - uses status color (green for completed)
  root: {
    barColor: 'bg-status-completed',
    icon: 'checkbox',
  },

  // Step types - step.run uses status color, others use default fallback (pending design)
  'step.run': {
    barColor: 'bg-status-completed',
  },
  'step.sleep': {
    barColor: 'bg-slate-400', // Fallback style (pending design)
  },
  'step.waitForEvent': {
    barColor: 'bg-slate-400', // Fallback style (pending design)
  },
  'step.invoke': {
    barColor: 'bg-slate-400', // Fallback style (pending design)
  },

  // Timing categories
  'timing.inngest': {
    barColor: 'bg-slate-300',
    labelFormat: 'uppercase',
    barHeight: 'short',
  },
  'timing.server': {
    barColor: 'bg-status-completed',
    pattern: 'barber-pole',
    labelFormat: 'uppercase',
    barHeight: 'tall',
  },
  'timing.connecting': {
    barColor: 'bg-transparent',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    barHeight: 'short',
  },

  // Fallback
  default: {
    barColor: 'bg-slate-400',
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
      rgba(255, 255, 255, 0.1) 6px,
      rgba(255, 255, 255, 0.1) 8px
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
  none: () => null,
};

// ============================================================================
// Sub-components
// ============================================================================

/**
 * Renders the icon for a bar based on style or explicit prop.
 */
function BarIconComponent({ icon, className }: { icon?: BarIcon; className?: string }) {
  if (!icon || icon === 'none') return null;
  const IconComponent = ICON_MAP[icon];
  // Checkbox icon is green to match the root run bar
  const iconColor = icon === 'checkbox' ? 'text-primary-moderate' : className;
  return IconComponent ? (
    <IconComponent className={cn('h-3.5 w-3.5 shrink-0', iconColor)} data-testid="bar-icon" />
  ) : null;
}

/**
 * Renders the expand/collapse toggle button as a solid triangle.
 */
function ExpandToggle({ expanded, onToggle }: { expanded: boolean; onToggle?: () => void }) {
  return (
    <button
      type="button"
      aria-label={expanded ? 'Collapse' : 'Expand'}
      className="flex cursor-pointer items-center justify-center border-none bg-transparent p-0"
      onClick={(e) => {
        e.stopPropagation();
        onToggle?.();
      }}
    >
      <span
        className={cn('block transition-transform', expanded && 'rotate-90')}
        style={{
          transitionDuration: `${TIMELINE_CONSTANTS.TRANSITION_MS}ms`,
          width: 0,
          height: 0,
          borderTop: '4px solid transparent',
          borderBottom: '4px solid transparent',
          borderLeft: '6px solid currentColor',
        }}
      />
    </button>
  );
}

/**
 * Height class mapping for bar variants.
 */
const BAR_HEIGHT_CLASSES: Record<BarHeight, string> = {
  short: 'h-2',
  tall: 'h-5',
};

/**
 * Renders the visual bar in the right panel.
 * For compound bars with segments, each segment is independently transformed
 * based on view offsets to only show the visible portion.
 */
function VisualBar({
  startPercent,
  widthPercent,
  style,
  segments,
  expanded,
  originalBarStart,
  originalBarWidth,
  viewStartOffset = 0,
  viewEndOffset = 100,
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
}) {
  const barStyle = getBarStyle(style);
  const pattern = getBarPattern(barStyle.pattern);
  const heightClass = BAR_HEIGHT_CLASSES[barStyle.barHeight ?? 'tall'];
  const opacityStyle = expanded ? { opacity: 0.5 } : {};

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
          return (
            <div
              key={segment.id}
              className={cn(
                'absolute top-1/2 -translate-y-1/2',
                segmentHeightClass,
                segmentStyle.barColor
              )}
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
      className={cn('absolute top-1/2 -translate-y-1/2', heightClass, barStyle.barColor)}
      style={{
        left: `${startPercent}%`,
        width: `${widthPercent}%`,
        minWidth: `${TIMELINE_CONSTANTS.MIN_BAR_WIDTH_PX}px`,
        ...pattern,
        ...opacityStyle,
      }}
    />
  );
}

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
  viewStartOffset = 0,
  viewEndOffset = 100,
}: TimelineBarProps): JSX.Element {
  const barStyle = getBarStyle(style);
  const effectiveIcon = icon ?? barStyle.icon;

  // Format the display name based on style
  let displayName = name;
  if (barStyle.labelFormat) {
    displayName = formatLabel(name, barStyle.labelFormat);
  }

  // For SERVER timing, show org name or fallback
  if (style === 'timing.server') {
    displayName = orgName ? orgName.toUpperCase() : 'YOUR SERVER';
  }

  // Calculate indentation (base padding + depth-based indent)
  const indentPx =
    TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX + depth * TIMELINE_CONSTANTS.INDENT_WIDTH_PX;

  return (
    <div data-testid="timeline-bar-container">
      {/* Main row */}
      <div
        data-testid="timeline-bar-row"
        className={cn('flex h-7 cursor-pointer items-center', selected && 'bg-canvasSubtle')}
        onClick={onClick}
        style={{ height: `${TIMELINE_CONSTANTS.ROW_HEIGHT_PX}px` }}
      >
        {/* Left panel - name, icon, controls */}
        <div
          data-testid="timeline-bar-left"
          className="flex h-full shrink-0 items-center gap-1.5 overflow-hidden pr-2"
          style={{
            width: `${leftWidth}%`,
            paddingLeft: `${indentPx}px`,
          }}
        >
          {/* Expand toggle */}
          {expandable && <ExpandToggle expanded={expanded ?? false} onToggle={onToggle} />}

          {/* Icon */}
          <BarIconComponent icon={effectiveIcon} className="text-subtle" />

          {/* Name */}
          <span
            className={cn(
              'text-basis min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-normal leading-tight',
              !expandable && !effectiveIcon && 'pl-1.5'
            )}
          >
            {displayName}
          </span>

          {/* Duration */}
          <span className="text-muted shrink-0 text-xs tabular-nums">
            {formatDuration(duration)}
          </span>
        </div>

        {/* Right panel - visual bar */}
        <div
          data-testid="timeline-bar-right"
          className="relative h-full flex-1"
          style={{ width: `${100 - leftWidth}%` }}
        >
          {/* Center line */}
          <div className="bg-canvasMuted absolute left-0 right-0 top-1/2 h-px -translate-y-1/2" />
          {/* Bar container - centered vertically */}
          <div className="absolute inset-y-0 flex w-full items-center">
            {(() => {
              // Transform bar position based on view offsets
              const transformed = transformBarPosition(
                startPercent,
                widthPercent,
                viewStartOffset,
                viewEndOffset
              );

              // Don't render bar if completely outside the view
              if (!transformed) return null;

              return (
                <VisualBar
                  startPercent={transformed.startPercent}
                  widthPercent={transformed.widthPercent}
                  style={style}
                  segments={segments}
                  expanded={expandable && expanded}
                  originalBarStart={startPercent}
                  originalBarWidth={widthPercent}
                  viewStartOffset={viewStartOffset}
                  viewEndOffset={viewEndOffset}
                />
              );
            })()}
          </div>
        </div>
      </div>

      {/* Children (expanded content) */}
      {expandable && expanded && children}
    </div>
  );
}
