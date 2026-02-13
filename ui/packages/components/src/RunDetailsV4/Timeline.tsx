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
import type {
  BarSegment,
  BarStyleKey,
  HTTPTimingBreakdownData,
  TimelineBarData,
  TimelineData,
} from './TimelineBar.types';
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
      name: 'Inngest',
      startTime: barStartTime,
      endTime: queueEndTime,
      style: 'timing.inngest',
    });
  }

  // SERVER timing bar (execution time)
  if (executionMs > 0) {
    timingBars.push({
      id: `${bar.id}-timing-server`,
      name: orgName ?? 'Your server',
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
      status: bar.status,
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
      status: bar.status,
    });
  }

  return segments.length > 0 ? segments : undefined;
}

/**
 * Generate HTTP timing breakdown bars for a step with HTTP timing metadata.
 * These are sequential phases of Inngest's HTTP call to the SDK endpoint:
 * DNS -> TCP -> TLS -> Server Processing (TTFB) -> Content Transfer
 *
 * Only non-zero phases are included.
 */
function generateHTTPTimingBars(
  parentBarId: string,
  httpTiming: HTTPTimingBreakdownData,
  parentStartTime: Date
): TimelineBarData[] {
  const phases: { key: string; label: string; ms: number; style: BarStyleKey }[] = [
    { key: 'dns', label: 'DNS', ms: httpTiming.dnsLookupMs, style: 'timing.http.dns' },
    { key: 'tcp', label: 'TCP', ms: httpTiming.tcpConnectionMs, style: 'timing.http.tcp' },
    { key: 'tls', label: 'TLS', ms: httpTiming.tlsHandshakeMs, style: 'timing.http.tls' },
    {
      key: 'server',
      label: 'TTFB',
      ms: httpTiming.serverProcessingMs,
      style: 'timing.http.server',
    },
    {
      key: 'transfer',
      label: 'Transfer',
      ms: httpTiming.contentTransferMs,
      style: 'timing.http.transfer',
    },
  ];

  let cumulativeMs = 0;

  return phases
    .filter((phase) => phase.ms > 0)
    .map((phase) => {
      const startTime = new Date(parentStartTime.getTime() + cumulativeMs);
      const endTime = new Date(startTime.getTime() + phase.ms);
      cumulativeMs += phase.ms;

      return {
        id: `${parentBarId}-http-${phase.key}`,
        name: phase.label,
        startTime,
        endTime,
        style: phase.style,
      };
    });
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
  const hasHTTPTiming = !!bar.httpTimingBreakdown;
  const hasChildren = bar.children && bar.children.length > 0;
  const isExpandable = !bar.isRoot && (hasTimingBreakdown || hasChildren);
  const isExpanded = bar.isRoot ? true : expandedBars.has(bar.id);

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
      status={bar.status}
      viewStartOffset={viewStartOffset}
      viewEndOffset={viewEndOffset}
      startTime={bar.startTime}
      endTime={bar.endTime}
      minTime={minTime}
      delayMs={bar.delayMs}
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

          // Timing bars are expandable when there are children or HTTP timing data to show
          const isServerBar = timingBar.style === 'timing.server';
          const isTimingBarExpandable = isServerBar && (hasChildren || hasHTTPTiming);
          const isTimingBarExpanded = isTimingBarExpandable && expandedBars.has(timingBar.id);

          // Generate HTTP timing bars for the SERVER bar
          const httpTimingBars =
            isServerBar && hasHTTPTiming
              ? generateHTTPTimingBars(bar.id, bar.httpTimingBreakdown!, timingBar.startTime)
              : [];

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
              status={bar.status}
              expandable={isTimingBarExpandable}
              expanded={isTimingBarExpanded}
              onToggle={isTimingBarExpandable ? () => onToggleExpand(timingBar.id) : undefined}
              viewStartOffset={viewStartOffset}
              viewEndOffset={viewEndOffset}
              startTime={timingBar.startTime}
              endTime={timingBar.endTime}
              minTime={minTime}
            >
              {/* HTTP timing bars nested under YOUR SERVER */}
              {isTimingBarExpanded &&
                isServerBar &&
                httpTimingBars.map((httpBar) => {
                  const httpPosition = calculateBarPosition(
                    httpBar.startTime,
                    httpBar.endTime,
                    minTime,
                    maxTime
                  );
                  const httpDuration = calculateDuration(httpBar.startTime, httpBar.endTime);

                  return (
                    <TimelineBar
                      key={httpBar.id}
                      name={httpBar.name}
                      duration={httpDuration}
                      startPercent={httpPosition.startPercent}
                      widthPercent={httpPosition.widthPercent}
                      depth={depth + 2}
                      leftWidth={leftWidth}
                      style={httpBar.style}
                      status={bar.status}
                      viewStartOffset={viewStartOffset}
                      viewEndOffset={viewEndOffset}
                      startTime={httpBar.startTime}
                      endTime={httpBar.endTime}
                      minTime={minTime}
                    />
                  );
                })}

              {/* Child bars nested under YOUR SERVER */}
              {isTimingBarExpanded &&
                isServerBar &&
                hasChildren &&
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

  // Get status from the first (root) bar for header coloring
  const rootStatus = bars.find((bar) => bar.isRoot)?.status ?? bars[0]?.status;

  return (
    <div className="w-full pb-4 pr-2" data-testid="timeline-container">
      {/* Run duration header with timing markers */}
      <TimelineHeader
        minTime={minTime}
        maxTime={maxTime}
        leftWidth={leftWidth}
        onSelectionChange={handleSelectionChange}
        status={rootStatus}
        selectionStart={viewStartOffset}
        selectionEnd={viewEndOffset}
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
