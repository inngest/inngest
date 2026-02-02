/**
 * Timing breakdown utilities for step.run span visualization
 * Feature: EXE-1217
 */ import {
  isStepInfoRun,
  type SpanTimingBreakdown,
  type TimingCategory,
  type TimingCategoryTotal,
  type Trace,
} from './types';

// ============================================================================
// Constants
// ============================================================================

/**
 * Color configuration for timing categories
 */
export const TIMING_COLORS = {
  inngest: {
    base: 'bg-slate-400',
    queue: 'bg-slate-300',
    concurrency_delay: 'bg-slate-400',
    processing: 'bg-slate-500',
  },
  connecting: {
    base: 'bg-amber-400',
    request: 'bg-amber-300',
    handshake: 'bg-amber-400',
  },
  customer_server: {
    base: 'bg-emerald-500',
    middleware: 'bg-emerald-300',
    running: 'bg-emerald-500',
    db_query: 'bg-emerald-400',
    checkpointing: 'bg-emerald-500',
  },
} as const;

/**
 * Icon mapping for categories
 */
export const CATEGORY_ICONS = {
  inngest: 'gear',
  connecting: 'lightning',
  customer_server: 'building',
} as const;

// ============================================================================
// Type Guards
// ============================================================================

/**
 * Check if a trace represents a step.run span
 * @param trace - The trace to check
 * @returns true if this is a step.run span
 */
export function isStepRunSpan(trace: Trace): boolean {
  return (
    trace.stepOp === 'StepRun' ||
    trace.stepType === 'RUN' ||
    (trace.stepInfo !== null && isStepInfoRun(trace.stepInfo))
  );
}

// ============================================================================
// Utilities
// ============================================================================

/**
 * Format duration in human-readable form (FR-011)
 * @param ms - Duration in milliseconds
 * @returns Formatted string
 */
export function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms < 1000) return `${ms.toFixed(2)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}

/**
 * Calculate timing breakdown from trace data
 * @param trace - The trace/span data with timestamps
 * @returns SpanTimingBreakdown or null if insufficient data
 */
export function calculateTimingBreakdown(trace: Trace): SpanTimingBreakdown | null {
  if (!trace.queuedAt) return null;

  const queuedAt = new Date(trace.queuedAt).getTime();
  const startedAt = trace.startedAt ? new Date(trace.startedAt).getTime() : null;
  const endedAt = trace.endedAt ? new Date(trace.endedAt).getTime() : Date.now();

  // Calculate durations (FR-002, FR-003)
  const queueDelay = startedAt ? Math.max(0, startedAt - queuedAt) : 0;
  const executionTime = startedAt ? Math.max(0, endedAt - startedAt) : 0;
  const totalDurationMs = startedAt ? queueDelay + executionTime : Date.now() - queuedAt;

  // Calculate percentages for bar segments
  const inngestPercent = totalDurationMs > 0 ? (queueDelay / totalDurationMs) * 100 : 100;
  const serverPercent = 100 - inngestPercent;

  // Build categories array
  const categories: TimingCategoryTotal[] = [
    {
      category: 'inngest',
      label: 'INNGEST',
      icon: 'gear',
      totalMs: queueDelay,
      segments: [
        {
          category: 'inngest',
          segmentType: 'queue',
          label: 'Queue',
          durationMs: queueDelay,
          color: TIMING_COLORS.inngest.queue,
        },
      ],
    },
  ];

  // Add customer_server category only if execution has started (FR-012)
  if (startedAt) {
    categories.push({
      category: 'customer_server',
      label: 'YOUR SERVER',
      icon: 'building',
      totalMs: executionTime,
      segments: [
        {
          category: 'customer_server',
          segmentType: 'running',
          label: 'Running',
          durationMs: executionTime,
          color: TIMING_COLORS.customer_server.running,
        },
      ],
    });
  }

  // Build bar segments
  const barSegments: SpanTimingBreakdown['barSegments'] = [
    { category: 'inngest', widthPercent: inngestPercent },
  ];

  if (startedAt) {
    barSegments.push({ category: 'customer_server', widthPercent: serverPercent });
  }

  return {
    totalDurationMs,
    categories,
    barSegments,
    queuedAt,
    startedAt,
    endedAt,
    startTime: new Date(queuedAt).toLocaleString(),
    endTime: trace.endedAt ? new Date(endedAt).toLocaleString() : '—',
  };
}
