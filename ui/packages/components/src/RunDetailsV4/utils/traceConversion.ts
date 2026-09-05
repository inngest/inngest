/**
 * Utilities to convert V3 Trace data to V4 TimelineData format.
 * Feature: 001-composable-timeline-bar
 */

import { maxDateString, toMaybeDate } from '@inngest/components/utils/date';
import { max, min } from 'date-fns';

import { scoreRows } from '../../RunDetails/ScoresAttrs';
import { KindInngestExperiment } from '../../generated';
import type {
  BarStyleKey,
  HTTPTimingBreakdownData,
  InngestBreakdownData,
  ScoreBadgeData,
  TimelineBarData,
  TimelineData,
} from '../TimelineBar.types';
import { formatDuration, traceWalk } from '../runDetailsUtils';
import {
  isExperimentMetadata,
  isScoreMetadata,
  isStepInfoRun,
  type RunDiscovery,
  type SpanMetadata,
  type SpanMetadataInngestHTTPTiming,
  type SpanMetadataInngestTiming,
  type Trace,
} from '../types';
import { TIMELINE_CONSTANTS } from './timing';

/**
 * Above this, a run's queue delay is drawn on the axis; below it, it is reported
 * as text and the plot starts when the run did. A second is the same line item G
 * uses before claiming flow control did something — under it, this is ordinary
 * scheduling rather than an event.
 */
const NOTEWORTHY_QUEUE_DELAY_MS = 1000;

/**
 * Bars that represent the run being suspended rather than working.
 *
 * Defined here rather than in `Timeline` because `leadInMs` needs it too, and
 * two copies of this set would drift into a bar that draws a lead-in without
 * naming one, or names one without drawing it.
 */
export const IDLE_STYLES = new Set<BarStyleKey>(['step.sleep', 'step.waitForEvent']);

/**
 * Rows that are the platform's own machinery rather than a step the reader
 * wrote. They still earn their place — finalization genuinely takes time, and on
 * `retry` it was itself retried — but they are not steps, so anything counting
 * or laying out steps must skip them. The minimap was packing `Finalization`
 * into a lane as though the function had written it.
 */
const PLATFORM_ROW_NAMES = new Set(['Finalization', 'Function error']);

/**
 * Check if a trace represents a step.run span
 */
function isStepRunSpan(trace: Trace): boolean {
  return trace.stepOp === 'RUN' || trace.stepType === 'RUN' || isStepInfoRun(trace.stepInfo);
}

function isNonStepSpan(trace: Trace): boolean {
  return !trace.stepOp && !trace.stepType;
}

/**
 * Get the display name for a span
 */
