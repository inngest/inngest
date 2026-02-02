import assert from 'node:assert';
import { describe, it } from 'vitest';

import {
  TIMING_COLORS,
  calculateTimingBreakdown,
  formatDuration,
  isStepRunSpan,
} from './timingBreakdown';
import type { Trace } from './types';

// ============================================================================
// Test Fixtures
// ============================================================================

const createMockTrace = (overrides: Partial<Trace> = {}): Trace => ({
  attempts: 1,
  endedAt: '2024-03-15T09:12:04.000Z',
  isRoot: false,
  name: 'step.run',
  outputID: 'output-123',
  queuedAt: '2024-03-15T09:12:00.000Z',
  spanID: 'span-123',
  startedAt: '2024-03-15T09:12:02.000Z',
  status: 'completed',
  stepInfo: { type: 'run' },
  stepOp: 'StepRun',
  stepType: 'RUN',
  userlandSpan: null,
  isUserland: false,
  ...overrides,
});

// ============================================================================
// T014: calculateTimingBreakdown with complete trace
// ============================================================================

describe('calculateTimingBreakdown', () => {
  it('calculates correct breakdown for complete trace', () => {
    const trace = createMockTrace({
      queuedAt: '2024-03-15T09:12:00.000Z',
      startedAt: '2024-03-15T09:12:02.000Z',
      endedAt: '2024-03-15T09:12:04.000Z',
    });

    const breakdown = calculateTimingBreakdown(trace);

    assert.ok(breakdown !== null, 'breakdown should not be null');
    assert.strictEqual(breakdown.totalDurationMs, 4000, 'total duration should be 4000ms');

    // Check categories
    assert.strictEqual(breakdown.categories.length, 2, 'should have 2 categories');

    // INNGEST category (queue delay: 2000ms)
    const inngestCategory = breakdown.categories[0];
    assert.strictEqual(inngestCategory?.category, 'inngest');
    assert.strictEqual(inngestCategory?.label, 'INNGEST');
    assert.strictEqual(inngestCategory?.icon, 'gear');
    assert.strictEqual(inngestCategory?.totalMs, 2000);
    assert.strictEqual(inngestCategory?.segments[0]?.segmentType, 'queue');
    assert.strictEqual(inngestCategory?.segments[0]?.durationMs, 2000);
    assert.strictEqual(inngestCategory?.segments[0]?.color, TIMING_COLORS.inngest.queue);

    // Customer server category (execution: 2000ms)
    const serverCategory = breakdown.categories[1];
    assert.strictEqual(serverCategory?.category, 'customer_server');
    assert.strictEqual(serverCategory?.label, 'YOUR SERVER');
    assert.strictEqual(serverCategory?.icon, 'building');
    assert.strictEqual(serverCategory?.totalMs, 2000);
    assert.strictEqual(serverCategory?.segments[0]?.segmentType, 'running');
    assert.strictEqual(serverCategory?.segments[0]?.durationMs, 2000);

    // Bar segments (50/50 split)
    assert.strictEqual(breakdown.barSegments.length, 2);
    assert.strictEqual(breakdown.barSegments[0]?.category, 'inngest');
    assert.strictEqual(breakdown.barSegments[0]?.widthPercent, 50);
    assert.strictEqual(breakdown.barSegments[1]?.category, 'customer_server');
    assert.strictEqual(breakdown.barSegments[1]?.widthPercent, 50);
  });

  it('returns null when queuedAt is missing', () => {
    const trace = createMockTrace({
      queuedAt: '',
    });

    const breakdown = calculateTimingBreakdown(trace);
    assert.strictEqual(breakdown, null);
  });

  // ============================================================================
  // T015: calculateTimingBreakdown with in-progress span (no endedAt)
  // ============================================================================

  it('handles in-progress span (no endedAt) by using current time', () => {
    const now = Date.now();
    const queuedTime = new Date(now - 5000).toISOString(); // 5 seconds ago
    const startedTime = new Date(now - 3000).toISOString(); // 3 seconds ago

    const trace = createMockTrace({
      queuedAt: queuedTime,
      startedAt: startedTime,
      endedAt: null,
    });

    const breakdown = calculateTimingBreakdown(trace);

    assert.ok(breakdown !== null);
    // Queue delay should be ~2000ms (startedAt - queuedAt)
    assert.ok(breakdown.categories[0]?.totalMs >= 1900 && breakdown.categories[0]?.totalMs <= 2100);
    // Execution time should be ~3000ms (now - startedAt)
    assert.ok(breakdown.categories[1]?.totalMs >= 2900 && breakdown.categories[1]?.totalMs <= 3500);
  });

  it('handles queued-only span (no startedAt) with only inngest category', () => {
    const now = Date.now();
    const queuedTime = new Date(now - 5000).toISOString(); // 5 seconds ago

    const trace = createMockTrace({
      queuedAt: queuedTime,
      startedAt: null,
      endedAt: null,
    });

    const breakdown = calculateTimingBreakdown(trace);

    assert.ok(breakdown !== null);
    // Only inngest category should exist (FR-012)
    assert.strictEqual(breakdown.categories.length, 1);
    assert.strictEqual(breakdown.categories[0]?.category, 'inngest');
    // Queue time should be ~5000ms (elapsed time since queuedAt)
    assert.ok(breakdown.categories[0]?.totalMs >= 4900 && breakdown.categories[0]?.totalMs <= 5500);

    // Only one bar segment
    assert.strictEqual(breakdown.barSegments.length, 1);
    assert.strictEqual(breakdown.barSegments[0]?.category, 'inngest');
    assert.strictEqual(breakdown.barSegments[0]?.widthPercent, 100);
  });

  it('handles zero queue delay correctly', () => {
    const trace = createMockTrace({
      queuedAt: '2024-03-15T09:12:00.000Z',
      startedAt: '2024-03-15T09:12:00.000Z', // Same as queuedAt
      endedAt: '2024-03-15T09:12:02.000Z',
    });

    const breakdown = calculateTimingBreakdown(trace);

    assert.ok(breakdown !== null);
    // Queue delay should be 0
    assert.strictEqual(breakdown.categories[0]?.totalMs, 0);
    // Execution should be 2000ms
    assert.strictEqual(breakdown.categories[1]?.totalMs, 2000);
    // Bar segments: inngest should be 0%, server 100%
    assert.strictEqual(breakdown.barSegments[0]?.widthPercent, 0);
    assert.strictEqual(breakdown.barSegments[1]?.widthPercent, 100);
  });
});

