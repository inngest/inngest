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

import { useCallback, useEffect, useMemo, useRef, useState, type JSX, type ReactNode } from 'react';
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
import { applyCollapseToBars, markBudget, type CollapsePlan } from './canvas/collapse';
import { formatDuration, useStepHover, useStepSelection } from './runDetailsUtils';
import { packMinimap } from './utils/density';
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
  /**
   * Loads another run's timeline, so a `step.invoke` can show the run it started
   * beneath it on the same axis rather than sending the user somewhere else.
   *
   * Called only when a row is actually expanded — a deep tree must not fetch the
   * world on first paint — and its result is cached for the life of the view.
   */
  loadChildRun?: (runID: string) => Promise<TimelineBarData[] | null>;
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
 *
 * THE BAR IS THE SPAN, AND SEGMENTS PARTITION IT. A breakdown that totals more
 * than the span it decorates is describing something outside that span, and
 * must not be used to size it — otherwise the bar and its own duration label
 * disagree, which is the most direct way this view can lie.
 *
 * `blocked` is the case that found this. Its `hold` step ran for 5999ms and
 * queued for none of it, but carries `inngest.timing` metadata with
 * `queue_delay_ms: 6299` — the RUN-level concurrency hold, stamped onto the
 * step. The bar was drawn from the 12298ms metadata total while the label came
 * from the 5999ms span, so a 6s step rendered ~12s wide with half of it shown
 * as waiting the step never did. The 6.3s hold is real, and belongs to the run
 * span, which reports it correctly one row up.
 */
/**
 * Draw a retried step as its attempts.
 *
 * A step that failed, waited, and succeeded is the clearest thing the platform
 * does for anyone — and collapsed it was a plain green bar, with the whole story
 * hidden behind a disclosure triangle. On `retry` the payload has all of it:
 * attempt 0 runs 137→206ms and FAILS, then nothing until 1209ms (the backoff),
 * then attempt 1 runs 10ms and succeeds.
 *
 * So each attempt contributes a ghosted stretch for the time it spent waiting
 * and a solid one for the time it ran, each coloured by that attempt's own
 * outcome. The result reads as: faint red, red, a long faint green wait, green.
 * The retry is visible without expanding anything, and the backoff — time the
 * platform spent waiting on the user's behalf — becomes a thing you can see
 * rather than a gap.
 */
const segmentEnd = (bar: TimelineBarData) => (bar.endTime ?? bar.startTime).getTime();

function generateAttemptSegments(bar: TimelineBarData): BarSegment[] | undefined {
  const attempts = bar.children?.filter((child) => /^Attempt \d+$/.test(child.name));
  if (!attempts || attempts.length < 2 || !bar.endTime) return undefined;

  const barStart = bar.startTime.getTime();
  const spanMs = bar.endTime.getTime() - barStart;
  if (spanMs <= 0) return undefined;

  const pct = (ms: number) => (ms / spanMs) * 100;
  const segments: BarSegment[] = [];

  attempts.forEach((attempt, i) => {
    // A bar starts at queuedAt and carries its own queue delay, which is the
    // only way back to when it actually began executing.
    const queued = attempt.startTime.getTime();
    const started = queued + (attempt.delayMs ?? 0);
    const ended = (attempt.endTime ?? attempt.startTime).getTime();

    // Everything from the end of the previous attempt up to this one starting is
    // waiting: for the first attempt its own queue time, for the rest the retry
    // backoff. Drawn as one ghosted stretch rather than left blank, because a
    // gap is the platform waiting on the user's behalf and is worth seeing.
    const previousEnd = i === 0 ? barStart : segmentEnd(attempts[i - 1]!);
    if (started > previousEnd) {
      segments.push({
        id: `${bar.id}-attempt-${i}-wait`,
        startPercent: pct(previousEnd - barStart),
        widthPercent: pct(started - previousEnd),
        style: 'timing.waiting',
        status: attempt.status,
      });
    }

    if (ended > started) {
      segments.push({
        id: `${bar.id}-attempt-${i}-run`,
        startPercent: pct(started - barStart),
        widthPercent: pct(ended - started),
        style: 'step.run',
        status: attempt.status,
      });
    }
  });

  return segments.length > 1 ? segments : undefined;
}

