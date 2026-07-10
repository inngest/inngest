/**
 * traceConversion utility tests.
 * Feature: 001-composable-timeline-bar
 */

import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup, traceToTimelineData } from './traceConversion';

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
    scheduledAt: '2024-01-01T00:00:00Z',
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

    it('calculates discovery from the previous completed sibling instead of run start', () => {
      const trace = createTrace({
        isRoot: true,
        startedAt: '2024-01-01T00:00:00Z',
        childrenSpans: [
          createTrace({
            spanID: 'slow-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:00Z',
            startedAt: '2024-01-01T00:00:00Z',
            endedAt: '2024-01-01T00:00:08Z',
          }),
          createTrace({
            spanID: 'quick-step',
            stepOp: 'RUN',
            queuedAt: '2024-01-01T00:00:08.100Z',
            startedAt: '2024-01-01T00:00:08.100Z',
            endedAt: '2024-01-01T00:00:08.350Z',
          }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      const quickStep = result.bars[0]?.children?.[1];
      expect(quickStep?.inngestBreakdown?.discoveryMs).toBe(100);
      expect(quickStep?.inngestBreakdown?.totalMs).toBe(100);
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

  describe('traceRollup', () => {
    it('passes single-attempt steps through unchanged, sorted by queuedAt', () => {
      const step1 = createTrace({
        spanID: 's1',
        stepID: 'step-1',
        attempts: 0,
        queuedAt: '2024-01-01T00:00:05Z',
        endedAt: '2024-01-01T00:00:06Z',
      });
      const step2 = createTrace({
        spanID: 's2',
        stepID: 'step-2',
        attempts: 0,
        queuedAt: '2024-01-01T00:00:01Z',
        endedAt: '2024-01-01T00:00:02Z',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [step1, step2] });

      const result = traceRollup(root);

      expect(result.childrenSpans?.map((c) => c.spanID)).toEqual(['s2', 's1']);
      expect(result.childrenSpans?.[1]).toBe(step1);
      expect(step1.name).toBe('test-step'); // no "Attempt N" renaming
    });

    it('rolls up a multi-attempt step into a virtual span', () => {
      const attempt0 = createTrace({
        spanID: 'a0',
        stepID: 'step-1',
        attempts: 0,
        name: 'my-step',
        queuedAt: '2024-01-01T00:00:00Z',
        startedAt: '2024-01-01T00:00:01Z',
        endedAt: '2024-01-01T00:00:02Z',
        status: 'FAILED',
      });
      const attempt1 = createTrace({
        spanID: 'a1',
        stepID: 'step-1',
        attempts: 1,
        name: 'my-step',
        queuedAt: '2024-01-01T00:00:02Z',
        startedAt: '2024-01-01T00:00:03Z',
        endedAt: '2024-01-01T00:00:04Z',
        status: 'COMPLETED',
        outputID: 'out-1',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [attempt0, attempt1] });

      const result = traceRollup(root);

      expect(result.childrenSpans).toHaveLength(1);
      const rollup = result.childrenSpans?.[0];
      expect(rollup?.spanID).toBe('step-1-rollup');
      expect(rollup?.name).toBe('my-step');
      expect(rollup?.stepID).toBe('step-1');
      expect(rollup?.attempts).toBe(1);
      // Start fields come from the first attempt, end fields from the last
      expect(rollup?.queuedAt).toBe('2024-01-01T00:00:00Z');
      expect(rollup?.startedAt).toBe('2024-01-01T00:00:01Z');
      expect(rollup?.endedAt).toBe('2024-01-01T00:00:04Z');
      expect(rollup?.status).toBe('COMPLETED');
      expect(rollup?.outputID).toBe('out-1');
      // Attempts are renamed and nested in order
      expect(rollup?.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);
      expect(rollup?.childrenSpans?.map((c) => c.spanID)).toEqual(['a0', 'a1']);
    });

    it('adopts grouped no-step spans as attempts of the step sharing their groupID', () => {
      // e.g. a network failure: has an output but never resolved to a stepID
      const failure = createTrace({
        spanID: 'f0',
        stepID: null,
        groupID: 'g1',
        attempts: 0,
        outputID: 'out-f',
        queuedAt: '2024-01-01T00:00:00Z',
        endedAt: '2024-01-01T00:00:01Z',
        status: 'FAILED',
      });
      const step = createTrace({
        spanID: 's1',
        stepID: 'step-1',
        groupID: 'g1',
        attempts: 1,
        queuedAt: '2024-01-01T00:00:01Z',
        endedAt: '2024-01-01T00:00:02Z',
        outputID: 'out-1',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [failure, step] });

      const result = traceRollup(root);

      // One rollup span; the grouped failure is not treated as finalization
      expect(result.childrenSpans).toHaveLength(1);
      const rollup = result.childrenSpans?.[0];
      expect(rollup?.spanID).toBe('step-1-rollup');
      expect(rollup?.childrenSpans?.map((c) => c.spanID)).toEqual(['f0', 's1']);
      expect(rollup?.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);
    });

    it('turns a trailing unmatched group into a Finalization span with clamped timestamps', () => {
      const step = createTrace({
        spanID: 's1',
        stepID: 'step-1',
        attempts: 0,
        queuedAt: '2024-01-01T00:00:00Z',
        endedAt: '2024-01-01T00:00:10Z',
      });
      const fin = createTrace({
        spanID: 'fin-0',
        stepID: null,
        groupID: 'g-final',
        attempts: 0,
        outputID: 'out-fin',
        // Queued before the last step ended; should be clamped to the step's end
        queuedAt: '2024-01-01T00:00:05Z',
        startedAt: '2024-01-01T00:00:06Z',
        endedAt: '2024-01-01T00:00:11Z',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [step, fin] });

      const result = traceRollup(root);

      expect(result.childrenSpans).toHaveLength(2);
      const finalization = result.childrenSpans?.[1];
      expect(finalization?.spanID).toBe('fin-0');
      expect(finalization?.name).toBe('Finalization');
      expect(finalization?.queuedAt).toBe('2024-01-01T00:00:10Z');
      expect(finalization?.startedAt).toBe('2024-01-01T00:00:10Z');
      expect(finalization?.endedAt).toBe('2024-01-01T00:00:11Z');
    });

    it('rolls up a multi-attempt unmatched group into a final-rollup virtual span', () => {
      const step = createTrace({
        spanID: 's1',
        stepID: 'step-1',
        attempts: 0,
        queuedAt: '2024-01-01T00:00:00Z',
        endedAt: '2024-01-01T00:00:10Z',
      });
      const fin0 = createTrace({
        spanID: 'fin-0',
        stepID: null,
        groupID: 'g-final',
        attempts: 0,
        outputID: 'out-f0',
        queuedAt: '2024-01-01T00:00:05Z',
        startedAt: '2024-01-01T00:00:06Z',
        endedAt: '2024-01-01T00:00:11Z',
        status: 'FAILED',
      });
      const fin1 = createTrace({
        spanID: 'fin-1',
        stepID: null,
        groupID: 'g-final',
        attempts: 1,
        outputID: 'out-f1',
        queuedAt: '2024-01-01T00:00:11Z',
        startedAt: '2024-01-01T00:00:12Z',
        endedAt: '2024-01-01T00:00:13Z',
        status: 'COMPLETED',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [step, fin0, fin1] });

      const result = traceRollup(root);

      expect(result.childrenSpans).toHaveLength(2);
      const finalization = result.childrenSpans?.[1];
      expect(finalization?.spanID).toBe('final-rollup');
      // The group ends COMPLETED, so this is a genuine (retried) finalization
      // — not a "Function error"
      expect(finalization?.name).toBe('Finalization');
      // Start clamped to the last step's end, end from the last attempt
      expect(finalization?.queuedAt).toBe('2024-01-01T00:00:10Z');
      expect(finalization?.endedAt).toBe('2024-01-01T00:00:13Z');
      expect(finalization?.status).toBe('COMPLETED');
      expect(finalization?.outputID).toBe('out-f1');
      expect(finalization?.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);
    });

    it('passes through output spans without stepID or groupID unchanged', () => {
      const outputSpan = createTrace({
        spanID: 'out-span',
        stepID: null,
        groupID: null,
        attempts: null,
        outputID: 'out-1',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [outputSpan] });

      const result = traceRollup(root);

      expect(result.childrenSpans).toHaveLength(1);
      expect(result.childrenSpans?.[0]).toBe(outputSpan);
      expect(outputSpan.name).toBe('test-step');
    });

    it('drops spans without a stepID/outputID and step spans with null attempts', () => {
      const noStepNoOutput = createTrace({
        spanID: 'x',
        stepID: null,
        outputID: null,
        attempts: 0,
      });
      const nullAttempts = createTrace({
        spanID: 'y',
        stepID: 'step-y',
        outputID: null,
        attempts: null,
      });
      const root = createTrace({ isRoot: true, childrenSpans: [noStepNoOutput, nullAttempts] });

      const result = traceRollup(root);

      expect(result.childrenSpans).toEqual([]);
    });

    it('handles a root with no children', () => {
      const root = createTrace({ isRoot: true, childrenSpans: undefined });

      const result = traceRollup(root);

      expect(result.childrenSpans).toEqual([]);
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

  // Realistic server shapes, captured from live runs. A step-execution
  // request can fail before the SDK returns a step opcode/ID, producing
  // "non-step" attempt spans with no stepID. These tests lock in the current
  // rendering of those shapes.
  describe('traceRollup — pre-stepID failed attempts', () => {
    const childNames = (t: Trace): string[] => (t.childrenSpans ?? []).map((c) => c.name);

    // Case A: attempts 0 and 1 fail with no step ID, attempt 2 succeeds. We DO
    // learn the step ID, so the attempts roll up under the resolved step and it
    // keeps its real name.
    it('groups failed pre-stepID attempts under the resolved step when it succeeds', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          createTrace({
            spanID: 'n0',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-step',
            outputID: 'o0',
          }),
          createTrace({
            spanID: 'n1',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 1,
            groupID: 'g-step',
            outputID: 'o1',
          }),
          createTrace({
            spanID: 'step-ok',
            name: 'the-only-step',
            stepID: 'c031',
            attempts: 2,
            groupID: 'g-step',
            outputID: 'o2',
          }),
          createTrace({
            spanID: 'final',
            name: 'executor.nonstep',
            status: 'COMPLETED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-root',
            outputID: 'o-final',
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));

      const step = out.childrenSpans?.find((c) => c.stepID === 'c031');
      expect(step?.name).toBe('the-only-step');
      expect(step?.childrenSpans?.map((c) => c.status)).toEqual(['FAILED', 'FAILED', 'COMPLETED']);

      // The terminal function output renders as "Finalization"; nothing is
      // labeled "Function error".
      expect(childNames(out)).toContain('Finalization');
      expect(childNames(out)).not.toContain('Function error');
    });

    // A stepless function whose body throws a real SDK error on every attempt:
    // the SDK responded each time (the outputs hold the user's actual error),
    // so this is the run's terminal work. The span shape is identical to Case
    // B's pre-SDK deaths minus the backend group span — the client cannot
    // tell them apart — which is exactly why the label is the neutral
    // "Function error": truthful whether the group holds the function's own
    // error or attempts that died before the SDK responded.
    it('labels an exhausted-retry function error as "Function error"', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        status: 'FAILED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          createTrace({
            spanID: 'n0',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-root',
            outputID: 'o0',
          }),
          createTrace({
            spanID: 'n1',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 1,
            groupID: 'g-root',
            outputID: 'o1',
          }),
          createTrace({
            spanID: 'n2',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 2,
            groupID: 'g-root',
            outputID: 'o2',
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));
      const rollup = out.childrenSpans?.find((c) => c.spanID === 'final-rollup');

      expect(out.childrenSpans).toHaveLength(1);
      expect(rollup?.name).toBe('Function error');
      expect(rollup?.childrenSpans?.map((c) => c.name)).toEqual([
        'Attempt 0',
        'Attempt 1',
        'Attempt 2',
      ]);
    });

    // Case B: the step 5xx's on every attempt, so its ID is never learned. The
    // server emits a pre-grouped backend span (no groupID, holds the attempts)
    // plus loose per-attempt spans. The grouped attempts have no step to
    // attribute to -> "Function error"; the backend passthrough still renders
    // alongside it (de-duping it is intentionally out of scope here).
    it('labels never-resolved failed attempts "Function error", not "Finalization"', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        status: 'FAILED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          createTrace({
            spanID: 'backend-group',
            name: 'Finalization',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            groupID: null,
            outputID: 'og',
            attempts: 2,
            childrenSpans: [
              createTrace({
                spanID: 'a0',
                name: 'Attempt 0',
                status: 'FAILED',
                stepID: null,
                stepOp: null,
                attempts: 0,
              }),
              createTrace({
                spanID: 'a1',
                name: 'Attempt 1',
                status: 'FAILED',
                stepID: null,
                stepOp: null,
                attempts: 1,
              }),
              createTrace({
                spanID: 'a2',
                name: 'Attempt 2',
                status: 'FAILED',
                stepID: null,
                stepOp: null,
                attempts: 2,
              }),
            ],
          }),
          createTrace({
            spanID: 'n0',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-root',
            outputID: 'o0',
          }),
          createTrace({
            spanID: 'n1',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 1,
            groupID: 'g-root',
            outputID: 'o1',
          }),
          createTrace({
            spanID: 'n2',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 2,
            groupID: 'g-root',
            outputID: 'o2',
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));
      const rollup = out.childrenSpans?.find((c) => c.spanID === 'final-rollup');

      // The rolled-up loose attempts are now surfaced as a function error.
      expect(rollup?.name).toBe('Function error');
      expect(rollup?.childrenSpans).toHaveLength(3);
      expect(childNames(out)).toEqual(['Finalization', 'Function error']);
    });

    // The final discovery itself can retry: the request after the last step
    // completes 500s once, then succeeds. Fixture mirrors a real dev-server
    // trace: the server emits a pre-grouped "Finalization" span (with a stepID
    // and no groupID) plus the loose per-attempt spans. Current behavior: the
    // pre-grouped span passes through via the steps map and the loose attempts
    // roll into a group ending COMPLETED — a genuine (retried) finalization,
    // not an unresolved step.
    it('labels a retried-but-successful finalization "Finalization", not "Function error"', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'flaky-finalization-probe',
        status: 'COMPLETED',
        stepID: 'fn-hash',
        stepOp: null,
        groupID: 'g-root',
        queuedAt: '2024-01-01T00:00:00Z',
        startedAt: '2024-01-01T00:00:00.050Z',
        endedAt: '2024-01-01T00:00:38Z',
        childrenSpans: [
          // Server-grouped finalization span: carries a stepID (the function
          // hash) and no groupID, so traceRollup passes it through untouched.
          createTrace({
            spanID: 'backend-final-group',
            name: 'Finalization',
            status: 'COMPLETED',
            stepID: 'fn-hash',
            stepOp: null,
            groupID: null,
            attempts: 1,
            outputID: 'og',
            queuedAt: '2024-01-01T00:00:00Z',
            endedAt: '2024-01-01T00:00:38Z',
          }),
          createTrace({
            spanID: 'n0',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-root',
            outputID: 'o0',
            queuedAt: '2024-01-01T00:00:00Z',
            endedAt: '2024-01-01T00:00:00.400Z',
          }),
          createTrace({
            spanID: 'step-work',
            name: 'work',
            status: 'COMPLETED',
            stepID: 'step-hash',
            stepOp: 'RUN',
            attempts: 0,
            groupID: null,
            outputID: 'ow',
            queuedAt: '2024-01-01T00:00:00.349Z',
            endedAt: '2024-01-01T00:00:00.350Z',
          }),
          createTrace({
            spanID: 'n1',
            name: 'executor.nonstep',
            status: 'COMPLETED',
            stepID: null,
            stepOp: null,
            attempts: 1,
            groupID: 'g-root',
            outputID: 'o1',
            queuedAt: '2024-01-01T00:00:00.400Z',
            endedAt: '2024-01-01T00:00:38Z',
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));
      const rollup = out.childrenSpans?.find((c) => c.spanID === 'final-rollup');

      expect(rollup?.name).toBe('Finalization');
      expect(rollup?.status).toBe('COMPLETED');
      expect(rollup?.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);

      // The backend pre-grouped span still renders alongside the rollup
      // (de-duped server-side; out of scope for the client rollup).
      expect(childNames(out)).toEqual(['Finalization', 'work', 'Finalization']);
    });

    // Case C: a single pre-SDK failure (e.g. retries: 0). The lone FAILED
    // nonstep is labeled "Function error" (status-based naming: the group is
    // where the run failed); the backend passthrough renders alongside it.
    it('renders a single failed pre-stepID attempt alongside its backend passthrough', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        status: 'FAILED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          createTrace({
            spanID: 'backend-group',
            name: 'Finalization',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            groupID: null,
            outputID: 'og',
            attempts: 0,
          }),
          createTrace({
            spanID: 'n0',
            name: 'executor.nonstep',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            attempts: 0,
            groupID: 'g-root',
            outputID: 'o0',
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));

      expect(out.childrenSpans).toHaveLength(2);
      expect(childNames(out)).toEqual(['Finalization', 'Function error']);
    });

    // Case D: a groupID-less finalization span with NO failed pre-SDK attempt
    // group to duplicate is kept — the happy path depends on it.
    it('keeps a groupID-less finalization when no terminal failure is rendered', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        status: 'COMPLETED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          createTrace({
            spanID: 'step-ok',
            name: 'the-only-step',
            status: 'COMPLETED',
            stepID: 'c031',
            attempts: 0,
            groupID: 'g-step',
            outputID: 'os',
          }),
          createTrace({
            spanID: 'lone-final',
            name: 'Finalization',
            status: 'COMPLETED',
            stepID: null,
            stepOp: null,
            groupID: null,
            outputID: 'of',
            attempts: 0,
          }),
        ],
      });

      const out = traceRollup(structuredClone(root));

      expect(childNames(out)).toContain('Finalization');
    });
  });
});
