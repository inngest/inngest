/**
 * Utilities to convert V3 Trace data to V4 TimelineData format.
 * Feature: 001-composable-timeline-bar
 */

import { maxDateString, toMaybeDate } from '@inngest/components/utils/date';
import { max, min } from 'date-fns';

import { KindInngestExperiment } from '../../generated';
import type {
  BarStyleKey,
  HTTPTimingBreakdownData,
  InngestBreakdownData,
  TimelineBarData,
  TimelineData,
} from '../TimelineBar.types';
import { traceWalk } from '../runDetailsUtils';
import {
  isExperimentMetadata,
  isStepInfoRun,
  type SpanMetadata,
  type SpanMetadataInngestHTTPTiming,
  type SpanMetadataInngestTiming,
  type Trace,
} from '../types';
import { TIMELINE_CONSTANTS } from './timing';

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

  return bars;
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

  return {
    id: trace.spanID,
    name: getSpanName(trace.name),
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
  /** spans emitted unchanged (output spans with no stepID or groupID) */
  passthrough: Trace[];
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
 */
function collectRollupGroups(children: Trace[]): RollupGroups {
  const stepOrder: string[] = [];
  const steps = new Map<string, Map<number, Trace>>();
  const passthrough: Trace[] = [];
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
        passthrough.push(child);
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

  return { stepOrder, steps, passthrough, finalizationAttempts, lastStep };
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

  const { stepOrder, steps, passthrough, finalizationAttempts, lastStep } = collectRollupGroups(
    root.childrenSpans ?? []
  );

  const rolledUpRunChildren: Trace[] = [...passthrough];

  for (const stepID of stepOrder) {
    const attempts = steps.get(stepID);
    if (!attempts) {
      continue;
    }
    rolledUpRunChildren.push(rollupStepAttempts(stepID, attempts));
  }

  if (finalizationAttempts) {
    rolledUpRunChildren.push(rollupFinalization(finalizationAttempts, lastStep));
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
    bars,
    leftWidth,
    orgName,
  };
}
