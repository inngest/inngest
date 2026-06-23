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
 * Calculate timing breakdown for a step.run span.
 *
 * For in-progress spans (no `endedAt`), `executionMs`/`totalMs` are returned
 * as `null`. The render path must recompute them live against `Date.now()`
 * — baking a `Date.now()` snapshot here would freeze inside the upstream
 * `useMemo` and visually misallocate width to the Inngest segment.
 */
function calculateTimingBreakdown(
  trace: Trace
): { inngestMs: number; executionMs: number | null; totalMs: number | null } | undefined {
  if (!trace.queuedAt) return undefined;

  const queuedAt = new Date(trace.queuedAt).getTime();
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : null;

  // Inngest-side overhead is fixed once `startedAt` exists; without `startedAt`
  // we cannot bake a value (would drift with Date.now), so leave it as 0 and
  // let the render path treat the whole bar as in-progress.
  const inngestMs = startedAt != null ? Math.max(0, startedAt - queuedAt) : 0;

  if (endedAt == null) {
    return { inngestMs, executionMs: null, totalMs: null };
  }

  const executionMs = startedAt != null ? Math.max(0, endedAt - startedAt) : 0;
  return { inngestMs, executionMs, totalMs: inngestMs + executionMs };
}

/**
 * Extract timing breakdown from inngest.timing metadata.
 * Returns server-computed queue delay and Inngest overhead when available.
 */
