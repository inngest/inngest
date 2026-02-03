/**
 * Timeline container component - Renders a complete timeline visualization.
 * Feature: 001-composable-timeline-bar
 *
 * This component manages:
 * - Converting timeline data to TimelineBar props
 * - Expansion state for all expandable bars
 * - Recursive rendering of nested steps
 * - Column resize handling
 */

import { useCallback, useState } from 'react';

import { TimelineBar } from './TimelineBar';
import type { BarSegment, TimelineBarData, TimelineData } from './TimelineBar.types';
import { TimelineHeader } from './TimelineHeader';
import { calculateBarPosition, calculateDuration } from './utils/timing';

// ============================================================================
// Types
// ============================================================================

type Props = {
  /** Timeline data to render */
  data: TimelineData;
  /** Callback when a step is selected */
  onSelectStep?: (stepId: string) => void;
};

// ============================================================================
// Timing Breakdown Utilities
// ============================================================================

/**
 * Generate timing breakdown bars for a step with timing data.
 */
function generateTimingBreakdownBars(bar: TimelineBarData, orgName?: string): TimelineBarData[] {
  if (!bar.timingBreakdown) return [];

  const { queueMs, executionMs } = bar.timingBreakdown;
  const barStartTime = bar.startTime;

  // Calculate queue end time
  const queueEndTime = new Date(barStartTime.getTime() + queueMs);
  const executionEndTime = new Date(queueEndTime.getTime() + executionMs);

  const timingBars: TimelineBarData[] = [];

  // INNGEST timing bar (queue time)
  if (queueMs > 0) {
    timingBars.push({
      id: `${bar.id}-timing-inngest`,
      name: 'INNGEST',
      startTime: barStartTime,
      endTime: queueEndTime,
      style: 'timing.inngest',
    });
  }

  // SERVER timing bar (execution time)
  if (executionMs > 0) {
    timingBars.push({
      id: `${bar.id}-timing-server`,
      name: orgName ?? 'YOUR SERVER',
      startTime: queueEndTime,
      endTime: executionEndTime,
      style: 'timing.server',
    });
  }

  return timingBars;
}

/**
 * Generate segments for a compound bar based on timing breakdown.
 */
function generateBarSegments(bar: TimelineBarData): BarSegment[] | undefined {
  if (!bar.timingBreakdown) return undefined;

  const { queueMs, executionMs, totalMs } = bar.timingBreakdown;

  if (totalMs <= 0) return undefined;

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  // Queue segment (INNGEST)
  if (queueMs > 0) {
    const queuePercent = (queueMs / totalMs) * 100;
    segments.push({
      id: `${bar.id}-seg-inngest`,
      startPercent: currentPercent,
      widthPercent: queuePercent,
      style: 'timing.inngest',
    });
    currentPercent += queuePercent;
  }

  // Execution segment (SERVER)
  if (executionMs > 0) {
    const execPercent = (executionMs / totalMs) * 100;
    segments.push({
      id: `${bar.id}-seg-server`,
      startPercent: currentPercent,
      widthPercent: execPercent,
      style: 'timing.server',
    });
  }

  return segments.length > 0 ? segments : undefined;
}

// ============================================================================
// Timeline Bar Renderer
// ============================================================================

type TimelineBarRendererProps = {
  bar: TimelineBarData;
  depth: number;
  minTime: Date;
  maxTime: Date;
  leftWidth: number;
  orgName?: string;
  expandedBars: Set<string>;
  onToggleExpand: (barId: string) => void;
  onSelectStep?: (stepId: string) => void;
  selectedStepId?: string;
  /** View offset - start position as percentage (0-100) for zooming */
  viewStartOffset?: number;
  /** View offset - end position as percentage (0-100) for zooming */
  viewEndOffset?: number;
};

/**
 * Recursively renders a bar and its children/timing breakdown.
 */
