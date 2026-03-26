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
  RunInngestBreakdownData,
  TimelineBarData,
  TimelineData,
  TimingDetail,
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
// Phase Definitions (shared between segment generators and sub-bar renderers)
// ============================================================================

/** A phase definition shared by segment generators and the sub-bar renderer. */
type PhaseDefinition<T> = {
  key: string;
  label: string;
  style: BarStyleKey;
  /** Extract the millisecond value for this phase from the breakdown data. */
  getMs: (data: T) => number;
};

const HTTP_PHASES: PhaseDefinition<HTTPTimingBreakdownData>[] = [
  { key: 'dns', label: 'DNS', style: 'timing.http.dns', getMs: (d) => d.dnsLookupMs },
  { key: 'tcp', label: 'TCP', style: 'timing.http.tcp', getMs: (d) => d.tcpConnectionMs },
  { key: 'tls', label: 'TLS', style: 'timing.http.tls', getMs: (d) => d.tlsHandshakeMs },
  {
    key: 'server',
    label: 'Server processing',
    style: 'timing.http.server',
    getMs: (d) => d.serverProcessingMs,
  },
  {
    key: 'transfer',
    label: 'Transfer',
    style: 'timing.http.transfer',
    getMs: (d) => d.contentTransferMs,
  },
];

const INNGEST_PHASES: PhaseDefinition<InngestBreakdownData>[] = [
  {
    key: 'discovery',
    label: 'Discovery',
    style: 'timing.inngest.discovery',
    getMs: (d) => d.discoveryMs,
  },
  {
    key: 'queue-delay',
    label: 'Concurrency delay',
    style: 'timing.inngest.concurrency',
    getMs: (d) => d.queueDelayMs,
  },
  {
    key: 'system-latency',
    label: 'System latency',
    style: 'timing.inngest.finalization',
    getMs: (d) => d.systemLatencyMs,
  },
];

const RUN_INNGEST_PHASES: PhaseDefinition<RunInngestBreakdownData>[] = [
  {
    key: 'run-queue',
    label: 'Run queue delay',
    style: 'timing.inngest.queue',
    getMs: (d) => d.runQueueDelayMs,
  },
  {
    key: 'finalization',
    label: 'Finalization',
    style: 'timing.inngest.finalization',
    getMs: (d) => d.finalizationMs,
  },
];

// ============================================================================
// Reusable Phase Sub-Bar Renderer
// ============================================================================

/** Renders a set of phase sub-bars within a parent bar's range. */
function PhaseSubBars<T extends { totalMs: number }>({
  phases,
  data,
  parentPosition,
  barIdPrefix,
  depth,
  leftWidth,
  status,
  onClick,
  viewStartOffset,
  viewEndOffset,
  startTime,
  endTime,
  minTime,
}: {
  phases: PhaseDefinition<T>[];
  data: T;
  parentPosition: { startPercent: number; widthPercent: number };
  barIdPrefix: string;
  depth: number;
  leftWidth: number;
  status?: string;
  onClick?: () => void;
  viewStartOffset?: number;
  viewEndOffset?: number;
  startTime?: Date;
  endTime?: Date | null;
  minTime: Date;
}) {
  if (data.totalMs <= 0) return null;

  let cumulativePercent = 0;
  return phases
    .filter((p) => p.getMs(data) > 0)
    .map((phase) => {
      const ms = phase.getMs(data);
      const phaseWidthPercent = (ms / data.totalMs) * parentPosition.widthPercent;
      const phaseStartPercent = parentPosition.startPercent + cumulativePercent;
      cumulativePercent += phaseWidthPercent;

      return (
        <TimelineBar
          key={`${barIdPrefix}-${phase.key}`}
          name={phase.label}
          duration={ms}
          startPercent={phaseStartPercent}
          widthPercent={phaseWidthPercent}
          depth={depth}
          leftWidth={leftWidth}
          style={phase.style}
          styleLabel={STYLE_LABELS[phase.style]}
          status={status}
          onClick={onClick}
          viewStartOffset={viewStartOffset}
          viewEndOffset={viewEndOffset}
          startTime={startTime}
          endTime={endTime ?? undefined}
          minTime={minTime}
        />
      );
    });
}