export function generateBarSegments(bar: TimelineBarData): BarSegment[] | undefined {
  // Attempts win: when a step was retried, that is the most important thing
  // about it, and the phase breakdown describes only the attempt that stuck.
  const attemptSegments = generateAttemptSegments(bar);
  if (attemptSegments) return attemptSegments;

  // The RUN row is drawn from its own clock, not from a breakdown.
  //
  // The root's `timingBreakdown` is synthesised as "total minus what the
  // children spent executing", so every scrap of delay scattered through the
  // run — discovery between steps, system latency, finalization — is summed
  // into one number. Drawn as a segment that number becomes a single block at
  // the FRONT of the bar, claiming the run sat still for all of it before
  // anything happened. On `failure` that rendered as 53% of the run spent
  // waiting when it was queued for 156ms of 653ms, and the reader is invited to
  // conclude the platform sat on their run for half its life.
  //
  // What is actually true of the run, and all that is claimed here: it was
  // queued for `delayMs`, and then it was running.
  if (bar.isRoot && bar.endTime && bar.delayMs !== undefined) {
    const spanMs = bar.endTime.getTime() - bar.startTime.getTime();
    if (spanMs > 0 && bar.delayMs > 0) {
      const waitPercent = (bar.delayMs / spanMs) * 100;
      return [
        {
          id: `${bar.id}-seg-run-queued`,
          startPercent: 0,
          widthPercent: waitPercent,
          style: 'timing.waiting',
          status: bar.status,
        },
        {
          id: `${bar.id}-seg-run-running`,
          startPercent: waitPercent,
          widthPercent: 100 - waitPercent,
          style: 'root',
          status: bar.status,
        },
      ];
    }
  }

  if (!bar.timingBreakdown) return undefined;

  const { executionMs } = bar.timingBreakdown;
  let { inngestMs, totalMs } = bar.timingBreakdown;

  // The span's own extent, which is what the bar is actually drawn across.
  const spanMs = bar.endTime ? bar.endTime.getTime() - bar.startTime.getTime() : null;

  if (spanMs !== null && spanMs > 0 && totalMs > spanMs) {
    // Trust the timestamps over the metadata and re-derive the split from the
    // span: anything left over after execution is this step's own overhead.
    inngestMs = Math.max(0, spanMs - executionMs);
    totalMs = spanMs;
  }

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
  onToggleExpand: (barId: string, childRunID?: string) => void;
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
  /** Whether any row in this timeline has a badge; keeps bars aligned. */
  badgeGutter?: boolean;
  /** Span id currently hovered anywhere in this run. */
  hoveredStepId?: string;
  /** Emits this row as hovered, for the other views. */
  onHoverStep?: (spanID: string | undefined) => void;
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
  badgeGutter,
  hoveredStepId,
  onHoverStep,
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
  // When a row decomposes into attempts, THAT is its decomposition. Expanding it
  // must unpack the bar the reader was just looking at — not replace it with a
  // different one (Inngest / your server) whose parts line up with nothing on
  // screen. Expansion splits a row into its own segments; nothing new appears.
  const hasAttemptSegments = !!generateAttemptSegments(bar);
  const hasInngestBreakdown = !!bar.inngestBreakdown;
  const isExpandable =
    hasTimingBreakdown ||
    hasInngestBreakdown ||
    hasChildren ||
    hasRunInngestBreakdown ||
    // An invoke can be opened before its child has been fetched — that is the
    // point of it being lazy.
    Boolean(bar.childRunID);
  const isExpanded = bar.isRoot ? true : expandedBars.has(bar.id);

  // Children of experiment bars inherit the dotted background
  const childInsideExperiment = insideExperiment || bar.hasExperiment;

  // Generate segments for compound bar visualization
  // Bars with timingBreakdown use queue+execution segments; others fall back to delay+execution
  // A collapsed group brings its own segments — one per member — so the row
  // shows where in the sequence things happened rather than one solid block.
  const segments = bar.segments ?? generateBarSegments(bar) ?? generateDelaySegments(bar);

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
      onToggle={bar.isRoot ? undefined : () => onToggleExpand(bar.id, bar.childRunID)}
      onClick={() => onSelectStep?.(bar.id)}
      selected={selectedStepId === bar.id}
      badgeGutter={badgeGutter}
      note={bar.note}
      hovered={hoveredStepId === bar.id}
      onHoverChange={(on) => onHoverStep?.(on ? bar.id : undefined)}
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
      {isExpanded && !bar.isRoot && !hasAttemptSegments && timingPositions?.inngest && (
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
      {isExpanded && !bar.isRoot && !hasAttemptSegments && timingPositions?.server && (
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
                badgeGutter={badgeGutter}
                hoveredStepId={hoveredStepId}
                onHoverStep={onHoverStep}
              />
            ))}
        </TimelineBar>
      )}

      {/* The run-level "Inngest" row used to live here. It is gone deliberately.

          It was named after the company rather than after anything the reader
          wrote, it spanned the whole run while showing two disconnected
          fragments inside it, and it did not line up with the Run bar directly
          above it — two rows describing the same 156ms, drawn in different
          places. Everything it carried is now somewhere better: the queue delay
          is the Run bar's own ghosted lead-in, and finalization is already its
          own real span row. */}

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
            badgeGutter={badgeGutter}
            hoveredStepId={hoveredStepId}
            onHoverStep={onHoverStep}
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
export function Timeline({
  data,
  onSelectStep,
  runID,
  collapse,
  loadChildRun,
}: Props): JSX.Element {
  const { minTime, maxTime, leftWidth, orgName } = data;

  // Repetition is detected once, in the model, and applied to both views. A
  // group becomes one row whose children are its members, so opening it uses
  // the expansion affordance that is already here rather than a second one.
  // How wide the plot actually is, so a collapsed group draws as many marks as
  // will fit and no more. A narrow pane gets fewer, larger marks rather than a
  // smear; measuring beats guessing a constant.
  const plotRef = useRef<HTMLDivElement>(null);
  const [plotWidth, setPlotWidth] = useState<number | undefined>(undefined);

  useEffect(() => {
    const el = plotRef.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry) setPlotWidth(entry.contentRect.width);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Child runs pulled in beneath their invoke row, keyed by run id. Loaded on
  // first expand and kept — a deep tree must not fetch the world on first
  // paint, but it should not re-fetch on every toggle either.
  const [childRuns, setChildRuns] = useState<Record<string, TimelineBarData[]>>({});
  const loadingChildren = useRef(new Set<string>());

  const bars = useMemo(
    () =>
      collapse
        ? applyCollapseToBars(
            data.bars,
            collapse,
            undefined,
            // Only the plot half of the row is available to marks; the label
            // column takes the rest.
            markBudget(plotWidth && (plotWidth * (100 - leftWidth)) / 100)
          )
        : data.bars,
    [data.bars, collapse, plotWidth, leftWidth]
  );

  // Graft any loaded child run beneath the invoke row that started it. Done
  // here rather than in the conversion so the fetch stays lazy and the tree
  // rebuilds only when something new has actually arrived.
  const barsWithChildren = useMemo(() => {
    if (!Object.keys(childRuns).length) return bars;
    const graft = (list: TimelineBarData[]): TimelineBarData[] =>
      list.map((bar) => {
        const loaded = bar.childRunID ? childRuns[bar.childRunID] : undefined;
        const children = bar.children ? graft(bar.children) : undefined;
        return loaded
          ? { ...bar, children: [...(children ?? []), ...loaded] }
          : { ...bar, children };
      });
    return graft(bars);
  }, [bars, childRuns]);

  // Decided once for the whole timeline so bars stay aligned across rows.
  const badgeGutter = useMemo(() => {
    const anyBadge = (list: TimelineBarData[]): boolean =>
      list.some(
        (bar) => bar.hasExperiment || (bar.scores?.length ?? 0) > 0 || anyBadge(bar.children ?? [])
      );
    return anyBadge(bars);
  }, [bars]);

  const rootBarIds = useMemo(() => bars.filter((bar) => bar.isRoot).map((bar) => bar.id), [bars]);

  // Initialize with root bars expanded by default
  const [expandedBars, setExpandedBars] = useState<Set<string>>(() => new Set(rootBarIds));

  // Read the highlighted row from the shared selection rather than local state,
  // so selecting a node on the canvas highlights the matching row here. Bar ids
  // are the rolled-up span ids, which is exactly what the canvas emits.
  const { selectedStep } = useStepSelection({ runID });
  const selectedStepId = selectedStep?.trace.spanID;

  // One subscriber for the whole list. Hovering a row here highlights the
  // matching node on the canvas and vice versa; rows only compare a string.
  const { hoveredSpanID, hoverStep } = useStepHover({ runID });

  // Timeline brush selection state (for zooming)
  const [viewStartOffset, setViewStartOffset] = useState(0);
  const [viewEndOffset, setViewEndOffset] = useState(100);

  // Pull in a child run the first time its row is opened, and only then.
  const ensureChildRun = useCallback(
    (runID: string) => {
      if (!loadChildRun) return;
      if (childRuns[runID] || loadingChildren.current.has(runID)) return;
      loadingChildren.current.add(runID);
      void loadChildRun(runID).then((childBars) => {
        if (childBars?.length) setChildRuns((prev) => ({ ...prev, [runID]: childBars }));
      });
    },
    [loadChildRun, childRuns]
  );

  const handleToggleExpand = useCallback(
    (barId: string, childRunID?: string) => {
      setExpandedBars((prev) => {
        const next = new Set(prev);
        if (next.has(barId)) {
          next.delete(barId);
        } else {
          next.add(barId);
          if (childRunID) ensureChildRun(childRunID);
        }
        return next;
      });
    },
    [ensureChildRun]
  );

  const handleHoverStep = useCallback(
    (spanID: string | undefined) => {
      if (!runID) return;
      hoverStep(spanID ? { spanID, runID } : undefined);
    },
    [hoverStep, runID]
  );

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
  const minimap = useMemo(() => packMinimap(data.bars, scale), [data.bars, scale]);

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
        minimap={minimap}
        hoveredStepId={hoveredSpanID}
      />

      {/* The rows, with the axis breaks drawn behind them. A break spans every
          row because it is a property of the axis rather than of any one step —
          and it is drawn at all because an axis that is not linear has to say
          so. Elided time is a filled band bounded by dashed rules, with its real
          duration set inside it: visibly a break, not a gap. */}
      <div className="relative" ref={plotRef}>
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
          {barsWithChildren.map((bar) => (
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
              badgeGutter={badgeGutter}
              hoveredStepId={hoveredSpanID}
              onHoverStep={handleHoverStep}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
