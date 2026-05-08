/**
 * Utilities to convert V3 Trace data to V4 TimelineData format.
 * Feature: 001-composable-timeline-bar
 */

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
  runStartedAtMs: number | null
): InngestBreakdownData | null {
  if (!trace.queuedAt) return null;

  // Discovery: time from when the run started executing to when this step was queued
  const stepQueuedAt = new Date(trace.queuedAt).getTime();
  const discoveryMs = runStartedAtMs !== null ? Math.max(0, stepQueuedAt - runStartedAtMs) : 0;

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
  runStartedAtMs?: number | null,
  functionSlug?: string
): TimelineBarData {
  const isStepRun = isStepRunSpan(trace) && !trace.isUserland;
  // Prefer server-computed timing from metadata, fall back to span-timestamp calculation
  let timingBreakdown = isStepRun
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
  const inngestBreakdown = isStepRun
    ? getInngestBreakdown(trace, runStartedAtMs ?? null) ?? undefined
    : undefined;

  // Check if this step has experiment metadata
  const hasExperiment = trace.metadata?.some((m) => m.kind === KindInngestExperiment) ?? false;

  // Extract experiment metadata for hover card display
  const experimentMd = trace.metadata?.find(isExperimentMetadata);
  const experimentMetadata = experimentMd
    ? {
        experimentName: experimentMd.values.experiment_name,
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
    children: trace.childrenSpans?.map((child) =>
      traceToBarData(child, orgName, rootStatus, runStartedAtMs, functionSlug)
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
