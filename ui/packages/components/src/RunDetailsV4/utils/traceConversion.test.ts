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

    it('computes discoveryMs as inter-step gap, not total elapsed time', () => {
      // Reproduces bug: without the fix, late-step would
      // get discoveryMs = 5000 (entire run elapsed time), making the
      // Inngest bar (5s+) vastly exceed the step bar (~40ms).
      // With the fix, discoveryMs is the gap between the previous sibling
      // ending and this step being queued (~5ms).
      const trace = createTrace({
        isRoot: true,
        startedAt: '2024-01-01T00:00:00Z',
        endedAt: '2024-01-01T00:00:05.100Z',
        childrenSpans: [
          createTrace({
            spanID: 'slow-step-1',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00.001Z', // 1ms after run start
            startedAt: '2024-01-01T00:00:00.010Z',
            endedAt: '2024-01-01T00:00:03Z', // 3s execution
          }),
          createTrace({
            spanID: 'slow-step-2',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:03.002Z', // 2ms after slow-step-1 ended
            startedAt: '2024-01-01T00:00:03.010Z',
            endedAt: '2024-01-01T00:00:05Z', // 2s execution
          }),
          createTrace({
            spanID: 'late-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:05.005Z', // 5ms after slow-step-2 ended
            startedAt: '2024-01-01T00:00:05.040Z', // 35ms queue wait
            endedAt: '2024-01-01T00:00:05.046Z', // 6ms execution
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const step1 = result.bars[0]?.children?.[0];
      const step2 = result.bars[0]?.children?.[1];
      const step3 = result.bars[0]?.children?.[2];

      // First step: discoveryMs = gap from runStartedAt to step queued = 1ms
      expect(step1?.inngestBreakdown?.discoveryMs).toBe(1);

      // Second step: discoveryMs = gap from slow-step-1 end to step queued = 2ms
      expect(step2?.inngestBreakdown?.discoveryMs).toBe(2);

      // Late step: discoveryMs = gap from slow-step-2 end to step queued = 5ms
      // NOT 5005ms (which would be the old runStartedAt-based calculation)
      expect(step3?.inngestBreakdown?.discoveryMs).toBe(5);

      // late-step bar duration is small (~41ms), and the Inngest bar
      // should fit within it since discoveryMs is only 5ms
      // inngestMs = startedAt - queuedAt = 5.040 - 5.005 = 35ms
      expect(step3?.timingBreakdown?.inngestMs).toBe(35);
      expect(step3?.timingBreakdown?.executionMs).toBe(6);
      expect(step3?.timingBreakdown?.totalMs).toBe(41);
    });

    it('widens timingBreakdown.inngestMs to include discovery when inngestBreakdown exceeds it', () => {
      const trace = createTrace({
        isRoot: true,
        startedAt: '2024-01-01T00:00:00Z',
        endedAt: '2024-01-01T00:00:10Z',
        childrenSpans: [
          createTrace({
            spanID: 'prev-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:00.010Z',
            endedAt: '2024-01-01T00:00:03Z', // ends at T+3s
          }),
          createTrace({
            spanID: 'run-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:03.100Z', // 100ms gap after prev-step
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
                  total_inngest_ms: 1000,
                },
              },
            ],
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });
      const bar = result.bars[0]?.children?.[1]; // second child

      // discoveryMs = gap from prev-step end (T+3) to this step queued (T+3.1) = 100ms
      expect(bar?.inngestBreakdown?.discoveryMs).toBe(100);
      expect(bar?.inngestBreakdown?.queueDelayMs).toBe(800);
      expect(bar?.inngestBreakdown?.systemLatencyMs).toBe(200);
      expect(bar?.inngestBreakdown?.totalMs).toBe(1100);

      // timingBreakdown.inngestMs (1000 from metadata) < inngestBreakdown.totalMs (1100)
      // so it gets widened to include discovery
      expect(bar?.timingBreakdown?.inngestMs).toBe(1100);
      expect(bar?.timingBreakdown?.executionMs).toBe(5000);
      expect(bar?.timingBreakdown?.totalMs).toBe(6100);
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