// ============================================================================
// Timing Breakdown Utilities
// ============================================================================

/**
 * Generate segments from phase definitions and breakdown data.
 * Used by all three breakdown segment generators.
 */
function generatePhaseSegments<T extends { totalMs: number }>(
  barId: string,
  segmentPrefix: string,
  phases: PhaseDefinition<T>[],
  data: T,
  status?: string
): BarSegment[] | undefined {
  const totalMs = data.totalMs;
  if (totalMs <= 0) return undefined;

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  for (const phase of phases) {
    const ms = phase.getMs(data);
    if (ms > 0) {
      const widthPercent = (ms / totalMs) * 100;
      segments.push({
        id: `${barId}-seg-${segmentPrefix}-${phase.key}`,
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
 * Generate segments for a compound bar based on timing breakdown.
 * Uses gray delay bar for the queue portion (matching V3's visual distinction).
 */
function generateBarSegments(bar: TimelineBarData): BarSegment[] | undefined {
  if (!bar.timingBreakdown) return undefined;

  const { inngestMs, executionMs, totalMs } = bar.timingBreakdown;

  if (totalMs <= 0) return undefined;

  const segments: BarSegment[] = [];
  let currentPercent = 0;

  // Inngest overhead segment — short gray delay bar
  if (inngestMs > 0) {
    const inngestPercent = (inngestMs / totalMs) * 100;
    segments.push({
      id: `${bar.id}-seg-delay`,
      startPercent: currentPercent,
      widthPercent: inngestPercent,
      style: 'timing.inngest',
    });
    currentPercent += inngestPercent;
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

/** Generate HTTP timing segments for the "Your server" compound bar. */
function generateHTTPSegments(
  barId: string,
  httpTiming: HTTPTimingBreakdownData,
  status?: string
): BarSegment[] | undefined {
  return generatePhaseSegments(barId, 'http', HTTP_PHASES, httpTiming, status);
}

/** Generate Inngest overhead segments for the Inngest compound bar. */
function generateInngestSegments(
  barId: string,
  breakdown: InngestBreakdownData
): BarSegment[] | undefined {
  return generatePhaseSegments(barId, 'inngest', INNGEST_PHASES, breakdown);
}

/** Generate segments for a run-level Inngest bar (run queue delay + finalization). */
function generateRunInngestSegments(
  barId: string,
  breakdown: RunInngestBreakdownData
): BarSegment[] | undefined {
  return generatePhaseSegments(barId, 'run-inngest', RUN_INNGEST_PHASES, breakdown);
}

// ============================================================================
// Hover Tooltip Timing Details
// ============================================================================

/** Human-readable labels for bar style keys shown in the hover tooltip. */
const STYLE_LABELS: Partial<Record<BarStyleKey, string>> = {
  'step.run': 'step.run',
  'step.sleep': 'step.sleep',
  'step.waitForEvent': 'step.waitForEvent',
  'step.invoke': 'step.invoke',
  'timing.inngest': 'Inngest overhead',
  'timing.inngest.queue': 'Run queue delay',
  'timing.inngest.concurrency': 'Concurrency delay',
  'timing.inngest.discovery': 'Discovery',
  'timing.inngest.finalization': 'Finalization',
  'timing.server': 'Your server',
  'timing.http.dns': 'DNS lookup',
  'timing.http.tcp': 'TCP connection',
  'timing.http.tls': 'TLS handshake',
  'timing.http.server': 'Server processing',
  'timing.http.transfer': 'Content transfer',
};

/** Derive tooltip rows from a phase definition array, filtering out zero values. */
function detailsFromPhases<T>(phases: PhaseDefinition<T>[], data: T): TimingDetail[] {
  return phases
    .map((p) => ({ label: p.label, durationMs: p.getMs(data) }))
    .filter((d) => d.durationMs > 0);
}

/**
 * Build timing detail rows for a bar's hover tooltip based on available data.
 */
function buildTimingDetails(bar: TimelineBarData): TimingDetail[] | undefined {
  const details: TimingDetail[] = [];

  // Inngest overhead breakdown (per-step)
  if (bar.inngestBreakdown) {
    details.push(...detailsFromPhases(INNGEST_PHASES, bar.inngestBreakdown));
  }

  // Run-level Inngest overhead (root bar)
  if (bar.runInngestBreakdown) {
    details.push(...detailsFromPhases(RUN_INNGEST_PHASES, bar.runInngestBreakdown));
  }

  // Timing breakdown (queue + execution) — no matching phase array
  if (bar.timingBreakdown) {
    const b = bar.timingBreakdown;
    if (b.inngestMs > 0) details.push({ label: 'Inngest', durationMs: b.inngestMs });
    if (b.executionMs > 0) details.push({ label: 'Your server', durationMs: b.executionMs });
  }

  // HTTP timing breakdown
  if (bar.httpTimingBreakdown) {
    details.push(...detailsFromPhases(HTTP_PHASES, bar.httpTimingBreakdown));
  }

  return details.length > 0 ? details : undefined;
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
  const hasRunInngestBreakdown = !!bar.runInngestBreakdown;
  const hasInngestBreakdown = !!bar.inngestBreakdown;
  const isExpandable =
    hasTimingBreakdown || hasInngestBreakdown || hasChildren || hasRunInngestBreakdown;
  const isExpanded = bar.isRoot ? true : expandedBars.has(bar.id);

  // Generate segments for compound bar visualization
  // Bars with timingBreakdown use queue+execution segments; others fall back to delay+execution
  const segments = generateBarSegments(bar) ?? generateDelaySegments(bar);

  // Pre-compute timing sub-bar positions from the parent bar's position.
  // This ensures sub-bars visually align with the parent's compound segments.
  //
  // For the Inngest portion: prefer timingBreakdown.inngestMs (from metadata),
  // but fall back to inngestBreakdown.totalMs (from timestamps) so we still
  // show the Inngest bar even when metadata is missing or reports 0.
  const timingPositions = (() => {
    const breakdownInngestMs = bar.timingBreakdown?.inngestMs ?? 0;
    const executionMs = bar.timingBreakdown?.executionMs ?? 0;
    const inngestMs =
      breakdownInngestMs > 0 ? breakdownInngestMs : bar.inngestBreakdown?.totalMs ?? 0;
    const totalMs = inngestMs + executionMs;
    if (totalMs <= 0) return null;

    const inngestWidth = (inngestMs / totalMs) * widthPercent;
    const serverWidth = (executionMs / totalMs) * widthPercent;
    const serverStart = startPercent + inngestWidth;

    return {
      inngest:
        inngestMs > 0
          ? { startPercent: startPercent, widthPercent: inngestWidth, duration: inngestMs }
          : null,
      server:
        executionMs > 0
          ? { startPercent: serverStart, widthPercent: serverWidth, duration: executionMs }
          : null,
    };
  })();

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
  const isInngestExpandable = hasInngestBreakdown;
  const isInngestExpanded = isInngestExpandable && expandedBars.has(inngestBarId);

  // Inngest breakdown segments for the Inngest bar's compound visualization
  const inngestBarSegments = hasInngestBreakdown
    ? generateInngestSegments(inngestBarId, bar.inngestBreakdown!)
    : undefined;

  // Run-level Inngest bar (run queue delay + finalization) — only for root bars
  const runInngestBarId = `${bar.id}-run-inngest`;
  const isRunInngestExpandable = hasRunInngestBreakdown;
  const isRunInngestExpanded = isRunInngestExpandable && expandedBars.has(runInngestBarId);
  const runInngestBarSegments = hasRunInngestBreakdown
    ? generateRunInngestSegments(runInngestBarId, bar.runInngestBreakdown!)
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
      styleLabel={STYLE_LABELS[bar.style]}
      segments={segments}
      // Root bar is always expanded (children always visible) but not expandable
      // (no toggle UI). expandable=false ensures VisualBar keeps opacity 1.
      expandable={bar.isRoot ? false : isExpandable}
      expanded={isExpanded}
      onToggle={bar.isRoot ? undefined : () => onToggleExpand(bar.id)}
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
      timingDetails={buildTimingDetails(bar)}
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
          styleLabel={STYLE_LABELS['timing.inngest']}
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
          {isInngestExpanded && hasInngestBreakdown && (
            <PhaseSubBars
              phases={INNGEST_PHASES}
              data={bar.inngestBreakdown!}
              parentPosition={timingPositions.inngest!}
              barIdPrefix={`${bar.id}-inngest`}
              depth={depth + 2}
              leftWidth={leftWidth}
              status={bar.status}
              onClick={() => onSelectStep?.(bar.id)}
              viewStartOffset={viewStartOffset}
              viewEndOffset={viewEndOffset}
              startTime={bar.startTime}
              endTime={bar.endTime}
              minTime={minTime}
            />
          )}
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
          styleLabel={STYLE_LABELS['timing.server']}
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
          {isServerExpanded && hasHTTPTiming && (
            <PhaseSubBars
              phases={HTTP_PHASES}
              data={bar.httpTimingBreakdown!}
              parentPosition={timingPositions.server!}
              barIdPrefix={`${bar.id}-http`}
              depth={depth + 2}
              leftWidth={leftWidth}
              status={bar.status}
              onClick={() => onSelectStep?.(bar.id)}
              viewStartOffset={viewStartOffset}
              viewEndOffset={viewEndOffset}
              startTime={bar.startTime}
              endTime={bar.endTime}
              minTime={minTime}
            />
          )}

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

      {/* Run-level Inngest bar — shows run queue delay + finalization for root bars */}
      {isExpanded && bar.isRoot && hasRunInngestBreakdown && timingPositions?.inngest && (
        <TimelineBar
          name="Inngest"
          duration={bar.runInngestBreakdown!.totalMs}
          startPercent={timingPositions.inngest.startPercent}
          widthPercent={timingPositions.inngest.widthPercent}
          depth={depth + 1}
          leftWidth={leftWidth}
          style="timing.inngest"
          styleLabel={STYLE_LABELS['timing.inngest']}
          segments={runInngestBarSegments}
          status={bar.status}
          expandable={isRunInngestExpandable}
          expanded={isRunInngestExpanded}
          onToggle={isRunInngestExpandable ? () => onToggleExpand(runInngestBarId) : undefined}
          onClick={() => onSelectStep?.(bar.id)}
          viewStartOffset={viewStartOffset}
          viewEndOffset={viewEndOffset}
          startTime={bar.startTime}
          endTime={bar.endTime}
          minTime={minTime}
        >
          {/* Run-level Inngest sub-bars */}
          {isRunInngestExpanded && (
            <PhaseSubBars
              phases={RUN_INNGEST_PHASES}
              data={bar.runInngestBreakdown!}
              parentPosition={timingPositions.inngest!}
              barIdPrefix={`${bar.id}-run-inngest`}
              depth={depth + 2}
              leftWidth={leftWidth}
              status={bar.status}
              onClick={() => onSelectStep?.(bar.id)}
              viewStartOffset={viewStartOffset}
              viewEndOffset={viewEndOffset}
              startTime={bar.startTime}
              endTime={bar.endTime}
              minTime={minTime}
            />
          )}
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