// ============================================================================
// T016: formatDuration edge cases
// ============================================================================

describe('formatDuration', () => {
  it('formats sub-millisecond durations as <1ms', () => {
    assert.strictEqual(formatDuration(0), '<1ms');
    assert.strictEqual(formatDuration(0.5), '<1ms');
    assert.strictEqual(formatDuration(0.99), '<1ms');
  });

  it('formats milliseconds with 2 decimal places', () => {
    assert.strictEqual(formatDuration(1), '1.00ms');
    assert.strictEqual(formatDuration(100), '100.00ms');
    assert.strictEqual(formatDuration(999.99), '999.99ms');
  });

  it('formats seconds with 2 decimal places', () => {
    assert.strictEqual(formatDuration(1000), '1.00s');
    assert.strictEqual(formatDuration(1500), '1.50s');
    // 59999ms = 59.999s which rounds to 60.00s with toFixed(2)
    assert.strictEqual(formatDuration(59999), '60.00s');
  });

  it('formats minutes with 2 decimal places', () => {
    assert.strictEqual(formatDuration(60000), '1.00m');
    assert.strictEqual(formatDuration(90000), '1.50m');
    assert.strictEqual(formatDuration(300000), '5.00m');
  });
});

// ============================================================================
// T017: isStepRunSpan detection
// ============================================================================

describe('isStepRunSpan', () => {
  it('returns true when stepOp is StepRun', () => {
    const trace = createMockTrace({
      stepOp: 'StepRun',
      stepType: null,
      stepInfo: null,
    });
    assert.strictEqual(isStepRunSpan(trace), true);
  });

  it('returns true when stepType is RUN', () => {
    const trace = createMockTrace({
      stepOp: null,
      stepType: 'RUN',
      stepInfo: null,
    });
    assert.strictEqual(isStepRunSpan(trace), true);
  });

  it('returns true when stepInfo is StepInfoRun (has type property)', () => {
    const trace = createMockTrace({
      stepOp: null,
      stepType: null,
      stepInfo: { type: 'run' },
    });
    assert.strictEqual(isStepRunSpan(trace), true);
  });

  // T034: Non-step.run spans (US5 - backwards compatibility)
  it('returns false for waitForEvent span', () => {
    const trace = createMockTrace({
      stepOp: 'StepWaitForEvent',
      stepType: 'WAIT',
      stepInfo: {
        eventName: 'test/event',
        expression: null,
        timeout: '1h',
        foundEventID: null,
        timedOut: null,
      },
    });
    assert.strictEqual(isStepRunSpan(trace), false);
  });

  it('returns false for sleep span', () => {
    const trace = createMockTrace({
      stepOp: 'StepSleep',
      stepType: 'SLEEP',
      stepInfo: { sleepUntil: '2024-03-15T10:00:00.000Z' },
    });
    assert.strictEqual(isStepRunSpan(trace), false);
  });

  it('returns false for invoke span', () => {
    const trace = createMockTrace({
      stepOp: 'StepInvoke',
      stepType: 'INVOKE',
      stepInfo: {
        triggeringEventID: 'event-123',
        functionID: 'fn-123',
        timeout: '1h',
        returnEventID: null,
        runID: null,
        timedOut: null,
      },
    });
    assert.strictEqual(isStepRunSpan(trace), false);
  });

  it('returns false when no step info present', () => {
    const trace = createMockTrace({
      stepOp: null,
      stepType: null,
      stepInfo: null,
    });
    assert.strictEqual(isStepRunSpan(trace), false);
  });
});