function getTimingFromMetadata(
  trace: Trace,
  metadata?: SpanMetadata[]
): { inngestMs: number; executionMs: number | null; totalMs: number | null } | null {
  if (!metadata) return null;

  const timing = metadata.find((m): m is SpanMetadataInngestTiming => m.kind === 'inngest.timing');

  if (!timing) return null;

  const inngestMs = timing.values.total_inngest_ms ?? 0;

  // Execution time is derived from span timestamps. For in-progress spans
  // we return null so the render path computes it live against Date.now().
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : null;

  if (endedAt == null) {
    return { inngestMs, executionMs: null, totalMs: null };
  }

  const executionMs = startedAt != null ? Math.max(0, endedAt - startedAt) : 0;
  return { inngestMs, executionMs, totalMs: inngestMs + executionMs };
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

// traceRollup groups attempts by stepID and groupID
// (for spans without stepIDs, typically finalization spans or failures hitting the SDK entirely)
// and creates virtual "rollup" spans that represent the entire step/finalization
// with all attempts grouped together. This simplifies the trace for
// display in the timeline while still allowing users to see
// individual attempts when they expand the step details.
export function traceRollup(root: Trace): Trace {
  const stepOrder: string[] = [];
  const steps = new Map<string, Map<number, Trace>>();
  const rolledUpRunChildren: Trace[] = [];
  const groupedSpans = new Map<string, Map<number, Trace>>();
  let lastStepEndedAt: Date | null = null;
  let lastStep: Trace | null = null;
  let finalSpan: Trace | null = null;
  for (const child of root.childrenSpans ?? []) {
    if (child.outputID && !child.stepID) {
      if (child.groupID) {
        finalSpan = child;
        const attempts = groupedSpans.get(child.groupID) ?? new Map<number, Trace>();
        attempts.set(child.attempts ?? 0, child);
        groupedSpans.set(child.groupID, attempts);
      } else {
        rolledUpRunChildren.push(child);
      }
      continue;
    } else if (!child.stepID || child.attempts === null) {
      continue;
    }

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

  for (const stepID of stepOrder) {
    const attempts = steps.get(stepID);
    if (!attempts) {
      continue;
    }

    const minAttempt = Math.min(...attempts.keys());
    const minAttemptTrace = attempts.get(minAttempt) as Trace;
    const maxAttempt = Math.max(...attempts.keys());
    const maxAttemptTrace = attempts.get(maxAttempt) as Trace;
    if (attempts.size == 1) {
      rolledUpRunChildren.push(maxAttemptTrace);
      continue;
    }

    // Create a virtual span to represent the step as a whole with all attempts
    const virtualSpan: Trace = {
      isRoot: false,
      isUserland: false,
      spanID: `${stepID}-rollup`, // virtual span
      groupID: maxAttemptTrace.groupID,
      name: maxAttemptTrace.name,
      attempts: maxAttempt,
      stepID: stepID,
      queuedAt: minAttemptTrace.queuedAt,
      scheduledAt: minAttemptTrace.scheduledAt,
      startedAt: minAttemptTrace.startedAt,
      endedAt: maxAttemptTrace.endedAt,
      status: maxAttemptTrace.status,
      outputID: maxAttemptTrace.outputID,
      debugRunID: maxAttemptTrace.debugRunID,
      debugSessionID: maxAttemptTrace.debugSessionID,
      stepInfo: maxAttemptTrace.stepInfo,

      childrenSpans: Array.from(attempts.values()).sort(
        (a, b) => (a.attempts ?? 0) - (b.attempts ?? 0)
      ),

      userlandSpan: null,
    };

    for (const attempt of attempts.values()) {
      attempt.name = `Attempt ${attempt.attempts}`;
    }

    rolledUpRunChildren.push(virtualSpan);
  }

  if (finalSpan && finalSpan.groupID && groupedSpans.has(finalSpan.groupID)) {
    const attempts = groupedSpans.get(finalSpan.groupID) as Map<number, Trace>;
    const minAttempt = Math.min(...attempts.keys());
    const minAttemptTrace = attempts.get(minAttempt) as Trace;
    const maxAttempt = Math.max(...attempts.keys());
    const maxAttemptTrace = attempts.get(maxAttempt) as Trace;
    if (attempts.size == 1) {
      maxAttemptTrace.name = 'Finalization';
      maxAttemptTrace.queuedAt = maxDateString(maxAttemptTrace.queuedAt, lastStep?.endedAt);
      maxAttemptTrace.scheduledAt = maxDateString(maxAttemptTrace.scheduledAt, lastStep?.endedAt);
      maxAttemptTrace.startedAt = maxDateString(maxAttemptTrace.startedAt, lastStep?.endedAt);
      rolledUpRunChildren.push(maxAttemptTrace);
    } else {
      // Create a virtual span to represent the finalization as a whole with all attempts
      const virtualSpan: Trace = {
        isRoot: false,
        isUserland: false,
        name: 'Finalization',
        spanID: `final-rollup`, // virtual span
        groupID: maxAttemptTrace.groupID,
        attempts: maxAttemptTrace.attempts,
        queuedAt: maxDateString(minAttemptTrace.queuedAt, lastStep?.endedAt),
        scheduledAt: maxDateString(minAttemptTrace.scheduledAt, lastStep?.endedAt),
        startedAt: maxDateString(minAttemptTrace.startedAt, lastStep?.endedAt),
        endedAt: maxAttemptTrace.endedAt,
        status: maxAttemptTrace.status,
        outputID: maxAttemptTrace.outputID,
        debugRunID: maxAttemptTrace.debugRunID,
        debugSessionID: maxAttemptTrace.debugSessionID,

        childrenSpans: Array.from(attempts.values()).sort(
          (a, b) => (a.attempts ?? 0) - (b.attempts ?? 0)
        ),

        stepInfo: null,
        userlandSpan: null,
      };

      for (const attempt of attempts.values()) {
        attempt.name = `Attempt ${attempt.attempts}`;
      }

      rolledUpRunChildren.push(virtualSpan);
    }
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

  // Calculate min/max time from the entire trace tree. `maxTime` stays null
  // while the run is in progress so callers substitute a live `Date.now()`
  // — baking a snapshot here would freeze inside the upstream useMemo.
  let minTime = new Date(trace.queuedAt);
  let maxTime: Date | null = trace.endedAt ? new Date(trace.endedAt) : null;
  let anyInProgress = !trace.endedAt;

  traceWalk(trace, (t) => {
    minTime = min([minTime, new Date(t.queuedAt)]);
    const endedAt = t.endedAt ? new Date(t.endedAt) : null;
    if (endedAt) {
      maxTime = maxTime ? max([endedAt, maxTime]) : endedAt;
    } else {
      anyInProgress = true;
    }
  });

  if (anyInProgress) {
    maxTime = null;
  }

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
  // Only roll up the root bar's timingBreakdown for fully-completed runs.
  // While any child is in-progress, both the run duration and child execution
  // sums are live (Date.now()-dependent); baking them here would freeze
  // inside useMemo. The renderer handles the in-progress root via segments.
  if (rootBar.endTime && !anyInProgress) {
    const runDurationMs = rootBar.endTime.getTime() - rootBar.startTime.getTime();
    if (runDurationMs > 0) {
      let totalExecutionMs = 0;
      for (const child of rootBar.children ?? []) {
        if (child.timingBreakdown?.executionMs != null) {
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
