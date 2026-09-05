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

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type JSX,
  type ReactNode,
} from 'react';
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
import { TIMELINE_CONSTANTS, calculateBarPosition, calculateDuration } from './utils/timing';
// IDLE_STYLES lives beside `leadInMs` because that function needs it too: two
// copies would drift into a bar that draws a lead-in without naming one. These
// are also the stretches the elastic axis may elide — they cost nothing, and a
// nine-day sleep drawn to scale leaves no room for the five seconds that ran.
// `step.invoke` is deliberately absent: a child run is genuinely executing.
import { IDLE_STYLES, leadInMs } from './utils/traceConversion';

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
        // Neutral, not the colour of either attempt around it — see
        // `timing.backoff` in TimelineBar's style table.
        style: 'timing.backoff',
        startMs: previousEnd,
        endMs: started,
        tooltip:
          i === 0
            ? `Queued ${formatDuration(started - previousEnd)} before the first attempt`
            : `Backed off ${formatDuration(started - previousEnd)} before attempt ${i + 1}`,
      });
    }

    if (ended > started) {
      segments.push({
        id: `${bar.id}-attempt-${i}-run`,
        startPercent: pct(started - barStart),
        widthPercent: pct(ended - started),
        style: 'step.run',
        status: attempt.status,
        startMs: started,
        endMs: ended,
        tooltip: `Attempt ${i + 1} of ${attempts.length} — ${(
          attempt.status ?? 'ran'
        ).toLowerCase()} after ${formatDuration(ended - started)}`,
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
  // This branch is exclusive: the root NEVER continues on to the breakdown
  // below, whatever it finds here. Guarding the breakdown instead left two ways
  // through — a clamped bar whose `delayMs` had been reclaimed into the label,
  // and a still-running root with no `endTime` — and both drew the synthesised
  // figure as a lead-in. `chains` came out as 86% waiting, `cancelled` 99.997%,
  // each directly above steps drawn as executing over the same interval.
  //
  // Returning nothing is not a gap: it gives the row one solid bar, which is
  // exactly what a run with nothing to say about its queue time should look
  // like. The queued portion is reported as text on the row either way.
  if (bar.isRoot) {
    if (!bar.endTime || !bar.delayMs || bar.delayMs <= 0) return undefined;

    const spanMs = bar.endTime.getTime() - bar.startTime.getTime();
    if (spanMs <= 0) return undefined;

    const waitPercent = (bar.delayMs / spanMs) * 100;
    return [
      {
        id: `${bar.id}-seg-run-queued`,
        startPercent: 0,
        widthPercent: waitPercent,
        style: 'timing.waiting',
        status: bar.status,
        startMs: bar.startTime.getTime(),
        endMs: bar.startTime.getTime() + bar.delayMs,
        tooltip: `Queued ${formatDuration(bar.delayMs)} before the run started`,
      },
      {
        id: `${bar.id}-seg-run-running`,
        startPercent: waitPercent,
        widthPercent: 100 - waitPercent,
        style: 'root',
        status: bar.status,
        startMs: bar.startTime.getTime() + bar.delayMs,
        endMs: bar.endTime.getTime(),
        tooltip: `Running ${formatDuration(spanMs - bar.delayMs)}`,
      },
    ];
  }

  // A widened bar leads with the request that planned it.
  //
  // Drawn at its real interval in the discovery style, so it is visibly the
  // platform's work rather than the step's, and identical on every row the same
  // request produced — which is what makes a fan-out read as one request
  // opening into several steps instead of N unexplained gaps.
  const leading: BarSegment[] = [];
  if (bar.unaccounted && bar.endTime) {
    const barStart = bar.startTime.getTime();
    const spanMs = bar.endTime.getTime() - barStart;
    const ms = bar.unaccounted.endMs - bar.unaccounted.startMs;
    if (spanMs > 0) {
      leading.push({
        id: `${bar.id}-unaccounted`,
        startPercent: ((bar.unaccounted.startMs - barStart) / spanMs) * 100,
        widthPercent: (ms / spanMs) * 100,
        style: 'timing.unaccounted',
        startMs: bar.unaccounted.startMs,
        endMs: bar.unaccounted.endMs,
        tooltip: `${formatDuration(
          ms
        )} that no span accounts for — queueing, concurrency, latency or processing, but nothing reported which`,
      });
    }
  }

  const planning: BarSegment[] = [...leading];
  if (bar.planning && bar.endTime) {
    const barStart = bar.startTime.getTime();
    const spanMs = bar.endTime.getTime() - barStart;
    const width = ((bar.planning.endMs - bar.planning.startMs) / spanMs) * 100;
    if (spanMs > 0 && width > 0) {
      const others = bar.planning.steps.filter((n) => n !== bar.name);
      planning.push({
        id: `${bar.id}-planning`,
        startPercent: ((bar.planning.startMs - barStart) / spanMs) * 100,
        widthPercent: width,
        style: 'timing.inngest.discovery',
        startMs: bar.planning.startMs,
        endMs: bar.planning.endMs,
        tooltip: others.length
          ? `Inngest planned this and ${others.length} other step${
              others.length === 1 ? '' : 's'
            } in one request: ${others.join(', ')}`
          : 'Inngest planned this step in its own request',
      });
    }
  }

  if (!bar.timingBreakdown) {
    return planning.length ? planning : undefined;
  }

  const { executionMs } = bar.timingBreakdown;
  let { inngestMs, totalMs } = bar.timingBreakdown;

  // The bar may start earlier than the STEP does, when it is drawn wide enough
  // to show the request that planned it. The wait and execution below describe
  // the step, so they are measured from the step's own start and then placed
  // against the bar — otherwise the wait would swallow the planning request and
  // draw over the segment that already accounts for it.
  const barStartMs = bar.startTime.getTime();
  const drawnMs = bar.endTime ? bar.endTime.getTime() - barStartMs : null;
  const stepStartMs =
    bar.reportedMs !== undefined && bar.endTime
      ? bar.endTime.getTime() - bar.reportedMs
      : barStartMs;

  // The step's own extent, which is what the wait/execution split is about.
  const spanMs = bar.endTime ? bar.endTime.getTime() - stepStartMs : null;

  if (spanMs !== null && spanMs > 0) {
    // Trust the timestamps over the metadata, ALWAYS — not only when the
    // metadata overshoots. Anything left of the span after execution is this
    // step's own overhead, and `leadInMs` is that derivation. The row's
    // `+72ms wait` label comes through the same function, so the bar and its
    // own label cannot disagree.
    //
    // Doing this only on an overshoot left them disagreeing on the undershoot:
    // `chains`' `left-2` labelled `+149ms wait` while its bar reported waiting
    // 119ms and running 17ms — 30ms of a 166ms span attributed to nothing at
    // all, and a sibling one row down doing it differently.
    inngestMs = leadInMs(bar);
    totalMs = spanMs;
  }

  if (totalMs <= 0 || drawnMs === null || drawnMs <= 0) {
    return planning.length ? planning : undefined;
  }

  const segments: BarSegment[] = [...planning];
  // Percentages are of the DRAWN bar; the millisecond figures are the step's.
  const pctOf = (ms: number) => (ms / drawnMs) * 100;
  const offset = pctOf(stepStartMs - barStartMs);
  let currentPercent = offset;

  // Inngest overhead segment — short gray delay bar
  if (inngestMs > 0) {
    const inngestPercent = pctOf(inngestMs);
    segments.push({
      id: `${bar.id}-seg-delay`,
      startPercent: currentPercent,
      widthPercent: inngestPercent,
      style: 'timing.waiting',
      status: bar.status,
      startMs: stepStartMs,
      endMs: stepStartMs + inngestMs,
      // The same number the row's `+72ms wait` label carries, from the same
      // derivation, so hovering a lead-in confirms the label rather than
      // offering the reader a third figure to reconcile.
      tooltip: `Waited ${formatDuration(inngestMs)} before this step ran`,
    });
    currentPercent += inngestPercent;
  }

  // Execution segment — root bar uses short status-colored bar, steps use tall barber-pole
  if (executionMs > 0) {
    const execPercent = pctOf(executionMs);
    segments.push({
      id: `${bar.id}-seg-server`,
      startPercent: currentPercent,
      widthPercent: execPercent,
      style: bar.isRoot ? 'root' : 'timing.server',
      status: bar.status,
      startMs: stepStartMs + inngestMs,
      endMs: stepStartMs + inngestMs + executionMs,
      // What the canvas node reports for this step, said in the trace too.
      tooltip: `Ran ${formatDuration(executionMs)} on your server`,
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
      tooltip: `Waited ${formatDuration(bar.delayMs)} before this ran`,
    });
  }

  if (execPercent > 0) {
    segments.push({
      id: `${bar.id}-seg-exec`,
      startPercent: delayPercent,
      widthPercent: execPercent,
      style: bar.style,
      status: bar.status,
      tooltip: `Ran ${formatDuration(totalMs - bar.delayMs)}`,
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

// ============================================================================
// Hover Tooltip Timing Details
// ============================================================================

/** Human-readable labels for bar style keys shown in the hover tooltip. */
const STYLE_LABELS: Partial<Record<BarStyleKey, string>> = {
  'timing.waiting': 'Waiting to run',
  'timing.unaccounted': 'Not reported',
  'timing.backoff': 'Suspended between attempts',
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

  // Every row's number is the width it occupies on the axis. No exceptions,
  // including the Run row.
  //
  // Reporting the run's whole life here instead was a regression across 43 of
  // 45 fixtures: the number moved and the plot did not, so `simple` drew a
  // full-width bar labelled 165ms against a ruler reading 0ms → 32ms. A ruler
  // that the largest bar on the page contradicts is worse than a number that
  // needs its note to complete it.
  //
  // The queued part is carried by the note, whose wording says whether it is
  // inside this number or on top of it.
  // The step's own span. The bar may be drawn wider than this to show the
  // request that planned it, which is a real interval belonging to this step's
  // causation — but the number beside the row stays the step's own, because
  // inflating a step's duration with the platform's work is the mistake this
  // whole approach exists to avoid.
  const duration = bar.reportedMs ?? calculateDuration(bar.startTime, bar.endTime);
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
  //
  // A sleep or a waitForEvent is decomposed by neither generator. Waiting IS the
  // substance of one, so splitting it into "waiting, then working" left a bar
  // that was ~100% pale lead-in: on `cancelled` the 35.2s wait, which is the
  // single most important fact in that run, drew as a 26%-opacity smudge
  // underneath the axis break — which is to say as empty space. One solid bar in
  // its own style says what happened.
  const segments = IDLE_STYLES.has(bar.style)
    ? undefined
    : bar.segments ?? generateBarSegments(bar) ?? generateDelaySegments(bar);

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

  return (
    <TimelineBar
      name={bar.name}
      duration={duration}
      startPercent={startPercent}
      widthPercent={widthPercent}
      depth={depth}
      leftWidth={leftWidth}
      style={bar.style}
      // A platform row is not a call the reader made. `Finalization` typed
      // itself `step.run`, which names a line of the user's code that does not
      // exist — the whole point of the subtitle is that it says what they wrote.
      styleLabel={bar.isPlatform ? 'Inngest' : STYLE_LABELS[bar.style]}
      segments={segments}
      // Root bar is always expanded (children always visible) but not expandable
      // (no toggle UI). expandable=false ensures VisualBar keeps opacity 1.
      expandable={bar.isRoot ? false : isExpandable}
      expanded={isExpanded}
      onToggle={bar.isRoot ? undefined : () => onToggleExpand(bar.id, bar.childRunID)}
      onClick={() => onSelectStep?.(bar.id)}
      selected={selectedStepId === bar.id}
      badgeGutter={badgeGutter}
      barID={bar.id}
      note={bar.note}
      interrupted={bar.interrupted}
      platform={bar.isPlatform}
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
        // A drawn queue delay is not elidable, because the Run row draws a
        // lead-in ACROSS it. A bar's segments are laid out as percentages of
        // the bar while the bar itself is placed through the scale, so a break
        // falling inside one pushes its own segments off: on `blocked` the Run
        // row was still drawn "queued" a third of the way past the point where
        // `hold` below it was drawn executing. Keeping the interval busy means
        // the delay is drawn to scale, which is what that run is about anyway.
        if (bar.isRoot && bar.delayMs && bar.delayMs > 0) {
          busy.push({
            startMs: bar.startTime.getTime(),
            endMs: bar.startTime.getTime() + bar.delayMs,
          });
        }

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

  // Where each request's lines run, measured after layout.
  //
  // Row positions are read from the DOM rather than derived from an index,
  // because what is expanded above a row changes where it sits and an index is
  // only right when nothing is.
  const [rowTops, setRowTops] = useState<ReadonlyMap<string, number>>(new Map());

  useLayoutEffect(() => {
    const plot = plotRef.current;
    if (!plot) return;

    const measure = () => {
      const box = plot.getBoundingClientRect();
      const next = new Map<string, number>();
      for (const el of plot.querySelectorAll('[data-bar-id]')) {
        const id = el.getAttribute('data-bar-id');
        if (!id) continue;
        const r = el.getBoundingClientRect();
        next.set(id, r.top - box.top + r.height / 2);
      }
      setRowTops((prev) => {
        if (prev.size === next.size && [...next].every(([k, v]) => prev.get(k) === v)) return prev;
        return next;
      });
    };

    measure();

    // jsdom has no ResizeObserver, and the lines are a rendering concern with
    // nothing to observe there. Measuring once is correct in that environment
    // and re-measuring on resize is correct in a browser.
    if (typeof ResizeObserver === 'undefined') return;
    const observer = new ResizeObserver(measure);
    observer.observe(plot);
    return () => observer.disconnect();
  });

  /**
   * One request, several steps leading off it — drawn the way the flow chart
   * draws causation.
   *
   * A trace states facts and says nothing about what led to what. These are the
   * one relationship the payload actually knows: `plannedStepIDs` names the
   * steps a request produced, so the line is reported rather than inferred. It
   * leaves the request where it finished and arrives at each step where that
   * step's own bar begins.
   */
  const planningFlows = useMemo(() => {
    const byID = new Map<string, TimelineBarData>();
    const collect = (list: TimelineBarData[]) => {
      for (const bar of list) {
        byID.set(bar.id, bar);
        if (bar.children?.length) collect(bar.children);
      }
    };
    collect(barsWithChildren);

    // Pixels, both axes. The lines are drawn in a plain SVG over the plot, and
    // a percentage x against a pixel y in one coordinate system does not map.
    const plotFraction = (100 - leftWidth) / 100;
    const xOf = (ms: number) =>
      ((leftWidth + scale.toPercent(ms) * plotFraction) / 100) * (plotWidth ?? 0);

    const paths: Array<{ key: string; d: string }> = [];
    if (!plotWidth) return paths;

    for (const bar of byID.values()) {
      if (!bar.planning) continue;
      const fromY = rowTops.get(bar.id);
      if (fromY === undefined) continue;

      const fromX = xOf(bar.planning.endMs);
      for (const other of byID.values()) {
        if (other.id === bar.id || other.plannedBy !== bar.planning.spanID) continue;
        const toY = rowTops.get(other.id);
        if (toY === undefined) continue;

        // Down from where the request finished, then a small quarter-turn into
        // the step's own start. Percentages horizontally, pixels vertically —
        // the two axes of this view are not the same kind of thing.
        const toX = xOf(other.startTime.getTime());
        const turn = Math.min(6, Math.abs(toY - fromY) / 2);
        paths.push({
          key: `${bar.planning.spanID}-${other.id}`,
          d: `M ${fromX} ${fromY} L ${fromX} ${toY - turn} Q ${fromX} ${toY} ${
            fromX + 0.6
          } ${toY} L ${toX} ${toY}`,
        });
      }
    }
    return paths;
  }, [barsWithChildren, rowTops, scale, leftWidth, plotWidth]);

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
                // Unfilled on purpose. A break is regularly occupied — a sleep
                // or a waitForEvent is a row drawn across the stretch being
                // elided — and a filled band painted its own content out:
                // `cancelled`'s 35.2s wait was drawn inside it and could not be
                // seen. The dashed rules and the rotated duration are enough to
                // read it as a break in the axis.
                className="border-muted absolute inset-y-0 flex items-center justify-center overflow-hidden border-x border-dashed"
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

        {planningFlows.length > 0 && (
          <svg
            className="pointer-events-none absolute inset-0 z-[2] h-full w-full"
            style={{ overflow: 'visible' }}
          >
            {planningFlows.map((flow) => (
              <path
                key={flow.key}
                d={flow.d}
                fill="none"
                stroke="rgb(var(--color-border-muted))"
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
              />
            ))}
          </svg>
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
