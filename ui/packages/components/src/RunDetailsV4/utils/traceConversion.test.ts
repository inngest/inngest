/**
 * traceConversion utility tests.
 * Feature: 001-composable-timeline-bar
 */

import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { STATEMENT_VERBS, getSandboxMetadata, statementVerb } from './sandbox';
import {
  BUILD_MACHINE_NAME,
  BUILD_MACHINE_STATEMENT_ID,
  BUILD_SANDBOX_ID,
  SERVE_NAME,
  SNAPSHOT_STATEMENT_ID,
  TEST_NAME,
  TEST_STATEMENT_ID,
  ciTestSteps,
  machineSteps,
  sandboxCIRun,
  snapshotSteps,
} from './sandboxTrace.fixture';
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
      expect(result.childrenSpans?.[1]).toEqual(step1); // clone of the input span
      expect(result.childrenSpans?.[1]?.name).toBe('test-step'); // no "Attempt N" renaming
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
      expect(result.childrenSpans?.[0]).toEqual(outputSpan); // clone of the input span
      expect(result.childrenSpans?.[0]?.name).toBe('test-step');
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

    it('attaches each span its own scores without walking children', () => {
      const trace = createTrace({
        isRoot: true,
        metadata: [
          {
            scope: 'run',
            kind: 'inngest.score',
            updatedAt: '2024-01-01T00:00:05Z',
            values: { relevance: { value: 0.98765 } },
          },
        ],
        childrenSpans: [
          createTrace({
            spanID: 'scored-step',
            metadata: [
              {
                scope: 'step',
                kind: 'inngest.score',
                updatedAt: '2024-01-01T00:00:06Z',
                values: { passed: { value: true }, accuracy: { value: 3 } },
              },
            ],
          }),
          createTrace({ spanID: 'unscored-step', stepID: 'step-2' }),
        ],
      });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.scores).toEqual([{ name: 'relevance', value: 0.98765 }]);
      expect(result.bars[0]?.children?.[0]?.scores).toEqual([
        { name: 'accuracy', value: 3 },
        { name: 'passed', value: true },
      ]);
      expect(result.bars[0]?.children?.[1]?.scores).toBeUndefined();
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
    // attribute to -> "Function error", and the redundant backend passthrough
    // is dropped (EXE-1992).
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
          // Backend group span: no groupID, holds the attempts. It duplicates
          // the loose per-attempt spans below (the same terminal failure
          // surfaced twice). Once those roll into a "Function error" group,
          // this redundant "Finalization" passthrough is dropped (EXE-1992).
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

      // EXE-1992: the duplicate groupID-less "Finalization" passthrough is
      // dropped once the attempts roll into "Function error", leaving it as
      // the only run child.
      expect(out.childrenSpans).toHaveLength(1);
      expect(childNames(out)).toEqual(['Function error']);
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
    // where the run failed), and the groupID-less passthrough must still be
    // de-duped — otherwise a redundant "Finalization" span renders alongside
    // it (EXE-1992 with a single attempt).
    it('de-dupes the finalization passthrough for a single failed pre-stepID attempt', () => {
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

      // Exactly one terminal span; the duplicate passthrough is gone.
      expect(out.childrenSpans).toHaveLength(1);
      expect(childNames(out)).toEqual(['Function error']);
    });

    // Case D: a groupID-less finalization span with NO failed pre-SDK attempt
    // group to duplicate must be KEPT. Guards the drop from degrading into a
    // blunt "always remove" (which would still pass the failure cases above).
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
    // Post-#4600: the dev server now emits its pre-grouped "Finalization" span
    // with stepID=null AND outputID=null (previously outputID carried a value).
    // collectRollupGroups's `child.outputID && !child.stepID` check no longer
    // matches, so the span falls through to the `!child.stepID` guard and is
    // silently dropped — the loose per-attempt spans are canonical and roll up
    // on their own. These two tests lock in that dev-server shape.
    it('drops the stepID-less, outputID-less finalization group span (COMPLETED-after-retry)', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'flaky-finalization-probe',
        status: 'COMPLETED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          // Cleared server group span: post-#4600, stepID AND outputID are
          // both null, so it no longer matches the outputID/stepID filter in
          // collectRollupGroups and is dropped outright (its nested attempts
          // are never visited — only the loose spans below are canonical).
          createTrace({
            spanID: 'backend-final-group',
            name: 'Finalization',
            status: 'COMPLETED',
            stepID: null,
            stepOp: null,
            groupID: null,
            outputID: null,
            attempts: 1,
            queuedAt: '2024-01-01T00:00:00Z',
            endedAt: '2024-01-01T00:00:38Z',
            childrenSpans: [
              createTrace({
                spanID: 'backend-a0',
                name: 'Attempt 0',
                status: 'FAILED',
                stepID: null,
              }),
              createTrace({
                spanID: 'backend-a1',
                name: 'Attempt 1',
                status: 'COMPLETED',
                stepID: null,
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

      // The loose spans roll up into a single genuine (retried) finalization.
      const rollup = out.childrenSpans?.find((c) => c.spanID === 'final-rollup');
      expect(rollup?.name).toBe('Finalization');
      expect(rollup?.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);

      // The cleared group span is dropped: no 'backend-final-group' survivor,
      // and no second "Finalization" alongside the rollup's.
      // The rollup's queuedAt is clamped past the step's end, so it sorts
      // after "work".
      expect(childNames(out)).toEqual(['work', 'Finalization']);
    });

    it('drops the stepID-less, outputID-less finalization group span (all attempts FAILED pre-SDK)', () => {
      const root = createTrace({
        isRoot: true,
        spanID: 'run',
        name: 'Run',
        status: 'FAILED',
        stepID: null,
        stepOp: null,
        groupID: 'g-root',
        childrenSpans: [
          // Cleared server group span, FAILED variant: still stepID=null AND
          // outputID=null post-#4600, so it's dropped the same way regardless
          // of status.
          createTrace({
            spanID: 'backend-final-group',
            name: 'Finalization',
            status: 'FAILED',
            stepID: null,
            stepOp: null,
            groupID: null,
            outputID: null,
            attempts: 2,
            childrenSpans: [
              createTrace({
                spanID: 'backend-a0',
                name: 'Attempt 0',
                status: 'FAILED',
                stepID: null,
              }),
              createTrace({
                spanID: 'backend-a1',
                name: 'Attempt 1',
                status: 'FAILED',
                stepID: null,
              }),
              createTrace({
                spanID: 'backend-a2',
                name: 'Attempt 2',
                status: 'FAILED',
                stepID: null,
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

      // The loose spans roll into a single "Function error" — the group is
      // where the run failed.
      expect(childNames(out)).toEqual(['Function error']);
    });

    // traceRollup must NOT mutate its input. The result is memoized against the
    // trace object, so on a re-render the memo re-runs traceRollup(trace) on the
    // same object; if that object had been mutated into the rolled-up shape, the
    // "Function error" group would collapse to a single span and get relabeled
    // "Finalization". Non-mutation keeps every call operating on pristine input.
    it('does not mutate its input, so repeated calls are stable', () => {
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
      const frozen = JSON.stringify(root);

      const first = traceRollup(root);
      // Input is untouched: the rollup operates on a clone.
      expect(JSON.stringify(root)).toBe(frozen);
      expect(childNames(first)).toContain('Function error');

      // Re-running on the same (still-pristine) input yields the same labels —
      // this is what the memoized render does across re-renders/polls.
      const second = traceRollup(root);
      expect(childNames(second)).toEqual(childNames(first));
    });
  });

  describe('sandbox statements', () => {
    const rolledUp = () => traceRollup(sandboxCIRun());
    const childByName = (root: Trace, name: string) =>
      root.childrenSpans?.find((c) => c.name === name);
    const barsOf = (root: Trace) =>
      traceToTimelineData(root, { runID: 'run-1' }).bars[0]?.children ?? [];
    const barByName = (root: Trace, name: string) => barsOf(root).find((b) => b.name === name);
    const phaseSummary = (root: Trace, name: string) =>
      barByName(root, name)?.sandbox?.phases?.map((p) => ({
        label: p.label,
        start: p.startTime.toISOString(),
        end: p.endTime?.toISOString() ?? null,
      }));

    it('folds a CI command with no statement step, sleeps included, into one row', () => {
      const root = rolledUp();

      expect(root.childrenSpans?.map((c) => c.name)).toEqual([
        'build › machine',
        'get-e2e',
        'install',
        'e2e-install',
        'notify-start',
        TEST_NAME,
        'e2e',
        'e2e-release',
        SERVE_NAME,
        'snapshot-build',
        'build › pause',
        'destroy-build',
      ]);

      const test = childByName(root, TEST_NAME)!;
      expect(test.spanID).toBe(`${TEST_STATEMENT_ID}-statement`);
      expect(test.queuedAt).toBe('2026-10-01T12:00:38.000Z');
      expect(test.endedAt).toBe('2026-10-01T12:02:02.200Z');
      expect(test.status).toBe('COMPLETED');
      expect(test.sandboxMembers).toHaveLength(8);
      // Internal steps are states of the row, not sub-rows
      expect(test.childrenSpans).toEqual([]);
      // The step panel shows the command's output, with its command and exit code
      expect(test.outputID).toBe('output-a10-output-0');
      expect(getSandboxMetadata(test)).toMatchObject({
        action: 'process.output',
        command_display: 'pnpm test',
        exit_code: 0,
      });
    });

    it('draws the CI command as a few states, not one per step', () => {
      const root = rolledUp();
      const bar = barByName(root, TEST_NAME)!;

      expect(phaseSummary(root, TEST_NAME)).toEqual([
        {
          label: 'Starting',
          start: '2026-10-01T12:00:38.000Z',
          end: '2026-10-01T12:00:38.500Z',
        },
        {
          label: 'Running',
          start: '2026-10-01T12:00:38.500Z',
          end: '2026-10-01T12:01:55.200Z',
        },
        {
          label: 'Collecting output',
          start: '2026-10-01T12:01:55.200Z',
          end: '2026-10-01T12:02:02.200Z',
        },
      ]);
      expect(bar.sandbox?.phases?.map((p) => p.waiting)).toEqual([false, true, false]);
      expect(bar.sandbox).toMatchObject({
        sandboxId: BUILD_SANDBOX_ID,
        machineLabel: BUILD_MACHINE_NAME,
        statement: 'commands.run',
        command: 'pnpm test',
        // CI titles the row with the command, so the annotation doesn't repeat it
        commandInTitle: true,
        exitCode: 0,
      });
      expect(bar.sandbox?.attempts).toBeUndefined();
      // Per-step timing doesn't apply to a row that spans several steps
      expect(bar.timingBreakdown).toBeUndefined();
      expect(bar.inngestBreakdown).toBeUndefined();
      expect(bar.delayMs).toBeUndefined();
    });

    it('titles a snapshot row by its statement step and owns its readiness wait', () => {
      const root = rolledUp();
      const snapshot = childByName(root, 'snapshot-build')!;

      expect(snapshot.spanID).toBe(`${SNAPSHOT_STATEMENT_ID}-statement`);
      expect(snapshot.stepID).toBe(SNAPSHOT_STATEMENT_ID);
      expect(childByName(root, 'snapshot-build:wait-until-ready')).toBeUndefined();
      expect(snapshot.outputID).toBe('output-b2000000000000000000000000wait-until-ready-0');
      expect(phaseSummary(root, 'snapshot-build')).toEqual([
        {
          label: 'Creating snapshot',
          start: '2026-10-01T12:02:02.500Z',
          end: '2026-10-01T12:02:03.300Z',
        },
        {
          label: 'Waiting until ready',
          start: '2026-10-01T12:02:03.300Z',
          end: '2026-10-01T12:02:23.300Z',
        },
      ]);
    });

    it('passes single-step statements through unchanged', () => {
      const raw = sandboxCIRun();
      const install = raw.childrenSpans!.find((c) => c.name === 'install')!;
      const root = traceRollup(raw);

      expect(childByName(root, 'install')).toEqual(install);
      expect(barByName(root, 'install')?.sandbox).toMatchObject({
        command: 'pnpm install',
        exitCode: 0,
        annotate: true,
      });
      expect(barByName(root, 'install')?.sandbox?.phases).toBeUndefined();
    });

    it('keeps retries of the statement step as its attempts', () => {
      const [create, wait] = snapshotSteps();
      const failedCreate = { ...create!, spanID: 'snap-attempt-0', status: 'FAILED' };
      const retriedCreate = {
        ...create!,
        spanID: 'snap-attempt-1',
        attempts: 1,
        queuedAt: '2026-10-01T12:02:02.900Z',
      };
      const root = traceRollup(
        createTrace({ isRoot: true, childrenSpans: [failedCreate, retriedCreate, wait!] })
      );

      expect(root.childrenSpans).toHaveLength(1);
      const snapshot = root.childrenSpans![0]!;
      expect(snapshot.name).toBe('snapshot-build');
      expect(snapshot.attempts).toBe(1);
      expect(snapshot.status).toBe('COMPLETED');
      expect(snapshot.childrenSpans?.map((c) => c.name)).toEqual(['Attempt 0', 'Attempt 1']);
    });

    it('hides step-level retries of an internal step inside the row', () => {
      const steps = ciTestSteps();
      const check = steps[2]!;
      const retriedCheck = { ...check, spanID: 'check-retry', attempts: 1 };
      const root = traceRollup(
        createTrace({ isRoot: true, childrenSpans: [...steps, retriedCheck] })
      );

      expect(root.childrenSpans).toHaveLength(1);
      expect(root.childrenSpans![0]!.name).toBe(TEST_NAME);
      expect(root.childrenSpans![0]!.childrenSpans).toEqual([]);
      expect(root.childrenSpans![0]!.sandboxMembers).toHaveLength(8);
    });

    it('keeps a still-running statement open', () => {
      const steps = ciTestSteps().slice(0, 6);
      const lastSleep = steps[5]!;
      steps[5] = { ...lastSleep, endedAt: null, status: 'WAITING' };
      const root = traceRollup(createTrace({ isRoot: true, childrenSpans: steps }));
      const test = root.childrenSpans![0]!;

      expect(test.endedAt).toBeNull();
      expect(test.status).toBe('RUNNING');

      const phases = barsOf(root)[0]?.sandbox?.phases;
      expect(phases?.map((p) => p.label)).toEqual(['Starting', 'Running']);
      expect(phases?.[1]?.endTime).toBeNull();
    });

    it('draws a retried CI command on one row, the failed attempt first', () => {
      const steps = [
        ...ciTestSteps({ attempt: 1, exitCode: 1 }),
        ...ciTestSteps({ attempt: 2, offset: 100 }),
      ];
      const root = traceRollup(createTrace({ isRoot: true, childrenSpans: steps }));

      expect(root.childrenSpans).toHaveLength(1);
      const test = root.childrenSpans![0]!;
      expect(test.name).toBe(TEST_NAME);
      expect(test.sandboxMembers).toHaveLength(16);
      expect(test.childrenSpans).toEqual([]);
      expect(test.status).toBe('COMPLETED');
      expect(test.outputID).toBe('output-a12-output-0');

      const bar = barsOf(root)[0]!;
      expect(bar.sandbox?.exitCode).toBe(0);
      // The badge reads "exit 0 · 2 attempts"
      expect(bar.sandbox?.attempts).toBe(2);
      expect(bar.sandbox?.phases?.map((p) => [p.label, p.failed])).toEqual([
        ['Attempt 1: Starting', true],
        ['Attempt 1: Running', true],
        ['Attempt 1: Collecting output', true],
        ['Attempt 2: Starting', false],
        ['Attempt 2: Running', false],
        ['Attempt 2: Collecting output', false],
      ]);
      // Attempt 1's output state ends where attempt 2 starts
      expect(bar.sandbox?.phases?.[2]?.endTime?.toISOString()).toBe('2026-10-01T12:02:18.000Z');
    });

    it('marks a statement failed when its last step failed', () => {
      const steps = ciTestSteps();
      const output = steps[7]!;
      steps[7] = { ...output, status: 'FAILED' };
      const root = traceRollup(createTrace({ isRoot: true, childrenSpans: steps }));

      expect(root.childrenSpans![0]!.status).toBe('FAILED');
    });

    it('falls back to the first step name with no statement step or statement_name', () => {
      const steps = ciTestSteps().map((step) => ({
        ...step,
        metadata: step.metadata?.map((md) => ({
          ...md,
          values: { ...md.values, statement_name: undefined },
        })),
      })) as Trace[];
      const root = traceRollup(createTrace({ isRoot: true, childrenSpans: steps }));

      expect(root.childrenSpans![0]!.name).toBe(`${TEST_NAME} › start`);
    });

    it('marks the first row of a machine the run did not create as existing', () => {
      const root = rolledUp();
      const existing = barsOf(root).map((b) => [b.name, b.sandbox?.existing]);

      expect(existing).toEqual([
        ['build › machine', false],
        ['get-e2e', true],
        ['install', undefined],
        ['e2e-install', undefined],
        ['notify-start', undefined],
        [TEST_NAME, undefined],
        ['e2e', undefined],
        ['e2e-release', undefined],
        [SERVE_NAME, undefined],
        ['snapshot-build', undefined],
        ['build › pause', undefined],
        ['destroy-build', undefined],
      ]);
    });

    it('titles a lone internal step by its statement_name and gives it states', () => {
      const root = rolledUp();
      const serve = childByName(root, SERVE_NAME)!;

      expect(childByName(root, `${SERVE_NAME} › start`)).toBeUndefined();
      expect(serve.sandboxMembers).toHaveLength(1);
      expect(serve.childrenSpans).toEqual([]);
      expect(barByName(root, SERVE_NAME)?.sandbox).toMatchObject({
        statement: 'processes.start',
        command: 'pnpm serve',
        commandInTitle: true,
      });
      expect(phaseSummary(root, SERVE_NAME)?.map((p) => p.label)).toEqual(['Starting']);
    });

    it("folds a machine's setup into its create row as states", () => {
      const root = rolledUp();
      const machine = childByName(root, 'build › machine')!;

      expect(childByName(root, 'build › machine › setup')).toBeUndefined();
      expect(machine.stepID).toBe(BUILD_MACHINE_STATEMENT_ID);
      expect(machine.sandboxMembers).toHaveLength(2);
      expect(phaseSummary(root, 'build › machine')).toEqual([
        { label: 'Creating', start: '2026-10-01T12:00:00.000Z', end: '2026-10-01T12:00:04.700Z' },
        { label: 'Setting up', start: '2026-10-01T12:00:04.700Z', end: '2026-10-01T12:00:05.100Z' },
      ]);
      // The setup command is CI's own work, not the row's command
      const sandbox = barByName(root, 'build › machine')?.sandbox;
      expect(sandbox?.statement).toBe('create');
      expect(sandbox?.command).toBeUndefined();
      expect(sandbox?.exitCode).toBeUndefined();
    });

    it("shows a create row's failing setup exit code", () => {
      const [create, setup] = machineSteps();
      const failedSetup = {
        ...setup!,
        status: 'FAILED',
        metadata: setup!.metadata?.map((md) => ({
          ...md,
          values: { ...md.values, exit_code: 2 },
        })),
      } as Trace;
      const root = traceRollup(
        createTrace({ isRoot: true, childrenSpans: [create!, failedSetup] })
      );

      expect(barsOf(root)[0]?.sandbox?.exitCode).toBe(2);
    });

    it('only skips repeating the command when the title ends with it', () => {
      const root = rolledUp();

      expect(barByName(root, 'install')?.sandbox?.commandInTitle).toBe(false);

      const cut = ciTestSteps().map((step) => ({
        ...step,
        name: step.name.replace(TEST_NAME, 'test › pnpm t…'),
      }));
      const cutRoot = traceRollup(
        createTrace({
          isRoot: true,
          childrenSpans: cut.map((step) => ({
            ...step,
            metadata: step.metadata?.map((md) => ({
              ...md,
              values: { ...md.values, statement_name: 'test › pnpm t…' },
            })),
          })) as Trace[],
        })
      );
      // CI cuts long labels with an ellipsis; a cut-off command still counts
      expect(barsOf(cutRoot)[0]?.sandbox?.commandInTitle).toBe(true);
    });

    it('has a verb for every statement the SDK emits', () => {
      // inngest-js components/sandbox/durable.ts statementForAction, plus the
      // explicit snapshot and snapshot.clone statements
      const sdkStatements = [
        'create',
        'list',
        'get',
        'waitUntilRunning',
        'commands.run',
        'destroy',
        'pause',
        'resume',
        'processes.start',
        'processes.list',
        'processes.get',
        'process.signal',
        'process.wait',
        'process.getOutput',
        'snapshot',
        'snapshots.list',
        'snapshots.get',
        'snapshot.waitUntilReady',
        'snapshot.delete',
        'snapshot.clone',
      ];

      expect(Object.keys(STATEMENT_VERBS).sort()).toEqual([...sdkStatements].sort());
      expect(statementVerb('pause')).toBe('Pause');
      expect(statementVerb('resume')).toBe('Resume');
      for (const statement of sdkStatements) {
        expect(statementVerb(statement)).toMatch(/^[A-Z]/);
      }
    });

    it("places a statement by its steps' starts, not a failed step's queue time", () => {
      // A real CI run: the start failed as ambiguous (no opcode) and carried
      // the run's queue time; CI reconciled it and carried on
      const steps = ciTestSteps();
      const start = steps[0]!;
      steps[0] = {
        ...start,
        stepOp: null,
        status: 'FAILED',
        queuedAt: '2026-10-01T12:00:00.000Z',
      } as Trace;
      const notify = createTrace({
        spanID: 'notify',
        stepID: 'notify',
        name: 'notify',
        attempts: 0,
        queuedAt: '2026-10-01T12:00:10.000Z',
        startedAt: '2026-10-01T12:00:10.000Z',
      });
      const root = traceRollup(createTrace({ isRoot: true, childrenSpans: [...steps, notify] }));

      expect(root.childrenSpans?.map((c) => c.name)).toEqual(['notify', TEST_NAME]);
      expect(root.childrenSpans![1]!.queuedAt).toBe('2026-10-01T12:00:38.000Z');
      expect(phaseSummary(root, TEST_NAME)?.[0]?.start).toBe('2026-10-01T12:00:38.000Z');
    });

    it('leaves runs without sandbox metadata exactly as before', () => {
      const plain = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({ spanID: 'a', stepID: 'step-a', attempts: 0 }),
          createTrace({
            spanID: 'b',
            stepID: 'step-b',
            attempts: 0,
            queuedAt: '2024-01-01T00:00:05Z',
          }),
        ],
      });
      const root = traceRollup(plain);

      expect(root).toEqual(plain);
      expect(barsOf(root).map((b) => b.sandbox)).toEqual([undefined, undefined]);
    });
  });
});