function TimelineBarRenderer({
  bar,
  depth,
  minTime,
  maxTime,
  leftWidth,
  orgName,
  expandedBars,
  onToggleExpand,
  onSelectStep,
  selectedStepId,
  viewStartOffset = 0,
  viewEndOffset = 100,
}: TimelineBarRendererProps): JSX.Element {
  const { startPercent, widthPercent } = calculateBarPosition(
    bar.startTime,
    bar.endTime,
    minTime,
    maxTime
  );

  const duration = calculateDuration(bar.startTime, bar.endTime);
  const hasTimingBreakdown = !!bar.timingBreakdown;
  const hasChildren = bar.children && bar.children.length > 0;
  const isExpandable = hasTimingBreakdown || hasChildren;
  const isExpanded = expandedBars.has(bar.id);

  // Generate segments for compound bar visualization
  const segments = generateBarSegments(bar);

  // Generate timing breakdown bars for expanded view
  const timingBars = hasTimingBreakdown ? generateTimingBreakdownBars(bar, orgName) : [];

  return (
    <TimelineBar
      name={bar.name}
      duration={duration}
      startPercent={startPercent}
      widthPercent={widthPercent}
      depth={depth}
      leftWidth={leftWidth}
      style={bar.style}
      segments={segments}
      expandable={isExpandable}
      expanded={isExpanded}
      onToggle={() => onToggleExpand(bar.id)}
      onClick={() => onSelectStep?.(bar.id)}
      selected={selectedStepId === bar.id}
      orgName={orgName}
      viewStartOffset={viewStartOffset}
      viewEndOffset={viewEndOffset}
    >
      {/* Timing breakdown bars */}
      {isExpanded &&
        timingBars.map((timingBar) => {
          const timingPosition = calculateBarPosition(
            timingBar.startTime,
            timingBar.endTime,
            minTime,
            maxTime
          );
          const timingDuration = calculateDuration(timingBar.startTime, timingBar.endTime);

          // Timing bars are only expandable when there are children to show
          const isServerBar = timingBar.style === 'timing.server';
          const isTimingBarExpandable = isServerBar && hasChildren;
          const isTimingBarExpanded = isTimingBarExpandable && expandedBars.has(timingBar.id);

          return (
            <TimelineBar
              key={timingBar.id}
              name={timingBar.name}
              duration={timingDuration}
              startPercent={timingPosition.startPercent}
              widthPercent={timingPosition.widthPercent}
              depth={depth + 1}
              leftWidth={leftWidth}
              style={timingBar.style}
              orgName={orgName}
              expandable={isTimingBarExpandable}
              expanded={isTimingBarExpanded}
              onToggle={isTimingBarExpandable ? () => onToggleExpand(timingBar.id) : undefined}
              viewStartOffset={viewStartOffset}
              viewEndOffset={viewEndOffset}
            >
              {/* Child bars nested under YOUR SERVER */}
              {isTimingBarExpanded &&
                isServerBar &&
                (hasChildren ? (
                  bar.children?.map((child) => (
                    <TimelineBarRenderer
                      key={child.id}
                      bar={child}
                      depth={depth + 2}
                      minTime={minTime}
                      maxTime={maxTime}
                      leftWidth={leftWidth}
                      orgName={orgName}
                      expandedBars={expandedBars}
                      onToggleExpand={onToggleExpand}
                      onSelectStep={onSelectStep}
                      selectedStepId={selectedStepId}
                      viewStartOffset={viewStartOffset}
                      viewEndOffset={viewEndOffset}
                    />
                  ))
                ) : (
                  <div
                    className="text-muted flex items-center py-1 text-xs"
                    style={{ paddingLeft: `${(depth + 2) * 40}px` }}
                  >
                    Executing step code
                  </div>
                ))}
            </TimelineBar>
          );
        })}

      {/* Child bars (when no timing breakdown, render directly) */}
      {isExpanded &&
        !hasTimingBreakdown &&
        bar.children?.map((child) => (
          <TimelineBarRenderer
            key={child.id}
            bar={child}
            depth={depth + 1}
            minTime={minTime}
            maxTime={maxTime}
            leftWidth={leftWidth}
            orgName={orgName}
            expandedBars={expandedBars}
            onToggleExpand={onToggleExpand}
            onSelectStep={onSelectStep}
            selectedStepId={selectedStepId}
            viewStartOffset={viewStartOffset}
            viewEndOffset={viewEndOffset}
          />
        ))}
    </TimelineBar>
  );
}

// ============================================================================
// Main Component
// ============================================================================

/**
 * Timeline container component that renders a complete timeline visualization
 * using the composable TimelineBar component.
 *
 * Features:
 * - Manages expansion state for all expandable bars
 * - Renders timing breakdowns when bars are expanded
 * - Supports nested children (recursive rendering)
 * - Column resize handling (planned)
 */
export function Timeline({ data, onSelectStep }: Props): JSX.Element {
  const { minTime, maxTime, bars, leftWidth, orgName } = data;
  // Initialize with root bars expanded by default
  const [expandedBars, setExpandedBars] = useState<Set<string>>(() => {
    const rootBarIds = bars.filter((bar) => bar.isRoot).map((bar) => bar.id);
    return new Set(rootBarIds);
  });
  const [selectedStepId, setSelectedStepId] = useState<string | undefined>();

  // Timeline brush selection state (for zooming)
  const [viewStartOffset, setViewStartOffset] = useState(0);
  const [viewEndOffset, setViewEndOffset] = useState(100);

  const handleToggleExpand = useCallback((barId: string) => {
    setExpandedBars((prev) => {
      const next = new Set(prev);
      if (next.has(barId)) {
        next.delete(barId);
      } else {
        next.add(barId);
      }
      return next;
    });
  }, []);

  const handleSelectStep = useCallback(
    (stepId: string) => {
      setSelectedStepId(stepId);
      onSelectStep?.(stepId);
    },
    [onSelectStep]
  );

  // Handle timeline brush selection change
  const handleSelectionChange = useCallback((start: number, end: number) => {
    setViewStartOffset(start);
    setViewEndOffset(end);
  }, []);

  return (
    <div className="w-full pb-4 pr-2" data-testid="timeline-container">
      {/* Run duration header with timing markers */}
      <TimelineHeader
        minTime={minTime}
        maxTime={maxTime}
        leftWidth={leftWidth}
        onSelectionChange={handleSelectionChange}
      />

      {/* Step bars */}
      {bars.map((bar) => (
        <TimelineBarRenderer
          key={bar.id}
          bar={bar}
          depth={0}
          minTime={minTime}
          maxTime={maxTime}
          leftWidth={leftWidth}
          orgName={orgName}
          expandedBars={expandedBars}
          onToggleExpand={handleToggleExpand}
          onSelectStep={handleSelectStep}
          selectedStepId={selectedStepId}
          viewStartOffset={viewStartOffset}
          viewEndOffset={viewEndOffset}
        />
      ))}
    </div>
  );
}
