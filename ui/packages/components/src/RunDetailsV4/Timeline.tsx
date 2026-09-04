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

import { useCallback, useMemo, useState, type JSX, type ReactNode } from 'react';
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
import { applyCollapseToBars, type CollapsePlan } from './canvas/collapse';
import { formatDuration, useStepSelection } from './runDetailsUtils';
import { densityBuckets } from './utils/density';
import { buildTimeScale, type Interval, type TimeScale } from './utils/timeScale';
import { calculateBarPosition, calculateDuration } from './utils/timing';

// ============================================================================
// Types
// ============================================================================

type Props = {
  /** Timeline data to render */
  data: TimelineData;
  /** Callback when a step is selected */
  onSelectStep?: (stepId: string) => void;
  /**
   * Scopes the shared step selection so the highlighted row follows selections
   * made elsewhere in the same run (e.g. the canvas), not just clicks here.
   */
  runID?: string;
  /**
   * Collapses repeated steps into one row each. Omit to draw every row — the
   * canvas and the timeline take the same plan, so the two stay in step.
   */
  collapse?: CollapsePlan;
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

/**
 * Bars that represent the run being suspended rather than working.
 *
 * These are the stretches the elastic axis is allowed to elide: they cost
 * nothing, and a nine-day sleep drawn to scale leaves no room for the five
 * seconds that actually ran. `step.invoke` is deliberately absent — a child run
 * is genuinely executing during one.
 */
const IDLE_STYLES = new Set<BarStyleKey>(['step.sleep', 'step.waitForEvent']);

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
  minTime: Date;
}) {
  if (data.totalMs <= 0) return null;

  let cumulativeMs = 0;
  let cumulativePercent = 0;
  return phases
    .filter((p) => p.getMs(data) > 0)
    .map((phase) => {
      const ms = phase.getMs(data);
      const phaseWidthPercent = (ms / data.totalMs) * parentPosition.widthPercent;
      const phaseStartPercent = parentPosition.startPercent + cumulativePercent;

      // Compute phase-specific start/end times so the hover card shows
      // the phase's own time window, not the parent bar's.
      const phaseStartTime = startTime ? new Date(startTime.getTime() + cumulativeMs) : undefined;
      const phaseEndTime = phaseStartTime ? new Date(phaseStartTime.getTime() + ms) : undefined;

      cumulativeMs += ms;
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
          startTime={phaseStartTime}
          endTime={phaseEndTime}
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
      style: 'timing.waiting',
      status: bar.status,
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
  breakdown: RunInngestBreakdownData,
  runDurationMs: number
): BarSegment[] | undefined {
  if (runDurationMs <= 0) return undefined;

  const segments: BarSegment[] = [];

  // Run queue delay: starts at 0% (beginning of run)
  if (breakdown.runQueueDelayMs > 0) {
    segments.push({
      id: `${barId}-seg-run-inngest-run-queue`,
      startPercent: 0,
      widthPercent: (breakdown.runQueueDelayMs / runDurationMs) * 100,
      style: 'timing.inngest.queue',
    });
  }

  // Finalization: positioned at end of run (lastStepEndedAt → runEndedAt)
  if (breakdown.finalizationMs > 0) {
    segments.push({
      id: `${barId}-seg-run-inngest-finalization`,
      startPercent: ((runDurationMs - breakdown.finalizationMs) / runDurationMs) * 100,
      widthPercent: (breakdown.finalizationMs / runDurationMs) * 100,
      style: 'timing.inngest.finalization',
    });
  }

  return segments.length > 0 ? segments : undefined;
}

// ============================================================================
// Hover Tooltip Timing Details
// ============================================================================

/** Human-readable labels for bar style keys shown in the hover tooltip. */
const STYLE_LABELS: Partial<Record<BarStyleKey, string>> = {
  'timing.waiting': 'Waiting to run',
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
  /** Whether this bar is inside an experiment (inherited from parent) */
  insideExperiment?: boolean;
  /**
   * The elastic axis, when the run has idle stretches worth breaking. Affects
   * only where a bar is drawn — never the duration it reports.
   */
  scale?: TimeScale;
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
  insideExperiment,
  scale,
}: TimelineBarRendererProps): JSX.Element {
  const { startPercent, widthPercent } = calculateBarPosition(
    bar.startTime,
    bar.endTime,
    minTime,
    maxTime,
    scale
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

  // Children of experiment bars inherit the dotted background
  const childInsideExperiment = insideExperiment || bar.hasExperiment;

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
    ? generateRunInngestSegments(
        runInngestBarId,
        bar.runInngestBreakdown!,
        calculateDuration(bar.startTime, bar.endTime)
      )
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
      hasExperiment={bar.hasExperiment}
      insideExperiment={insideExperiment}
      experimentMetadata={bar.experimentMetadata}
      scores={bar.scores}
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
          insideExperiment={childInsideExperiment}
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
                insideExperiment={childInsideExperiment}
                scale={scale}
              />
            ))}
        </TimelineBar>
      )}

      {/* Run-level Inngest bar — shows run queue delay + finalization for root bars */}
      {isExpanded && bar.isRoot && hasRunInngestBreakdown && (
        <TimelineBar
          name="Inngest"
          duration={bar.runInngestBreakdown!.totalMs}
          startPercent={startPercent}
          widthPercent={widthPercent}
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
          {/* Run-level Inngest sub-bars — positioned at their absolute timeline locations */}
          {isRunInngestExpanded && (
            <>
              {/* Run queue delay: run.queuedAt → run.startedAt */}
              {bar.runInngestBreakdown!.runQueueDelayMs > 0 && (
                <TimelineBar
                  key={`${bar.id}-run-inngest-run-queue`}
                  name="Run queue delay"
                  duration={bar.runInngestBreakdown!.runQueueDelayMs}
                  startPercent={startPercent}
                  widthPercent={
                    (bar.runInngestBreakdown!.runQueueDelayMs /
                      Math.max(1, (bar.endTime?.getTime() ?? 0) - bar.startTime.getTime())) *
                    widthPercent
                  }
                  depth={depth + 2}
                  leftWidth={leftWidth}
                  style="timing.inngest.queue"
                  styleLabel={STYLE_LABELS['timing.inngest.queue']}
                  status={bar.status}
                  onClick={() => onSelectStep?.(bar.id)}
                  viewStartOffset={viewStartOffset}
                  viewEndOffset={viewEndOffset}
                  startTime={bar.startTime}
                  endTime={
                    new Date(bar.startTime.getTime() + bar.runInngestBreakdown!.runQueueDelayMs)
                  }
                  minTime={minTime}
                />
              )}
              {/* Finalization: lastStep.endedAt → run.endedAt */}
              {bar.runInngestBreakdown!.finalizationMs > 0 && bar.endTime && (
                <TimelineBar
                  key={`${bar.id}-run-inngest-finalization`}
                  name="Finalization"
                  duration={bar.runInngestBreakdown!.finalizationMs}
                  startPercent={
                    startPercent +
                    ((bar.endTime.getTime() -
                      bar.runInngestBreakdown!.finalizationMs -
                      bar.startTime.getTime()) /
                      Math.max(1, bar.endTime.getTime() - bar.startTime.getTime())) *
                      widthPercent
                  }
                  widthPercent={
                    (bar.runInngestBreakdown!.finalizationMs /
                      Math.max(1, bar.endTime.getTime() - bar.startTime.getTime())) *
                    widthPercent
                  }
                  depth={depth + 2}
                  leftWidth={leftWidth}
                  style="timing.inngest.finalization"
                  styleLabel={STYLE_LABELS['timing.inngest.finalization']}
                  status={bar.status}
                  onClick={() => onSelectStep?.(bar.id)}
                  viewStartOffset={viewStartOffset}
                  viewEndOffset={viewEndOffset}
                  startTime={
                    new Date(bar.endTime.getTime() - bar.runInngestBreakdown!.finalizationMs)
                  }
                  endTime={bar.endTime}
                  minTime={minTime}
                />
              )}
            </>
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
            insideExperiment={childInsideExperiment}
            scale={scale}
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
export function Timeline({ data, onSelectStep, runID, collapse }: Props): JSX.Element {
  const { minTime, maxTime, leftWidth, orgName } = data;

  // Repetition is detected once, in the model, and applied to both views. A
  // group becomes one row whose children are its members, so opening it uses
  // the expansion affordance that is already here rather than a second one.
  const bars = useMemo(
    () => (collapse ? applyCollapseToBars(data.bars, collapse) : data.bars),
    [data.bars, collapse]
  );

  const rootBarIds = useMemo(() => bars.filter((bar) => bar.isRoot).map((bar) => bar.id), [bars]);

  // Initialize with root bars expanded by default
  const [expandedBars, setExpandedBars] = useState<Set<string>>(() => new Set(rootBarIds));

  // Read the highlighted row from the shared selection rather than local state,
  // so selecting a node on the canvas highlights the matching row here. Bar ids
  // are the rolled-up span ids, which is exactly what the canvas emits.
  const { selectedStep } = useStepSelection({ runID });
  const selectedStepId = selectedStep?.trace.spanID;

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
      // The row highlight comes back through the shared selection, which
      // `onSelectStep` drives — no local mirror to keep in sync.
      onSelectStep?.(stepId);
    },
    [onSelectStep]
  );

  const expandableIds = useMemo(() => collectExpandableIds(bars), [bars]);

  const handleExpandAll = useCallback(() => {
    setExpandedBars(new Set([...rootBarIds, ...expandableIds]));
  }, [expandableIds, rootBarIds]);

  const handleCollapseAll = useCallback(() => {
    setExpandedBars(new Set(rootBarIds));
  }, [rootBarIds]);

  // Only offer each action when it would actually change something: expanding
  // requires a collapsed expandable bar, collapsing requires an expanded
  // non-root bar (roots are always expanded).
  const canExpandAll = expandableIds.some((id) => !expandedBars.has(id));
  const canCollapseAll = useMemo(() => {
    const roots = new Set(rootBarIds);
    return [...expandedBars].some((id) => !roots.has(id));
  }, [expandedBars, rootBarIds]);

  const expandCollapseActions = useMemo(
    () =>
      canExpandAll || canCollapseAll ? (
        <span className="flex shrink-0 items-center gap-0.5" onClick={(e) => e.stopPropagation()}>
          {canExpandAll && (
            <Button
              size="small"
              appearance="ghost"
              icon={<RiExpandUpDownLine className="h-3.5 w-3.5" />}
              title="Expand all"
              tooltip="Expand all"
              aria-label="Expand all"
              onClick={handleExpandAll}
            />
          )}
          {canCollapseAll && (
            <Button
              size="small"
              appearance="ghost"
              icon={<RiContractUpDownLine className="h-3.5 w-3.5" />}
              title="Collapse all"
              tooltip="Collapse all"
              aria-label="Collapse all"
              onClick={handleCollapseAll}
            />
          )}
        </span>
      ) : undefined,
    [canExpandAll, canCollapseAll, handleExpandAll, handleCollapseAll]
  );

  // Handle timeline brush selection change
  const handleSelectionChange = useCallback((start: number, end: number) => {
    setViewStartOffset(start);
    setViewEndOffset(end);
  }, []);

  // Get status from the first (root) bar for header coloring
  const rootStatus = bars.find((bar) => bar.isRoot)?.status ?? bars[0]?.status;

  // Break the axis wherever the run was idle. A realistic run is days elapsed
  // and seconds executing; drawn linearly every step is sub-pixel.
  //
  // Two rules decide what counts as "busy". Only leaf bars, because a parent
  // spans its children and would fill every gap they left — nothing would ever
  // look idle. And a sleep or a waitForEvent is NOT busy: it is precisely the
  // suspended stretch costing nothing, which is the thing worth eliding. An
  // invoke is left as busy, because a child run really is doing work in there.
  const scale = useMemo(() => {
    const busy: Interval[] = [];
    const collect = (list: TimelineBarData[]) => {
      for (const bar of list) {
        if (bar.children?.length) {
          collect(bar.children);
        } else if (!bar.isRoot && !IDLE_STYLES.has(bar.style)) {
          busy.push({
            startMs: bar.startTime.getTime(),
            endMs: (bar.endTime ?? bar.startTime).getTime(),
          });
        }
      }
    };
    collect(bars);
    return buildTimeScale(minTime.getTime(), maxTime.getTime(), busy);
  }, [bars, minTime, maxTime]);

  // Deliberately over `data.bars`, not the collapsed `bars`. The strip is the
  // map of the whole run and must not shrink because the rows below it did —
  // otherwise the one view that is supposed to show you everything hides the
  // same things as everything else.
  const buckets = useMemo(() => densityBuckets(data.bars, scale), [data.bars, scale]);

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
        scale={scale}
        buckets={buckets}
      />

      {/* The rows, with the axis breaks drawn behind them. A break spans every
          row because it is a property of the axis rather than of any one step —
          and it is drawn at all because an axis that is not linear has to say
          so. Elided time is a filled band bounded by dashed rules, with its real
          duration set inside it: visibly a break, not a gap. */}
      <div className="relative">
        {/* The one structural rule in the view: where the label column ends and
            the plot begins. Drawn once, spanning every row, rather than as a
            border on each — a rule per row separates nothing. */}
        <div
          className="bg-canvasMuted pointer-events-none absolute inset-y-0 z-0 w-px"
          style={{ left: `${leftWidth}%` }}
        />

        {scale.compressed && (
          <div
            className="pointer-events-none absolute inset-y-0 z-0"
            style={{ left: `${leftWidth}%`, right: 0 }}
          >
            {scale.gaps.map((gap) => (
              <div
                key={gap.startMs}
                data-testid="timeline-axis-break"
                className="border-muted bg-canvasMuted absolute inset-y-0 flex items-center justify-center overflow-hidden border-x border-dashed"
                style={{
                  left: `${gap.startPercent}%`,
                  width: `${gap.endPercent - gap.startPercent}%`,
                }}
              >
                <span
                  className="text-muted whitespace-nowrap text-[10px] tabular-nums"
                  style={{ writingMode: 'vertical-rl' }}
                >
                  ⋯ {formatDuration(gap.durationMs)} ⋯
                </span>
              </div>
            ))}
          </div>
        )}

        <div className="relative z-[1]">
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
              scale={scale}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
