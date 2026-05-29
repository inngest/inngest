/**
 * traceConversion utility tests.
 * Feature: 001-composable-timeline-bar
 */

import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceToTimelineData } from './traceConversion';

describe('traceConversion', () => {
  // Helper to create a minimal valid trace
  const createTrace = (overrides: Partial<Trace> = {}): Trace => ({
    attempts: null,
    childrenSpans: undefined,
    endedAt: '2024-01-01T00:00:10Z',
    isRoot: false,
    name: 'test-step',
    outputID: null,
    queuedAt: '2024-01-01T00:00:00Z',
    spanID: 'span-1',
    stepID: 'step-1',
    startedAt: '2024-01-01T00:00:02Z',
    status: 'COMPLETED',
    stepInfo: null,
    stepOp: 'RUN',
    stepType: null,
    userlandSpan: null,
    isUserland: false,
    ...overrides,
  });

  describe('traceToTimelineData', () => {
    it('converts a simple trace to timeline data', () => {
      const trace = createTrace({ isRoot: true });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.minTime).toEqual(new Date('2024-01-01T00:00:00Z'));
      expect(result.maxTime).toEqual(new Date('2024-01-01T00:00:10Z'));
      expect(result.bars).toHaveLength(1);
      expect(result.leftWidth).toBe(35); // default
    });

    it('uses provided leftWidth option', () => {
      const trace = createTrace({ isRoot: true });
      const result = traceToTimelineData(trace, { runID: 'run-1', leftWidth: 45 });

      expect(result.leftWidth).toBe(45);
    });

    it('passes orgName through to result', () => {
      const trace = createTrace({ isRoot: true });
      const result = traceToTimelineData(trace, { runID: 'run-1', orgName: 'Acme Corp' });

      expect(result.orgName).toBe('Acme Corp');
    });

    it('renames root trace to "Run"', () => {
      const trace = createTrace({ isRoot: true, name: 'my-function' });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.name).toBe('Run');
    });

    it('sets isRoot to true on root bar', () => {
      const trace = createTrace({ isRoot: true });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.isRoot).toBe(true);
    });

    it('calculates min/max time from nested children', () => {
      const trace = createTrace({
        isRoot: true,
        queuedAt: '2024-01-01T00:00:05Z',
        endedAt: '2024-01-01T00:00:15Z',
        childrenSpans: [
          createTrace({
            spanID: 'child-1',
            queuedAt: '2024-01-01T00:00:00Z', // Earlier than parent
            endedAt: '2024-01-01T00:00:10Z',
          }),
          createTrace({
            spanID: 'child-2',
            queuedAt: '2024-01-01T00:00:08Z',
            endedAt: '2024-01-01T00:00:20Z', // Later than parent
          }),
        ],
      });

      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.minTime).toEqual(new Date('2024-01-01T00:00:00Z'));
      expect(result.maxTime).toEqual(new Date('2024-01-01T00:00:20Z'));
    });

    it('handles trace with null endedAt (in progress)', () => {
      const trace = createTrace({
        isRoot: true,
        endedAt: null,
      });

      const result = traceToTimelineData(trace, { runID: 'run-1' });

      // maxTime should be roughly "now" - just check it's a valid date
      expect(result.maxTime).toBeInstanceOf(Date);
      expect(result.maxTime.getTime()).toBeGreaterThan(result.minTime.getTime());
    });
  });

  describe('bar style mapping', () => {
    it('returns "root" style for root traces', () => {
      const trace = createTrace({ isRoot: true });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.style).toBe('root');
    });

    it('returns "default" style for userland traces', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ isUserland: true, spanID: 'userland-1' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('default');
    });

    it('returns "step.run" for RUN stepOp', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: 'RUN', spanID: 'run-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.run');
    });

    it('returns "step.sleep" for SLEEP stepOp', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: 'SLEEP', spanID: 'sleep-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.sleep');
    });

    it('returns "step.waitForEvent" for WAIT_FOR_EVENT stepOp', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: 'WAIT_FOR_EVENT', spanID: 'wait-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.waitForEvent');
    });

    it('returns "step.invoke" for INVOKE stepOp', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: 'INVOKE', spanID: 'invoke-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.invoke');
    });

    it('handles case-insensitive stepOp (lowercase)', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: 'sleep', spanID: 'sleep-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.sleep');
    });

    it('falls back to stepType when stepOp is not set', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ stepOp: null, stepType: 'SLEEP', spanID: 'sleep-step' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.style).toBe('step.sleep');
    });
  });

  describe('timing breakdown', () => {
    it('calculates timing breakdown for step.run spans', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:02Z', // 2s queue time
            endedAt: '2024-01-01T00:00:07Z', // 5s execution time
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      expect(childBar?.timingBreakdown).toBeDefined();
      expect(childBar?.timingBreakdown?.inngestMs).toBe(2000);
      expect(childBar?.timingBreakdown?.executionMs).toBe(5000);
      expect(childBar?.timingBreakdown?.totalMs).toBe(7000);
    });

    it('does not calculate timing breakdown for userland spans', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'userland-step',
            isUserland: true,
            stepOp: 'RUN',
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.timingBreakdown).toBeUndefined();
    });

    it('does not calculate timing breakdown for non-RUN steps', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'sleep-step',
            stepOp: 'SLEEP',
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.timingBreakdown).toBeUndefined();
    });

    it('includes non-step.run children wall-clock duration in root bar execution time', () => {
      const trace = createTrace({
        isRoot: true,
        queuedAt: '2024-01-01T00:00:00Z',
        startedAt: '2024-01-01T00:00:01Z',
        endedAt: '2024-01-01T00:01:11Z', // 71s total
        childrenSpans: [
          // step.run child: 1s execution (has timingBreakdown)
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:01Z',
            startedAt: '2024-01-01T00:00:02Z',
            endedAt: '2024-01-01T00:00:03Z',
          }),
          // step.sleep child: 60s wall-clock (no timingBreakdown)
          createTrace({
            spanID: 'sleep-step',
            stepOp: 'SLEEP',
            queuedAt: '2024-01-01T00:00:03Z',
            startedAt: '2024-01-01T00:00:03Z',
            endedAt: '2024-01-01T00:01:03Z',
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const rootBar = result.bars[0];
      expect(rootBar?.timingBreakdown).toBeDefined();
      // execution = 1s (step.run) + 60s (step.sleep wall-clock) = 61s
      expect(rootBar?.timingBreakdown?.executionMs).toBe(61000);
      // inngest overhead = 71s total - 61s execution = 10s (not 70s!)
      expect(rootBar?.timingBreakdown?.inngestMs).toBe(10000);
      expect(rootBar?.timingBreakdown?.totalMs).toBe(71000);
    });

    it('prefers metadata timing over timestamp-based calculation', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:02Z', // 2s timestamp delta
            endedAt: '2024-01-01T00:00:07Z', // 5s execution
            metadata: [
              {
                scope: 'step_attempt',
                kind: 'inngest.timing',
                updatedAt: '2024-01-01T00:00:07Z',
                values: {
                  total_inngest_ms: 3500, // Different from timestamp-based 2000ms
                },
              },
            ],
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      expect(childBar?.timingBreakdown).toBeDefined();
      // Should use metadata value (3500ms), not timestamp delta (2000ms)
      expect(childBar?.timingBreakdown?.inngestMs).toBe(3500);
      expect(childBar?.timingBreakdown?.executionMs).toBe(5000);
      expect(childBar?.timingBreakdown?.totalMs).toBe(8500);
    });

    it('falls back to timestamp calculation when metadata has no inngest.timing', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:02Z',
            endedAt: '2024-01-01T00:00:07Z',
            metadata: [
              {
                scope: 'step_attempt',
                kind: 'inngest.http',
                updatedAt: '2024-01-01T00:00:07Z',
                values: { req_method: 'POST' },
              },
            ],
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      expect(childBar?.timingBreakdown).toBeDefined();
      // Falls back to timestamp-based: startedAt - queuedAt = 2000ms
      expect(childBar?.timingBreakdown?.inngestMs).toBe(2000);
      expect(childBar?.timingBreakdown?.executionMs).toBe(5000);
      expect(childBar?.timingBreakdown?.totalMs).toBe(7000);
    });

    it('calculates inngestBreakdown from metadata timing values', () => {
      const trace = createTrace({
        isRoot: true,
        startedAt: '2024-01-01T00:00:01Z', // run started at T+1s
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:03Z', // queued 2s after run started
            startedAt: '2024-01-01T00:00:05Z',
            endedAt: '2024-01-01T00:00:10Z',
            metadata: [
              {
                scope: 'step_attempt',
                kind: 'inngest.timing',
                updatedAt: '2024-01-01T00:00:10Z',
                values: {
                  queue_delay_ms: 800,
                  system_latency_ms: 200,
                  total_inngest_ms: 2000,
                },
              },
            ],
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      expect(childBar?.inngestBreakdown).toBeDefined();
      // discoveryMs = queuedAt - runStartedAt = 3s - 1s = 2000ms
      expect(childBar?.inngestBreakdown?.discoveryMs).toBe(2000);
      expect(childBar?.inngestBreakdown?.queueDelayMs).toBe(800);
      expect(childBar?.inngestBreakdown?.systemLatencyMs).toBe(200);
      expect(childBar?.inngestBreakdown?.totalMs).toBe(3000);
    });

    it('returns no inngestBreakdown without metadata timing', () => {
      const trace = createTrace({
        isRoot: true,
        startedAt: '2024-01-01T00:00:01Z',
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:01Z', // same as run start => discoveryMs=0
            startedAt: '2024-01-01T00:00:01Z',
            endedAt: '2024-01-01T00:00:05Z',
            // No metadata
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      // Without metadata, queueDelayMs=0 and systemLatencyMs=0, discoveryMs=0 => totalMs=0 => null
      expect(childBar?.inngestBreakdown).toBeUndefined();
    });

    it('handles zero queue time', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:00Z', // Same as queuedAt
            endedAt: '2024-01-01T00:00:05Z',
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const childBar = result.bars[0]?.children?.[0];
      expect(childBar?.timingBreakdown?.inngestMs).toBe(0);
      expect(childBar?.timingBreakdown?.executionMs).toBe(5000);
    });
  });

  describe('name formatting', () => {
    it('removes "step." prefix from names', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ name: 'step.processPayment', spanID: 'step-1' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.name).toBe('processPayment');
    });

    it('removes "inngest/" prefix from names', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ name: 'inngest/sendEmail', spanID: 'step-1' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.name).toBe('sendEmail');
    });

    it('keeps names without known prefixes unchanged', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ name: 'myCustomStep', spanID: 'step-1' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.name).toBe('myCustomStep');
    });
  });

  describe('nested children conversion', () => {
    it('converts nested children recursively', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'parent-step',
            name: 'parent',
            childrenSpans: [
              createTrace({
                spanID: 'child-step',
                name: 'child',
              }),
            ],
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.name).toBe('parent');
      expect(result.bars[0]?.children?.[0]?.children?.[0]?.name).toBe('child');
    });

    it('preserves spanID as bar id', () => {
      const trace = createTrace({
        isRoot: true,
        spanID: 'root-span-id',
        childrenSpans: [createTrace({ spanID: 'child-span-id' })],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.id).toBe('root-span-id');
      expect(result.bars[0]?.children?.[0]?.id).toBe('child-span-id');
    });
  });
});
