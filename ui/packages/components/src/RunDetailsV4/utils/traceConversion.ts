/**
 * Utilities to convert V3 Trace data to V4 TimelineData format.
 * Feature: 001-composable-timeline-bar
 */

import { max, min } from 'date-fns';

import type { BarStyleKey, TimelineBarData, TimelineData } from '../TimelineBar.types';
import { traceWalk } from '../runDetailsUtils';
import type { Trace } from '../types';

/**
 * Check if a trace represents a step.run span
 */
function isStepRunSpan(trace: Trace): boolean {
  return (
    trace.stepOp === 'RUN' ||
    trace.stepType === 'RUN' ||
    (trace.stepInfo !== null &&
      typeof trace.stepInfo === 'object' &&
      'type' in (trace.stepInfo as object))
  );
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
 * Convert a single Trace to TimelineBarData
 */
function traceToBarData(trace: Trace, orgName?: string): TimelineBarData {
  const isStepRun = isStepRunSpan(trace) && !trace.isUserland;
  const timingBreakdown = isStepRun ? calculateTimingBreakdown(trace) : undefined;

  return {
    id: trace.spanID,
    name: getSpanName(trace.name),
    startTime: new Date(trace.queuedAt),
    endTime: trace.endedAt ? new Date(trace.endedAt) : null,
    style: getStyleForTrace(trace),
    children: trace.childrenSpans?.map((child) => traceToBarData(child, orgName)),
    timingBreakdown,
    isRoot: trace.isRoot,
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
  const { orgName, leftWidth = 30 } = options;

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

  // Convert root trace (rename to "Run")
  // Ensure isRoot is set to true for the root bar so clicking it shows TopInfo
  const rootBar = traceToBarData({ ...trace, name: 'Run', isRoot: true }, orgName);

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

/**
 * Re-export types that may be needed by consumers
 */
export type { Trace };
