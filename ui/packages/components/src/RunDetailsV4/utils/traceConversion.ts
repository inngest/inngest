/**
 * Utilities to convert V3 Trace data to V4 TimelineData format.
 * Feature: 001-composable-timeline-bar
 */

import { max, min } from 'date-fns';

import type {
  BarStyleKey,
  HTTPTimingBreakdownData,
  InngestBreakdownData,
  TimelineBarData,
  TimelineData,
} from '../TimelineBar.types';
import { traceWalk } from '../runDetailsUtils';
import {
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
): { queueMs: number; executionMs: number; totalMs: number } | undefined {
  if (!trace.queuedAt) return undefined;

  const queuedAt = new Date(trace.queuedAt).getTime();
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : Date.now();

  // Calculate durations
  const queueMs = startedAt
    ? Math.max(0, startedAt - queuedAt)
    : Math.max(0, Date.now() - queuedAt);
  const executionMs = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalMs = startedAt ? queueMs + executionMs : Date.now() - queuedAt;

  return { queueMs, executionMs, totalMs };
}

/**
 * Extract timing breakdown from inngest.timing metadata.
 * Returns server-computed queue delay and Inngest overhead when available.
 */
function getTimingFromMetadata(
  trace: Trace,
  metadata?: SpanMetadata[]
): { queueMs: number; executionMs: number; totalMs: number } | null {
  if (!metadata) return null;

  const timing = metadata.find((m): m is SpanMetadataInngestTiming => m.kind === 'inngest.timing');

  if (!timing) return null;

  const queueMs = timing.values.total_inngest_ms ?? 0;

  // Execution time is still derived from span timestamps since the metadata
  // captures Inngest-side overhead, not SDK execution duration.
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : Date.now();
  const executionMs = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalMs = queueMs + executionMs;

  return { queueMs, executionMs, totalMs };
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
 * Root timing context passed to child bars so steps can span the full run lifecycle.
 */
type RootTiming = {
  startTime: Date;
  startedAt: Date | null;
  endTime: Date | null;
};

/**
 * Convert a single Trace to TimelineBarData
 */
function traceToBarData(
  trace: Trace,
  orgName?: string,
  rootStatus?: string,
  rootTiming?: RootTiming
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

  // For steps with timing data and a known root time range, widen the step bar to
  // span the full run lifecycle. The Inngest (queue) portion then includes all
  // non-execution overhead: discovery calls, queue delay, scheduling, and finalization.
  let effectiveStartTime = new Date(trace.queuedAt);
  let effectiveEndTime: Date | null = trace.endedAt ? new Date(trace.endedAt) : null;
  let inngestBreakdown: InngestBreakdownData | undefined;

  if (isStepRun && timingBreakdown && rootTiming?.endTime) {
    const runDurationMs = rootTiming.endTime.getTime() - rootTiming.startTime.getTime();
    if (runDurationMs > 0) {
      const executionMs = timingBreakdown.executionMs;
      const inngestMs = Math.max(0, runDurationMs - executionMs);
      timingBreakdown = { queueMs: inngestMs, executionMs, totalMs: runDurationMs };
      effectiveStartTime = rootTiming.startTime;
      effectiveEndTime = rootTiming.endTime;

      // Break down the Inngest overhead into distinct phases
      const runStartedAt = rootTiming.startedAt?.getTime() ?? rootTiming.startTime.getTime();
      const stepQueuedAt = new Date(trace.queuedAt).getTime();
      const stepEndedAt = trace.endedAt
        ? new Date(trace.endedAt).getTime()
        : rootTiming.endTime.getTime();

      const runQueueDelayMs = Math.max(0, runStartedAt - rootTiming.startTime.getTime());
      const discoveryMs = Math.max(0, stepQueuedAt - runStartedAt);
      const finalizationMs = Math.max(0, rootTiming.endTime.getTime() - stepEndedAt);

      if (inngestMs > 0) {
        inngestBreakdown = {
          runQueueDelayMs,
          discoveryMs,
          finalizationMs,
          totalMs: inngestMs,
        };
      }
    }
  }

  return {
    id: trace.spanID,
    name: getSpanName(trace.name),
    startTime: effectiveStartTime,
    endTime: effectiveEndTime,
    style: getStyleForTrace(trace),
    children: trace.childrenSpans?.map((child) =>
      traceToBarData(child, orgName, rootStatus, rootTiming)
    ),
    timingBreakdown,
    httpTimingBreakdown,
    inngestBreakdown,
    isRoot: trace.isRoot,
    status,
    delayMs,
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
  }
): TimelineData {
  const { orgName, leftWidth = TIMELINE_CONSTANTS.DEFAULT_LEFT_WIDTH } = options;

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

  // Root timing context for step bars to span the full run lifecycle
  const rootTiming: RootTiming = {
    startTime: new Date(trace.queuedAt),
    startedAt: trace.startedAt ? new Date(trace.startedAt) : null,
    endTime: trace.endedAt ? new Date(trace.endedAt) : null,
  };

  // Convert root trace (rename to "Run")
  // Ensure isRoot is set to true for the root bar so clicking it shows TopInfo
  // Pass root status so all bars share the same status-based coloring
  const rootBar = traceToBarData(
    { ...trace, name: 'Run', isRoot: true },
    orgName,
    trace.status,
    rootTiming
  );

  // Give the Run bar a timingBreakdown matching the step-level inngest/execution split.
  // Sum execution time from all step children, and attribute the rest to Inngest overhead.
  if (rootBar.endTime) {
    const runDurationMs = rootBar.endTime.getTime() - rootBar.startTime.getTime();
    if (runDurationMs > 0) {
      let totalExecutionMs = 0;
      for (const child of rootBar.children ?? []) {
        if (child.timingBreakdown) {
          totalExecutionMs += child.timingBreakdown.executionMs;
        }
      }
      if (totalExecutionMs > 0) {
        const inngestMs = Math.max(0, runDurationMs - totalExecutionMs);
        rootBar.timingBreakdown = {
          queueMs: inngestMs,
          executionMs: totalExecutionMs,
          totalMs: runDurationMs,
        };
      }
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
