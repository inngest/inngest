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

import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { RiContractUpDownLine, RiExpandUpDownLine } from '@remixicon/react';

import { Button } from '../Button';
import { TimelineBar } from './TimelineBar';
import type {
  BarSegment,
  BarStyleKey,
  HTTPTimingBreakdownData,
  InngestBreakdownData,
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
 * Generate segments for a compound bar based on timing breakdown.
 * Uses gray delay bar for the queue portion (matching V3's visual distinction).
 */
function generateBarSegments(bar: TimelineBarData): BarSegment[] | undefined {
  if (!bar.timingBreakdown) return undefined;

  const { queueMs, executionMs, totalMs } = bar.timingBreakdown;

  if (totalMs <= 0) return undefined;

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  // Queue segment — short gray delay bar
  if (queueMs > 0) {
    const queuePercent = (queueMs / totalMs) * 100;
    segments.push({
      id: `${bar.id}-seg-delay`,
      startPercent: currentPercent,
      widthPercent: queuePercent,
      style: 'timing.inngest',
    });
    currentPercent += queuePercent;
  }

  // Execution segment — root bar uses short status-colored bar, steps use tall barber-pole
  if (executionMs > 0) {
    const execPercent = (executionMs / totalMs) * 100;
    segments.push({
      id: `${bar.id}-seg-server`,
      startPercent: currentPercent,
      widthPercent: execPercent,
      style: bar.isRoot ? 'root' : 'timing.server',
      status: bar.status,
    });
  }

  return segments.length > 0 ? segments : undefined;
}

/**
 * Generate delay + execution segments for any bar with delay data.
 * Fallback for bars without timingBreakdown (e.g. root bar, Finalization span).
 * Shows checkpoint/queue delay as a gray bar followed by the execution portion.
 */
function generateDelaySegments(bar: TimelineBarData): BarSegment[] | undefined {
  if (bar.delayMs == null || bar.delayMs <= 0) return undefined;

  const totalMs = bar.endTime
    ? bar.endTime.getTime() - bar.startTime.getTime()
    : Date.now() - bar.startTime.getTime();

  if (totalMs <= 0) return undefined;

  const delayPercent = Math.min((bar.delayMs / totalMs) * 100, 100);
  const execPercent = Math.max(100 - delayPercent, 0);

  const segments: BarSegment[] = [];

  if (delayPercent > 0) {
    segments.push({
      id: `${bar.id}-seg-delay`,
      startPercent: 0,
      widthPercent: delayPercent,
      style: 'timing.inngest',
    });
  }

  if (execPercent > 0) {
    segments.push({
      id: `${bar.id}-seg-exec`,
      startPercent: delayPercent,
      widthPercent: execPercent,
      style: bar.style,
      status: bar.status,
    });
  }

  return segments.length > 0 ? segments : undefined;
}

/**
 * Generate HTTP timing segments for the "Your server" compound bar.
 * Shows each HTTP phase (DNS, TCP, TLS, TTFB, Transfer) as a colored segment
 * within the bar, proportional to its duration.
 */
function generateHTTPSegments(
  barId: string,
  httpTiming: HTTPTimingBreakdownData,
  status?: string
): BarSegment[] | undefined {
  const totalMs = httpTiming.totalMs;
  if (totalMs <= 0) return undefined;

  const phases: { key: string; ms: number; style: BarStyleKey }[] = [
    { key: 'dns', ms: httpTiming.dnsLookupMs, style: 'timing.http.dns' },
    { key: 'tcp', ms: httpTiming.tcpConnectionMs, style: 'timing.http.tcp' },
    { key: 'tls', ms: httpTiming.tlsHandshakeMs, style: 'timing.http.tls' },
    { key: 'server', ms: httpTiming.serverProcessingMs, style: 'timing.http.server' },
    { key: 'transfer', ms: httpTiming.contentTransferMs, style: 'timing.http.transfer' },
  ];

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  for (const phase of phases) {
    if (phase.ms > 0) {
      const widthPercent = (phase.ms / totalMs) * 100;
      segments.push({
        id: `${barId}-seg-http-${phase.key}`,
        startPercent: currentPercent,
        widthPercent,
        style: phase.style,
        status,
      });
      currentPercent += widthPercent;
    }
  }

  return segments.length > 0 ? segments : undefined;
}

/**
 * Generate Inngest overhead segments for the Inngest compound bar.
 * Shows queue delay, discovery/scheduling, and finalization as colored segments.
 */
function generateInngestSegments(
  barId: string,
  breakdown: InngestBreakdownData
): BarSegment[] | undefined {
  const totalMs = breakdown.totalMs;
  if (totalMs <= 0) return undefined;

  const phases: { key: string; ms: number; style: BarStyleKey }[] = [
    { key: 'queue', ms: breakdown.runQueueDelayMs, style: 'timing.inngest.queue' },
    { key: 'discovery', ms: breakdown.discoveryMs, style: 'timing.inngest.discovery' },
    { key: 'finalization', ms: breakdown.finalizationMs, style: 'timing.inngest.finalization' },
  ];

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  for (const phase of phases) {
    if (phase.ms > 0) {
      const widthPercent = (phase.ms / totalMs) * 100;
      segments.push({
        id: `${barId}-seg-inngest-${phase.key}`,
        startPercent: currentPercent,
        widthPercent,
        style: phase.style,
      });
      currentPercent += widthPercent;
    }
  }

  return segments.length > 0 ? segments : undefined;
}

// ============================================================================
// Expand All / Collapse All Utilities
// ============================================================================

/**
 * Recursively collect all expandable bar IDs from the timeline data.
 * This includes:
 * - Step bars with timingBreakdown or children (non-root)
 * - Server timing bars that can expand to show HTTP timing or children
 */
function collectExpandableIds(bars: TimelineBarData[]): string[] {
  const ids: string[] = [];
  for (const bar of bars) {
    const hasTimingBreakdown = !!bar.timingBreakdown;
    const hasChildren = bar.children && bar.children.length > 0;
    const hasHTTPTiming = !!bar.httpTimingBreakdown;

    // Step-level bars are expandable when they have timing or children
    if (!bar.isRoot && (hasTimingBreakdown || hasChildren)) {
      ids.push(bar.id);

      // The server timing bar is expandable when there are children or HTTP timing
      if (hasTimingBreakdown && (hasHTTPTiming || hasChildren)) {
        ids.push(`${bar.id}-timing-server`);
      }
    }

    // Recurse into children
    if (bar.children) {
      ids.push(...collectExpandableIds(bar.children));
    }
  }
  return ids;
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
  /** Optional actions to render in the bar's left panel (e.g. expand/collapse all) */
  actions?: ReactNode;
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
  actions,
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
  // Bars with timingBreakdown use queue+execution segments; others fall back to delay+execution
  const segments = generateBarSegments(bar) ?? generateDelaySegments(bar);

  // Pre-compute timing sub-bar positions from the parent bar's position.
  // This ensures sub-bars visually align with the parent's compound segments.
  const timingPositions = hasTimingBreakdown
    ? (() => {
        const { queueMs, executionMs, totalMs } = bar.timingBreakdown!;
        if (totalMs <= 0) return null;

        const inngestWidth = (queueMs / totalMs) * widthPercent;
        const serverWidth = (executionMs / totalMs) * widthPercent;
        const serverStart = startPercent + inngestWidth;

        return {
          inngest:
            queueMs > 0
              ? { startPercent: startPercent, widthPercent: inngestWidth, duration: queueMs }
              : null,
          server:
            executionMs > 0
              ? { startPercent: serverStart, widthPercent: serverWidth, duration: executionMs }
              : null,
        };
      })()
    : null;

  // Server timing bar IDs for expansion tracking
  const serverBarId = `${bar.id}-timing-server`;
  const isServerExpandable = hasChildren || hasHTTPTiming;
  const isServerExpanded = isServerExpandable && expandedBars.has(serverBarId);

  // HTTP timing segments for the server bar's compound visualization
  const serverBarSegments = hasHTTPTiming
    ? generateHTTPSegments(serverBarId, bar.httpTimingBreakdown!, bar.status)
    : undefined;

  // Inngest timing bar IDs for expansion tracking
  const inngestBarId = `${bar.id}-timing-inngest`;
  const hasInngestBreakdown = !!bar.inngestBreakdown;
  const isInngestExpandable = hasInngestBreakdown;
  const isInngestExpanded = isInngestExpandable && expandedBars.has(inngestBarId);

  // Inngest breakdown segments for the Inngest bar's compound visualization
  const inngestBarSegments = hasInngestBreakdown
    ? generateInngestSegments(inngestBarId, bar.inngestBreakdown!)
    : undefined;

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
      actions={actions}
    >
      {/* Inngest timing bar — positioned to match the queue segment of the parent.
          Only for non-root bars; the root uses timingBreakdown only for compound segments. */}
      {isExpanded && !bar.isRoot && timingPositions?.inngest && (
        <TimelineBar
          name="Inngest"
          duration={timingPositions.inngest.duration}
          startPercent={timingPositions.inngest.startPercent}
          widthPercent={timingPositions.inngest.widthPercent}
          depth={depth + 1}
          leftWidth={leftWidth}
          style="timing.inngest"
          segments={inngestBarSegments}
          orgName={orgName}
          status={bar.status}
          expandable={isInngestExpandable}
          expanded={isInngestExpanded}
          onToggle={isInngestExpandable ? () => onToggleExpand(inngestBarId) : undefined}
          onClick={() => onSelectStep?.(bar.id)}
          viewStartOffset={viewStartOffset}
          viewEndOffset={viewEndOffset}
          startTime={bar.startTime}
          endTime={bar.endTime}
          minTime={minTime}
        >
          {/* Inngest breakdown sub-bars — each positioned within the Inngest bar's range */}
          {isInngestExpanded &&
            hasInngestBreakdown &&
            (() => {
              const breakdown = bar.inngestBreakdown!;
              const inngestTotalMs = breakdown.totalMs;
              if (inngestTotalMs <= 0) return null;

              const iPos = timingPositions.inngest!;
              const phases = [
                {
                  key: 'queue',
                  label: 'Queue delay',
                  ms: breakdown.runQueueDelayMs,
                  style: 'timing.inngest.queue' as BarStyleKey,
                },
                {
                  key: 'discovery',
                  label: 'Discovery',
                  ms: breakdown.discoveryMs,
                  style: 'timing.inngest.discovery' as BarStyleKey,
                },
                {
                  key: 'finalization',
                  label: 'Finalization',
                  ms: breakdown.finalizationMs,
                  style: 'timing.inngest.finalization' as BarStyleKey,
                },
              ];

              let cumulativePercent = 0;
              return phases
                .filter((p) => p.ms > 0)
                .map((phase) => {
                  const phaseWidthPercent = (phase.ms / inngestTotalMs) * iPos.widthPercent;
                  const phaseStartPercent = iPos.startPercent + cumulativePercent;
                  cumulativePercent += phaseWidthPercent;

                  return (
                    <TimelineBar
                      key={`${bar.id}-inngest-${phase.key}`}
                      name={phase.label}
                      duration={phase.ms}
                      startPercent={phaseStartPercent}
                      widthPercent={phaseWidthPercent}
                      depth={depth + 2}
                      leftWidth={leftWidth}
                      style={phase.style}
                      status={bar.status}
                      onClick={() => onSelectStep?.(bar.id)}
                      viewStartOffset={viewStartOffset}
                      viewEndOffset={viewEndOffset}
                      startTime={bar.startTime}
                      endTime={bar.endTime}
                      minTime={minTime}
                    />
                  );
                });
            })()}
        </TimelineBar>
      )}

      {/* Your server timing bar — positioned to match the execution segment of the parent.
          Only for non-root bars; the root renders step children directly. */}
      {isExpanded && !bar.isRoot && timingPositions?.server && (
        <TimelineBar
          name={orgName ?? 'Your server'}
          duration={timingPositions.server.duration}
          startPercent={timingPositions.server.startPercent}
          widthPercent={timingPositions.server.widthPercent}
          depth={depth + 1}
          leftWidth={leftWidth}
          style="timing.server"
          segments={serverBarSegments}
          orgName={orgName}
          status={bar.status}
          expandable={isServerExpandable}
          expanded={isServerExpanded}
          onToggle={isServerExpandable ? () => onToggleExpand(serverBarId) : undefined}
          onClick={() => onSelectStep?.(bar.id)}
          viewStartOffset={viewStartOffset}
          viewEndOffset={viewEndOffset}
          startTime={bar.startTime}
          endTime={bar.endTime}
          minTime={minTime}
        >
          {/* HTTP timing bars — each positioned within the server bar's range */}
          {isServerExpanded &&
            hasHTTPTiming &&
            (() => {
              const httpTiming = bar.httpTimingBreakdown!;
              const httpTotalMs = httpTiming.totalMs;
              if (httpTotalMs <= 0) return null;

              const sPos = timingPositions.server!;
              const phases = [
                {
                  key: 'dns',
                  label: 'DNS',
                  ms: httpTiming.dnsLookupMs,
                  style: 'timing.http.dns' as BarStyleKey,
                },
                {
                  key: 'tcp',
                  label: 'TCP',
                  ms: httpTiming.tcpConnectionMs,
                  style: 'timing.http.tcp' as BarStyleKey,
                },
                {
                  key: 'tls',
                  label: 'TLS',
                  ms: httpTiming.tlsHandshakeMs,
                  style: 'timing.http.tls' as BarStyleKey,
                },
                {
                  key: 'server',
                  label: 'TTFB',
                  ms: httpTiming.serverProcessingMs,
                  style: 'timing.http.server' as BarStyleKey,
                },
                {
                  key: 'transfer',
                  label: 'Transfer',
                  ms: httpTiming.contentTransferMs,
                  style: 'timing.http.transfer' as BarStyleKey,
                },
              ];

              let cumulativePercent = 0;
              return phases
                .filter((p) => p.ms > 0)
                .map((phase) => {
                  const phaseWidthPercent = (phase.ms / httpTotalMs) * sPos.widthPercent;
                  const phaseStartPercent = sPos.startPercent + cumulativePercent;
                  cumulativePercent += phaseWidthPercent;

                  return (
                    <TimelineBar
                      key={`${bar.id}-http-${phase.key}`}
                      name={phase.label}
                      duration={phase.ms}
                      startPercent={phaseStartPercent}
                      widthPercent={phaseWidthPercent}
                      depth={depth + 2}
                      leftWidth={leftWidth}
                      style={phase.style}
                      status={bar.status}
                      onClick={() => onSelectStep?.(bar.id)}
                      viewStartOffset={viewStartOffset}
                      viewEndOffset={viewEndOffset}
                      startTime={bar.startTime}
                      endTime={bar.endTime}
                      minTime={minTime}
                    />
                  );
                });
            })()}

          {/* Child bars nested under YOUR SERVER */}
          {isServerExpanded &&
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
      )}

      {/* Child bars (for root bars, or non-root bars without timing breakdown) */}
      {isExpanded &&
        (bar.isRoot || !hasTimingBreakdown) &&
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

  const rootBarIds = useMemo(() => bars.filter((bar) => bar.isRoot).map((bar) => bar.id), [bars]);

  // Initialize with root bars expanded by default
  const [expandedBars, setExpandedBars] = useState<Set<string>>(() => new Set(rootBarIds));
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

  const handleExpandAll = useCallback(() => {
    const allExpandableIds = collectExpandableIds(bars);
    setExpandedBars(new Set([...rootBarIds, ...allExpandableIds]));
  }, [bars, rootBarIds]);

  const handleCollapseAll = useCallback(() => {
    setExpandedBars(new Set(rootBarIds));
  }, [rootBarIds]);

  const expandCollapseActions = useMemo(
    () => (
      <span className="flex shrink-0 items-center gap-0.5" onClick={(e) => e.stopPropagation()}>
        <Button
          size="small"
          appearance="ghost"
          icon={<RiExpandUpDownLine className="h-3.5 w-3.5" />}
          title="Expand all"
          tooltip="Expand all"
          aria-label="Expand all"
          onClick={handleExpandAll}
        />
        <Button
          size="small"
          appearance="ghost"
          icon={<RiContractUpDownLine className="h-3.5 w-3.5" />}
          title="Collapse all"
          tooltip="Collapse all"
          aria-label="Collapse all"
          onClick={handleCollapseAll}
        />
      </span>
    ),
    [handleExpandAll, handleCollapseAll]
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
          actions={bar.isRoot ? expandCollapseActions : undefined}
        />
      ))}
    </div>
  );
}