function getSpanName(name: string): string {
  // Remove common prefixes/suffixes that aren't useful for display
  return name.replace(/^step\./, '').replace(/^inngest\//, '');
}

/**
 * Get the style key for a trace based on its type
 */
function getStyleForTrace(trace: Trace): BarStyleKey {
  // Root bar gets special checkbox icon style
  if (trace.isRoot) {
    return 'root';
  }

  if (trace.isUserland) {
    return 'default';
  }

  const stepOp = trace.stepOp?.toUpperCase() ?? trace.stepType?.toUpperCase();

  switch (stepOp) {
    case 'RUN':
      return 'step.run';
    case 'SLEEP':
      return 'step.sleep';
    case 'WAIT_FOR_EVENT':
    case 'WAITFOREVENT':
      return 'step.waitForEvent';
    case 'INVOKE':
      return 'step.invoke';
    default:
      return 'step.run';
  }
}

/**
 * Calculate timing breakdown for a step.run span
 */
function calculateTimingBreakdown(
  trace: Trace
): { inngestMs: number; executionMs: number; totalMs: number } | undefined {
  if (!trace.queuedAt) return undefined;

  const queuedAt = new Date(trace.queuedAt).getTime();
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : Date.now();

  // Calculate durations
  const inngestMs = startedAt
    ? Math.max(0, startedAt - queuedAt)
    : Math.max(0, Date.now() - queuedAt);
  const executionMs = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalMs = startedAt ? inngestMs + executionMs : Date.now() - queuedAt;

  return { inngestMs, executionMs, totalMs };
}

/**
 * Extract timing breakdown from inngest.timing metadata.
 * Returns server-computed queue delay and Inngest overhead when available.
 */
function getTimingFromMetadata(
  trace: Trace,
  metadata?: SpanMetadata[]
): { inngestMs: number; executionMs: number; totalMs: number } | null {
  if (!metadata) return null;

  const timing = metadata.find((m): m is SpanMetadataInngestTiming => m.kind === 'inngest.timing');

  if (!timing) return null;

  const inngestMs = timing.values.total_inngest_ms ?? 0;

  // Execution time is still derived from span timestamps since the metadata
  // captures Inngest-side overhead, not SDK execution duration.
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : Date.now();
  const executionMs = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalMs = inngestMs + executionMs;

  return { inngestMs, executionMs, totalMs };
}

/**
 * Extract per-step Inngest overhead breakdown from metadata + timestamps.
 * Combines discovery time (from timestamps) with concurrency delay and
 * system latency (from inngest.timing metadata).
 */
function getInngestBreakdown(
  trace: Trace,
  discoveryStartAtMs: number | null
): InngestBreakdownData | null {
  if (!trace.queuedAt) return null;

  // Discovery: time from the previous completed sibling to when this step was queued.
  const stepQueuedAt = new Date(trace.queuedAt).getTime();
  const discoveryMs =
    discoveryStartAtMs !== null ? Math.max(0, stepQueuedAt - discoveryStartAtMs) : 0;

  // Concurrency delay + system latency from metadata
  let queueDelayMs = 0;
  let systemLatencyMs = 0;

  const timing = trace.metadata?.find(
    (m): m is SpanMetadataInngestTiming => m.kind === 'inngest.timing'
  );
  if (timing) {
    queueDelayMs = timing.values.queue_delay_ms ?? 0;
    systemLatencyMs = timing.values.system_latency_ms ?? 0;
  }

  const totalMs = discoveryMs + queueDelayMs + systemLatencyMs;
  if (totalMs <= 0) return null;

  return { discoveryMs, queueDelayMs, systemLatencyMs, totalMs };
}

function tracesToBarData(
  traces: Trace[] | undefined,
  orgName?: string,
  rootStatus?: string,
  initialDiscoveryStartAtMs?: number | null,
  functionSlug?: string
): TimelineBarData[] | undefined {
  if (!traces) return undefined;

  let latestCompletedSiblingEndMs = initialDiscoveryStartAtMs ?? null;
  const bars: TimelineBarData[] = [];
  for (const trace of traces) {
    const bar = traceToBarData(
      trace,
      orgName,
      rootStatus,
      latestCompletedSiblingEndMs,
      functionSlug
    );

    if (bar.endTime) {
      const endMs = bar.endTime.getTime();
      latestCompletedSiblingEndMs =
        latestCompletedSiblingEndMs === null ? endMs : Math.max(latestCompletedSiblingEndMs, endMs);
    }

    bars.push(bar);
  }

  // Top to bottom is the order things started.
  //
  // The spans arrive in the order the SDK reported them, which §3.5 says is
  // non-deterministic under parallelism: `t19-parallel` listed its three
  // parallel steps b, c, a. A waterfall whose rows are not in time order is
  // not a waterfall, and the canvas orders its lanes by start time for the
  // same reason — the two disagreeing about where the same three steps go is
  // the thing to avoid.
  //
  // A sequential run is already in this order, so this changes nothing there.
  // The tiebreak on name keeps the layout stable between renders as well as
  // between runs.
  return bars.sort(
    (a, b) =>
      a.startTime.getTime() + (a.delayMs ?? 0) - (b.startTime.getTime() + (b.delayMs ?? 0)) ||
      a.name.localeCompare(b.name)
  );
}

/**
 * Extract HTTP timing breakdown from span metadata.
 * Returns timing phases for Inngest's HTTP call to the SDK endpoint.
 */
function getHTTPTimingFromMetadata(metadata?: SpanMetadata[]): HTTPTimingBreakdownData | null {
  if (!metadata) return null;

  const httpTiming = metadata.find(
    (m): m is SpanMetadataInngestHTTPTiming => m.kind === 'inngest.http.timing'
  );

  if (!httpTiming) return null;

  return {
    dnsLookupMs: httpTiming.values.dns_lookup_ms,
    tcpConnectionMs: httpTiming.values.tcp_connection_ms,
    tlsHandshakeMs: httpTiming.values.tls_handshake_ms,
    serverProcessingMs: httpTiming.values.server_processing_ms,
    contentTransferMs: httpTiming.values.content_transfer_ms,
    totalMs: httpTiming.values.total_ms,
  };
}

/**
 * Extract scores recorded directly on this span. Child spans are deliberately
 * not walked: the timeline already renders them as their own bars.
 */
function getScores(metadata?: SpanMetadata[]): ScoreBadgeData[] | undefined {
  const scores = scoreRows(metadata?.filter(isScoreMetadata) ?? []).map(({ name, value }) => ({
    name,
    value,
  }));

  return scores.length > 0 ? scores : undefined;
}

/**
 * Convert a single Trace to TimelineBarData
 */
function traceToBarData(
  trace: Trace,
  orgName?: string,
  rootStatus?: string,
  discoveryStartAtMs?: number | null,
  functionSlug?: string
): TimelineBarData {
  const shouldShowTiming = (isStepRunSpan(trace) || isNonStepSpan(trace)) && !trace.isUserland;
  // Prefer server-computed timing from metadata, fall back to span-timestamp calculation
  let timingBreakdown = shouldShowTiming
    ? getTimingFromMetadata(trace, trace.metadata) ?? calculateTimingBreakdown(trace)
    : undefined;

  // HTTP timing applies to any step that Inngest calls via HTTP, not just step.run
  const httpTimingBreakdown = !trace.isRoot
    ? getHTTPTimingFromMetadata(trace.metadata) ?? undefined
    : undefined;

  // Each bar uses its own status for coloring. rootStatus is only used as a
  // fallback for bars that don't have a meaningful status of their own.
  const status = trace.status || rootStatus;

  // Actual queue delay: time from when the step was queued to when execution started
  const delayMs = trace.startedAt
    ? Math.max(0, new Date(trace.startedAt).getTime() - new Date(trace.queuedAt).getTime())
    : undefined;

  // Per-step Inngest overhead breakdown (discovery + metadata timing)
  const inngestBreakdown = shouldShowTiming
    ? getInngestBreakdown(trace, discoveryStartAtMs ?? null) ?? undefined
    : undefined;

  const traceStartedAtMs = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const traceQueuedAtMs = trace.queuedAt ? new Date(trace.queuedAt).getTime() : null;
  const childDiscoveryStartAtMs = trace.isRoot
    ? traceStartedAtMs ?? discoveryStartAtMs
    : traceStartedAtMs ?? traceQueuedAtMs ?? discoveryStartAtMs;

  // Check if this step has experiment metadata
  const hasExperiment = trace.metadata?.some((m) => m.kind === KindInngestExperiment) ?? false;

  // Extract experiment metadata for hover card display
  const experimentMd = trace.metadata?.find(isExperimentMetadata);
  const experimentMetadata = experimentMd
    ? {
        experimentName: experimentMd.values.experiment_name ?? experimentMd.values.name,
        variantSelected: experimentMd.values.variant,
        availableVariants: experimentMd.values.available_variants,
        variantWeights: experimentMd.values.variant_weights,
        functionSlug,
      }
    : undefined;

  const invokeInfo = trace.stepInfo as { runID?: string | null } | null | undefined;

  return {
    id: trace.spanID,
    stepID: trace.stepID ?? null,
    name: getSpanName(trace.name),
    isPlatform: PLATFORM_ROW_NAMES.has(getSpanName(trace.name)),
    // Present on step.invoke, and the only thing needed to pull the child run
    // in beneath this row.
    childRunID: invokeInfo?.runID ?? undefined,
    startTime: new Date(trace.queuedAt),
    endTime: trace.endedAt ? new Date(trace.endedAt) : null,
    style: getStyleForTrace(trace),
    children: tracesToBarData(
      trace.childrenSpans,
      orgName,
      rootStatus,
      childDiscoveryStartAtMs,
      functionSlug
    ),
    timingBreakdown,
    httpTimingBreakdown,
    inngestBreakdown,
    isRoot: trace.isRoot,
    status,
    delayMs,
    hasExperiment,
    experimentMetadata,
    scores: getScores(trace.metadata),
  };
}

/**
 * Attempt spans bucketed for rollup, produced by a single pass over the
 * run's direct children.
 */
type RollupGroups = {
  /** stepIDs in first-seen order, so rollups keep the original span order */
  stepOrder: string[];
  /** attempt spans per stepID, keyed by attempt number */
  steps: Map<string, Map<number, Trace>>;
  /**
   * Finalization-type spans (output, no stepID) that also carry no groupID,
   * so they can't be rolled into a step or finalization group. Usually
   * emitted unchanged, but dropped when the finalization rollup renders the
   * same terminal failure — the duplicate is the server's relabeled discovery
   * span mirroring the grouped failed attempts (EXE-1992).
   */
  ungroupedFinalizations: Trace[];
  /** attempt spans of the trailing group that never matched a step, if any */
  finalizationAttempts: Map<number, Trace> | null;
  /** step span with the latest endedAt; finalization can't start before it ends */
  lastStep: Trace | null;
};

/**
 * Classify the run's direct children for rollup.
 *
 * Spans with an outputID but no stepID (network failures, finalization) are
 * grouped by groupID. A group later claimed by a step span becomes extra
 * attempts of that step; a group that never matches a step is the run's
 * finalization.
 *
 * Classification is ORDER-SENSITIVE, deliberately: a grouped no-stepID span
 * is adopted by a step only when the step span appears LATER in
 * childrenSpans than the grouped span (see the `finalSpan?.groupID ==
 * child.groupID` check below). The happy path relies on this ordering — the
 * genuine finalization span shares the last step's groupID, and is only
 * classified as finalization (rather than being claimed as an extra attempt
 * of that step) because the server emits it after that step.
 */
function collectRollupGroups(children: Trace[]): RollupGroups {
  const stepOrder: string[] = [];
  const steps = new Map<string, Map<number, Trace>>();
  const ungroupedFinalizations: Trace[] = [];
  const groupedSpans = new Map<string, Map<number, Trace>>();
  let lastStepEndedAt: Date | null = null;
  let lastStep: Trace | null = null;
  let finalSpan: Trace | null = null;

  for (const child of children) {
    if (child.outputID && !child.stepID) {
      if (child.groupID) {
        finalSpan = child;
        const attempts = groupedSpans.get(child.groupID) ?? new Map<number, Trace>();
        attempts.set(child.attempts ?? 0, child);
        groupedSpans.set(child.groupID, attempts);
      } else {
        ungroupedFinalizations.push(child);
      }
      continue;
    }
    if (!child.stepID || child.attempts === null) {
      continue;
    }

    // The grouped spans we saw belong to this step, not finalization
    if (finalSpan?.groupID == child.groupID) {
      finalSpan = null;
    }

    if (!steps.get(child.stepID)) {
      stepOrder.push(child.stepID);
    }

    const endedAt = toMaybeDate(child.endedAt);
    if (!lastStepEndedAt || (endedAt && endedAt > lastStepEndedAt)) {
      lastStepEndedAt = endedAt;
      lastStep = child;
    }

    const attempts = steps.get(child.stepID) ?? new Map<number, Trace>();
    if (child.groupID) {
      // Associate any other spans with the same groupID (IE network failures/similar) with this step
      for (const [attempt, attemptSpan] of groupedSpans.get(child.groupID) ?? []) {
        attempts.set(attempt, attemptSpan);
      }
    }

    attempts.set(child.attempts, child);
    steps.set(child.stepID, attempts);
  }

  const finalizationAttempts = finalSpan?.groupID
    ? groupedSpans.get(finalSpan.groupID) ?? null
    : null;

  return { stepOrder, steps, ungroupedFinalizations, finalizationAttempts, lastStep };
}

/** First (lowest attempt number) and last (highest) attempt spans of a group */
function attemptBounds(attempts: Map<number, Trace>): {
  first: Trace;
  last: Trace;
  lastAttempt: number;
} {
  const lastAttempt = Math.max(...attempts.keys());
  return {
    first: attempts.get(Math.min(...attempts.keys())) as Trace,
    last: attempts.get(lastAttempt) as Trace,
    lastAttempt,
  };
}

/**
 * Rename each attempt span (in place) to "Attempt N" and return them ordered
 * by attempt number, for nesting under a rollup span.
 */
function toAttemptChildren(attempts: Map<number, Trace>): Trace[] {
  for (const attempt of attempts.values()) {
    attempt.name = `Attempt ${attempt.attempts}`;
  }
  return Array.from(attempts.values()).sort((a, b) => (a.attempts ?? 0) - (b.attempts ?? 0));
}

/**
 * Roll up a step's attempts into a single span: a single attempt passes
 * through unchanged; multiple attempts get a virtual span that spans the
 * first attempt's queue time to the last attempt's end, carries the last
 * attempt's status/output, and nests the attempts as children.
 */
function rollupStepAttempts(stepID: string, attempts: Map<number, Trace>): Trace {
  const { first, last, lastAttempt } = attemptBounds(attempts);
  if (attempts.size === 1) {
    return last;
  }

  const name = last.name; // capture before toAttemptChildren renames the attempts
  return {
    isRoot: false,
    isUserland: false,
    spanID: `${stepID}-rollup`, // virtual span
    groupID: last.groupID,
    name,
    attempts: lastAttempt,
    stepID: stepID,
    queuedAt: first.queuedAt,
    scheduledAt: first.scheduledAt,
    startedAt: first.startedAt,
    endedAt: last.endedAt,
    status: last.status,
    outputID: last.outputID,
    debugRunID: last.debugRunID,
    debugSessionID: last.debugSessionID,
    stepInfo: last.stepInfo,
    childrenSpans: toAttemptChildren(attempts),
    metadata: last.metadata,
    userlandSpan: null,
  };
}

/**
 * Roll up the trailing unmatched group's attempts into a single span. These
 * spans can carry queue timestamps from before the last step finished, so
 * start timestamps are clamped to the last step's end to keep the timeline
 * ordered.
 *
 * Naming is decided by how the group ended, not by attempt count — the final
 * discovery itself can retry (e.g. the app 500s on the final request, then
 * succeeds), so multiple attempts don't imply an unresolved step:
 * - Ends COMPLETED: only the finalization can produce the function's output
 *   with no stepID (a step that succeeds gets a stepID and is claimed out of
 *   the trailing group), so this is genuinely "Finalization".
 * - Ends FAILED: this group is the site of the run's failure, but the client
 *   can't tell why — the attempts may have died before the SDK identified the
 *   work, or the SDK may have returned the function's own terminal error.
 *   "Function error" is neutral and truthful for both, unlike "Finalization",
 *   which implies a normal wind-down.
 * - Still running: default to "Finalization" rather than flickering "Function
 *   error" on in-flight runs.
 *
 * Only a FAILED terminal status flips the label to "Function error" —
 * CANCELLED or any other non-COMPLETED terminal state deliberately keeps
 * "Finalization", matching the pre-existing default and avoiding label
 * flicker for groups that aren't cleanly resolved as failures.
 */
function rollupFinalization(attempts: Map<number, Trace>, lastStep: Trace | null): Trace {
  const { first, last } = attemptBounds(attempts);
  const notBefore = lastStep?.endedAt;
  const name = last.status === 'FAILED' ? 'Function error' : 'Finalization';

  if (attempts.size === 1) {
    last.name = name;
    last.queuedAt = maxDateString(last.queuedAt, notBefore);
    last.scheduledAt = maxDateString(last.scheduledAt, notBefore);
    last.startedAt = maxDateString(last.startedAt, notBefore);
    return last;
  }

  return {
    isRoot: false,
    isUserland: false,
    name,
    spanID: `final-rollup`, // virtual span
    groupID: last.groupID,
    attempts: last.attempts,
    queuedAt: maxDateString(first.queuedAt, notBefore),
    scheduledAt: maxDateString(first.scheduledAt, notBefore),
    startedAt: maxDateString(first.startedAt, notBefore),
    endedAt: last.endedAt,
    status: last.status,
    outputID: last.outputID,
    debugRunID: last.debugRunID,
    debugSessionID: last.debugSessionID,
    childrenSpans: toAttemptChildren(attempts),
    metadata: last.metadata,
    stepInfo: null,
    userlandSpan: null,
  };
}

// traceRollup groups attempts by stepID and groupID
// (for spans without stepIDs, typically finalization spans or failures hitting the SDK entirely)
// and creates virtual "rollup" spans that represent the entire step/finalization
// with all attempts grouped together. This simplifies the trace for
// display in the timeline while still allowing users to see
// individual attempts when they expand the step details.
export function traceRollup(root: Trace): Trace {
  // Operate on a clone so we never mutate the caller's (cached) trace. The
  // rolled-up result is memoized against the trace object; without this, a
  // re-render would feed our own output back in and roll it up a second time
  // (which, e.g., collapses the multi-attempt "Function error" group into a
  // single-span group and relabels it — see rollupFinalization's size === 1
  // branch). The helpers below rename/reshape spans in place, which is safe
  // precisely because they only ever see this clone.
  root = structuredClone(root);

  const { stepOrder, steps, ungroupedFinalizations, finalizationAttempts, lastStep } =
    collectRollupGroups(root.childrenSpans ?? []);

  const rolledUpRunChildren: Trace[] = [];

  for (const stepID of stepOrder) {
    const attempts = steps.get(stepID);
    if (!attempts) {
      continue;
    }
    rolledUpRunChildren.push(rollupStepAttempts(stepID, attempts));
  }

  let finalization: Trace | null = null;
  if (finalizationAttempts) {
    finalization = rollupFinalization(finalizationAttempts, lastStep);
    rolledUpRunChildren.push(finalization);
  }

  // Emit ungrouped finalization spans only when they don't duplicate a
  // terminal failure already rendered as the finalization rollup (the single
  // "Finalization" or the multi-attempt "Function error") — that duplicate is
  // the server's relabeled discovery span mirroring the grouped failed
  // attempts (EXE-1992); the grouped attempts already carry the terminal
  // error output. A COMPLETED finalization is legitimate and left alone.
  if (finalization?.status !== 'FAILED') {
    rolledUpRunChildren.push(...ungroupedFinalizations);
  }

  const sortingKey = (trace: Trace) =>
    toMaybeDate(trace.queuedAt)?.getTime() ?? toMaybeDate(trace.startedAt)?.getTime() ?? 0;
  root.childrenSpans = rolledUpRunChildren.sort((a, b) => sortingKey(a) - sortingKey(b));

  return root;
}

/**
 * Convert a V3 Trace to V4 TimelineData
 */
export function traceToTimelineData(
  trace: Trace,
  options: {
    runID: string;
    orgName?: string;
    leftWidth?: number;
    functionSlug?: string;
  }
): TimelineData {
  const { orgName, leftWidth = TIMELINE_CONSTANTS.DEFAULT_LEFT_WIDTH, functionSlug } = options;

  // Calculate min/max time from the entire trace tree
  let minTime = new Date(trace.queuedAt);
  let maxTime = trace.endedAt ? new Date(trace.endedAt) : new Date();

  traceWalk(trace, (t) => {
    minTime = min([minTime, new Date(t.queuedAt)]);
    const endedAt = t.endedAt ? new Date(t.endedAt) : null;
    if (endedAt) {
      maxTime = max([endedAt, maxTime]);
    }
  });

  // Run startedAt timestamp for computing per-step discovery time
  const runStartedAtMs = trace.startedAt ? new Date(trace.startedAt).getTime() : null;

  // Every run starts with some platform delay before the first step. Drawn to
  // scale it is dead space at the front of every single trace, spent on
  // something that is the same on all of them and rarely what anyone came to
  // look at — so below a threshold the plot simply starts when the run started
  // and the delay is reported as text on the Run row instead.
  //
  // Above the threshold it stays on the axis, because then it is not background
  // noise but the story: `blocked` waited 6.3s behind a concurrency limit, and
  // that is the whole point of that run.
  const runQueueDelayMs =
    runStartedAtMs !== null ? Math.max(0, runStartedAtMs - minTime.getTime()) : 0;
  const drawQueueDelay = runQueueDelayMs >= NOTEWORTHY_QUEUE_DELAY_MS;
  if (!drawQueueDelay && runStartedAtMs !== null) {
    minTime = new Date(runStartedAtMs);
  }

  // Convert root trace (rename to "Run")
  // Ensure isRoot is set to true for the root bar so clicking it shows TopInfo
  // Pass root status so all bars share the same status-based coloring
  const rootBar = traceToBarData(
    { ...trace, name: 'Run', isRoot: true },
    orgName,
    trace.status,
    runStartedAtMs,
    functionSlug
  );

  // Give the Run bar a timingBreakdown matching the step-level inngest/execution split.
  // Sum execution time from all step children, and attribute the rest to Inngest overhead.
  // For children without a timingBreakdown (sleep, waitForEvent, invoke, etc.),
  // use their wall-clock duration as execution time so it isn't misattributed as overhead.
  if (rootBar.endTime) {
    const runDurationMs = rootBar.endTime.getTime() - rootBar.startTime.getTime();
    if (runDurationMs > 0) {
      let totalExecutionMs = 0;
      for (const child of rootBar.children ?? []) {
        if (child.timingBreakdown) {
          totalExecutionMs += child.timingBreakdown.executionMs;
        } else if (child.endTime) {
          totalExecutionMs += Math.max(0, child.endTime.getTime() - child.startTime.getTime());
        }
      }
      if (totalExecutionMs > 0) {
        const inngestMs = Math.max(0, runDurationMs - totalExecutionMs);
        rootBar.timingBreakdown = {
          inngestMs: inngestMs,
          executionMs: totalExecutionMs,
          totalMs: runDurationMs,
        };
      }
    }
  }

  // Compute run-level Inngest overhead: run queue delay + finalization
  if (rootBar.endTime) {
    const runQueuedAtMs = rootBar.startTime.getTime();
    const runEndedAtMs = rootBar.endTime.getTime();

    // Run queue delay: time from queued to first execution
    const runQueueDelayMs =
      runStartedAtMs !== null ? Math.max(0, runStartedAtMs - runQueuedAtMs) : 0;

    // Finalization: time after last step ended until run ended
    let lastStepEndedAtMs = 0;
    for (const child of rootBar.children ?? []) {
      if (child.endTime) {
        lastStepEndedAtMs = Math.max(lastStepEndedAtMs, child.endTime.getTime());
      }
    }
    const finalizationMs =
      lastStepEndedAtMs > 0 ? Math.max(0, runEndedAtMs - lastStepEndedAtMs) : 0;

    const totalMs = runQueueDelayMs + finalizationMs;
    if (totalMs > 0) {
      rootBar.runInngestBreakdown = {
        runQueueDelayMs,
        finalizationMs,
        totalMs,
      };
    }
  }

  // Include the root bar in the rendered bars so users can click it
  // to return to the TopInfo view (Input/Function Payload)
  const bars = [rootBar];

  return {
    minTime,
    maxTime,
    bars: markInterrupted(
      withPlanning(
        withWaitNotes(
          withRunNote(
            // When the queue delay is reported as text rather than drawn, the run
            // bar has to start where the plot does too — otherwise it extends off
            // the left edge and reports more time than the axis covers, which is
            // how `simple` ended up with a Run bar longer than the run. Its
            // duration then reads as time spent executing, and the note carries
            // the queued part: 497ms + "+156ms queued" rather than a bare 653ms.
            clampToRunStart(bars, drawQueueDelay ? null : minTime, !drawQueueDelay),
            runQueueDelayMs,
            !drawQueueDelay
          )
        ),
        trace.discoveries ?? null
      ),
      trace.endedAt ? new Date(trace.endedAt) : null
    ),
    leftWidth,
    orgName,
  };
}

/**
 * The request that planned each step, drawn where the gap actually is.
 *
 * `Planning` used to be a row of its own. It was the wrong shape three ways: the
 * eye had to travel to it and back to read one step, it left gaps unfilled
 * because it only showed requests that were not already steps, and it said
 * nothing about WHICH request produced which step — the interesting part,
 * because one request can produce several.
 *
 * The trace says exactly that. On `chains`, one discovery runs 102→180ms with
 * `plannedStepIDs` naming both `right-1` and `left-1`, and both of those steps
 * are queued at 181. On `wide`, one request at 104→188 names all twelve. So the
 * gap before a step is not unexplained latency: it is a specific request, and
 * the steps that share it are exactly the steps that ran in parallel.
 *
 * A request is drawn ONCE, on the first of the steps it planned, and tied down
 * to the others with a hairline at the moment it finished. Drawing a copy on
 * each row would say the opposite of the truth — that each step had its own
 * planning request — when the whole point is that one request produced all of
 * them. One bar, several steps leading off it.
 *
 * The step's own reported duration does NOT change. This is the distinction that
 * matters, and the reason an earlier attempt at this was reverted: the bar is
 * widened to show a real, attributed, differently-styled interval, and the
 * number beside the row is still the step's own. Nothing is inflated.
 */
function withPlanning(
  bars: TimelineBarData[],
  discoveries: RunDiscovery[] | null
): TimelineBarData[] {
  if (!discoveries?.length) return bars;

  const timed = discoveries
    .map((d) => ({
      startMs: Date.parse(d.startedAt ?? d.queuedAt),
      endMs: Date.parse(d.endedAt ?? d.startedAt ?? d.queuedAt),
      planned: d.plannedStepIDs ?? [],
      spanID: d.spanID,
    }))
    .filter((d) => Number.isFinite(d.startMs) && Number.isFinite(d.endMs) && d.planned.length);

  const nameOfStep = new Map<string, string>();
  /** Row order, so a request can be drawn on the first step it produced. */
  const order = new Map<string, number>();
  let seen = 0;
  const walkNames = (list: TimelineBarData[]) => {
    for (const bar of list) {
      if (bar.stepID) {
        nameOfStep.set(bar.stepID, bar.name);
        order.set(bar.stepID, seen++);
      }
      if (bar.children?.length) walkNames(bar.children);
    }
  };
  walkNames(bars);

  const walk = (list: TimelineBarData[]): TimelineBarData[] =>
    list.map((bar) => {
      const children = bar.children ? walk(bar.children) : undefined;
      if (bar.isRoot || !bar.stepID || !bar.endTime) return { ...bar, children };

      const queued = bar.startTime.getTime();

      // The request that planned this step and finished before it was queued.
      // Later requests naming the same step are the ones that REPORTED it,
      // which is a different thing and belongs to whatever followed.
      const planning = timed
        .filter((d) => d.planned.includes(bar.stepID!) && d.endMs <= queued + 1)
        .sort((a, b) => b.endMs - a.endMs)[0];

      if (!planning || planning.startMs >= queued) return { ...bar, children };

      // Only the FIRST step a request planned carries the bar. The rest record
      // that they came from it, so the view can draw the tie without repeating.
      const owner = planning.planned
        .map((stepID) => order.get(stepID))
        .filter((i): i is number => i !== undefined)
        .sort((a, b) => a - b)[0];

      if (owner === undefined || order.get(bar.stepID) !== owner) {
        return { ...bar, children, plannedBy: planning.spanID };
      }

      const siblings = planning.planned
        .map((stepID) => nameOfStep.get(stepID))
        .filter((n): n is string => Boolean(n));

      // The bar is longer than its number by exactly this, so this is the term
      // that reconciles them. Named here rather than in `withWaitNotes`, which
      // runs BEFORE the widening and so could only ever measure zero — which is
      // why `gnarly-foreign-async`'s `late` drew a bar across 91% of the plot,
      // labelled `1ms`, with its 249ms of planning appearing in no text at all.
      const planningMs = queued - planning.startMs;
      const note = [
        planningMs >= LEAD_IN_NOTE_MIN_MS ? `+${formatDuration(planningMs)} planning` : null,
        bar.note,
      ]
        .filter(Boolean)
        .join(' · ');

      return {
        ...bar,
        children,
        plannedBy: planning.spanID,
        startTime: new Date(planning.startMs),
        // The label keeps the step's own span.
        reportedMs: bar.endTime.getTime() - queued,
        note: note || undefined,
        planning: {
          startMs: planning.startMs,
          endMs: planning.endMs,
          steps: siblings,
          spanID: planning.spanID,
        },
      };
    });

  const withPlan = walk(bars);

  /**
   * Time no span accounts for, stated as exactly that.
   *
   * On `t19-parallel` the three parallel steps end at 186ms and `d` is not
   * queued until 321ms. Nothing in the payload covers the 135ms between: no
   * step, no discovery, no attempt. It is real time the run spent, and leaving
   * it blank is how the trace ends up looking full of unexplained holes.
   *
   * It is NOT labelled "processing". That would be an invention, and the whole
   * value of this view is that it does not invent. What is known is that
   * nothing was reported, so that is what it says.
   */
  const fill = (list: TimelineBarData[]): TimelineBarData[] => {
    let previousEnd: number | null = null;
    return list.map((bar) => {
      const children = bar.children ? fill(bar.children) : undefined;
      const start = bar.startTime.getTime();
      const gapMs = previousEnd === null ? 0 : start - previousEnd;
      const from = previousEnd;
      if (bar.endTime) previousEnd = Math.max(previousEnd ?? 0, bar.endTime.getTime());

      if (bar.isRoot || from === null || gapMs < UNACCOUNTED_MIN_MS) {
        return { ...bar, children };
      }

      // The metadata usually names it. `inngest.timing` reports a
      // `discoveryMs` per step, and on `t19-parallel` `d`'s is 135 — exactly
      // the gap before it. So this is not unaccounted time at all: it is the
      // discovery request, and the payload said so. Only where the two do not
      // agree is it left as time nothing reported.
      const reported = bar.inngestBreakdown?.discoveryMs ?? 0;
      const isDiscovery = reported > 0 && Math.abs(reported - gapMs) <= 2;

      // Named for the same reason the planning widening is: this moves the bar
      // left without moving the number beside it, and on `gnarly-foreign-async`
      // it moved `late` across 91% of the plot while the row still read `1ms`.
      // Wording follows the segment's own tooltip, so the row and the thing it
      // describes cannot use two names for one interval.
      const note = [`+${formatDuration(gapMs)} ${isDiscovery ? 'planning' : 'queued'}`, bar.note]
        .filter(Boolean)
        .join(' · ');

      return {
        ...bar,
        children,
        startTime: new Date(from),
        reportedMs: bar.reportedMs ?? (bar.endTime ? bar.endTime.getTime() - start : undefined),
        unaccounted: { startMs: from, endMs: start, kind: isDiscovery ? 'discovery' : 'unknown' },
        note,
      };
    });
  };

  return withPlan.map((bar) => (bar.isRoot ? { ...bar, children: fill(bar.children ?? []) } : bar));
}

/**
 * Below this a gap is scheduling noise between two adjacent spans, and drawing
 * it would put a sliver on almost every row for nothing.
 */
const UNACCOUNTED_MIN_MS = 20;

/**
 * A step that never finished, in a run that did.
 *
 * `calculateDuration` measures an unfinished span against *now*, which is right
 * for a run still in flight and a lie for one that ended. On `cancelled` the
 * open `waitForEvent` reported **29 minutes** on a run that was cancelled after
 * 35 seconds, and the number grew every time the page was opened.
 *
 * What is true: the wait ran until the run was cancelled, and then stopped
 * without completing. So the bar ends where the run ended, and the row says the
 * duration was cut short rather than presenting it as a measurement. This is
 * Terraform's `(known after apply)` rule — a token where the value would be,
 * never a silent gap and never a fabricated number.
 *
 * A run still in flight is left alone: there, measuring against now is correct.
 */
function markInterrupted(bars: TimelineBarData[], runEndedAt: Date | null): TimelineBarData[] {
  if (!runEndedAt) return bars;
  const end = runEndedAt.getTime();

  const walk = (list: TimelineBarData[]): TimelineBarData[] =>
    list.map((bar) => {
      const children = bar.children ? walk(bar.children) : undefined;
      if (bar.isRoot || bar.endTime) return { ...bar, children };
      return {
        ...bar,
        endTime: new Date(Math.max(end, bar.startTime.getTime())),
        interrupted: true,
        children,
      };
    });

  return walk(bars);
}

/**
 * Report the run's queue delay in words on the Run row.
 *
 * Shown whether or not it is drawn: when it is small the plot no longer spends
 * width on it, and the number is the only trace of it left; when it is large the
 * bar shows it too and the number names it.
 */
function withRunNote(
  bars: TimelineBarData[],
  queueDelayMs: number,
  /** True when the delay was reclaimed out of the bar and must be added back. */
  clamped: boolean
): TimelineBarData[] {
  if (queueDelayMs <= 0) return bars;
  return bars.map((bar) =>
    bar.isRoot
      ? {
          ...bar,
          // The note says whether the queue is ON TOP of the number beside it
          // or INSIDE it, because those are two different situations and it
          // used to use one wording for both.
          //
          // When the delay is small it is reclaimed out of the plot, so the row
          // reports execution and the queue is additional: `+120ms queued`.
          // When it is large enough to draw, the axis contains it and so does
          // the row's number, so saying `+6.299s queued` invited the reader to
          // add 6.3s to a total that already held it.
          note: clamped
            ? `+${formatDuration(queueDelayMs)} queued`
            : `incl. ${formatDuration(queueDelayMs)} queued`,
          // The clamp above may have shortened the bar so it fits the plot; the
          // reported number is the run's whole life either way, so the headline
          // means the same thing whether the queue was drawn or reclaimed.
          // Only when the bar was clamped. When the delay is DRAWN the span
          // already contains it, and adding it again would report 18.6s for a
          // 12.3s run.
          runTotalMs: bar.endTime
            ? bar.endTime.getTime() - bar.startTime.getTime() + (clamped ? queueDelayMs : 0)
            : undefined,
        }
      : bar
  );
}

/**
 * How long a step sat before it ran — the pale lead-in on its bar.
 *
 * This is the ONLY definition of it. The bar and its label both come through
 * here, because the two disagreeing is how the reader ends up with a row
 * headlined 104ms next to a canvas node headlined 32ms and no way to tell which
 * is lying.
 *
 * The timestamps beat the metadata: a step's own span is what the bar is drawn
 * across, so whatever is left of it after execution is the wait, whatever the
 * SDK reported.
 */
export function leadInMs(bar: TimelineBarData): number {
  if (!bar.endTime) return 0;

  // The STEP's span, not the bar's. A bar drawn wide enough to show the request
  // that planned it is longer than the step, and that extra belongs to the
  // planning segment rather than to this step's wait.
  const ownMs = bar.reportedMs ?? bar.endTime.getTime() - bar.startTime.getTime();

  // A sleep or a waitForEvent has NO lead-in, by construction: waiting is its
  // whole substance, so it draws one solid bar and both views measure it from
  // its queue. Reporting a lead-in for one would name a thing that is not
  // drawn, which is the mismatch this function exists to prevent.
  if (IDLE_STYLES.has(bar.style)) return 0;

  // A step.invoke has no timing breakdown, so it fell through both rules: the
  // canvas measured it from `startedAt` and the row from `queuedAt`, and
  // `call child` reported 520ms against the canvas's 519ms with nothing naming
  // the millisecond between them. Its own `delayMs` is exactly that wait.
  if (!bar.timingBreakdown) return Math.max(0, bar.delayMs ?? 0);

  const spanMs = ownMs;
  const { executionMs, inngestMs } = bar.timingBreakdown;
  if (spanMs <= 0) return Math.max(0, inngestMs);

  return Math.max(0, Math.min(spanMs, spanMs - executionMs));
}

/**
 * Any lead-in that is DRAWN gets named. The two thresholds are the same one.
 *
 * They were not, and the gap between them put a hole in the arithmetic: the bar
 * for `step`'s `second step` drew a 2ms ghost, the row's 7ms total included it,
 * and the note stayed silent — so a reader subtracting the named wait from the
 * total got 7ms against the canvas's 5ms, with nothing on screen accounting for
 * the difference. `invoke`'s `call child` was the same at 1ms.
 *
 * A millisecond named on a few extra rows is a smaller cost than a number that
 * does not reconcile, and it keeps the one rule this view is built on: nothing
 * is drawn that cannot be asked what it is.
 */
const LEAD_IN_NOTE_MIN_MS = 1;

/**
 * Name the lead-in on every step that draws a visible one.
 *
 * The canvas reports how long a step RAN and puts the delay on the edge into
 * it, which is the decomposition asked for. The trace row reports the step's
 * whole span. Both are true and they are different numbers, so the row now
 * carries the third number that reconciles them: `+72ms wait · 104ms` says the
 * 32ms the canvas shows is in here too, and says what the pale part is.
 */
function withWaitNotes(bars: TimelineBarData[]): TimelineBarData[] {
  const walk = (list: TimelineBarData[]): TimelineBarData[] =>
    list.map((bar) => {
      const children = bar.children ? walk(bar.children) : undefined;
      if (bar.isRoot || bar.note || !bar.endTime) return { ...bar, children };

      // Named exactly when it is drawn — `generateBarSegments` draws a lead-in
      // for any non-zero `leadInMs`, so this is the same condition and not a
      // second, looser one.
      const wait = leadInMs(bar);

      return wait >= LEAD_IN_NOTE_MIN_MS
        ? { ...bar, note: `+${formatDuration(wait)} wait`, children }
        : { ...bar, children };
    });

  return walk(bars);
}

/**
 * No step may be drawn as waiting before the run itself started.
 *
 * Spans record a step's `queuedAt` as the moment the *run* was queued, so on
 * `failure` the first step reports `queuedAt` at +1ms while the run did not
 * start until +156ms. Drawn literally, the step's bar began at the very left of
 * the plot and overlapped the run's own Inngest row, which covers exactly that
 * 156ms — the same delay counted twice, in two places, contradicting each other.
 * A reader sees a step apparently running before anything had started.
 *
 * That 156ms is real and belongs to the run, which reports it on its own row.
 * Before the run started, no individual step was waiting on anything: it had not
 * been discovered yet. So a step's bar begins no earlier than the run does.
 *
 * The root bar is left alone — the run's queue delay is precisely what it is
 * there to show.
 */
function clampToRunStart(
  bars: TimelineBarData[],
  runStartedAt: Date | null,
  includeRoot = false
): TimelineBarData[] {
  if (!runStartedAt) return bars;
  const floor = runStartedAt.getTime();

  const clamp = (list: TimelineBarData[]): TimelineBarData[] =>
    list.map((bar) => {
      const children = bar.children ? clamp(bar.children) : undefined;
      if ((bar.isRoot && !includeRoot) || bar.startTime.getTime() >= floor) {
        return { ...bar, children };
      }

      // Never past its own end: a step that finished before the run was marked
      // started would otherwise be drawn backwards.
      const end = bar.endTime?.getTime();
      const start = end !== undefined ? Math.min(floor, end) : floor;

      // `delayMs` is measured from the ORIGINAL start, and callers reconstruct
      // "when this actually began executing" as `startTime + delayMs`. Moving
      // the start without shrinking the delay by the same amount would push that
      // reconstruction forward by however much was clamped — which is how the
      // retry segments first came out 5% too wide.
      const shift = start - bar.startTime.getTime();
      const delayMs = bar.delayMs !== undefined ? Math.max(0, bar.delayMs - shift) : bar.delayMs;

      return { ...bar, startTime: new Date(start), delayMs, children };
    });

  return clamp(bars);
}
