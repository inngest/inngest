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

    it.each([
      ['inngest@3.44.0', 'COMPLETED', true],
      ['@inngest/ci@0.1.0', 'RUNNING', true],
      ['@inngest/ci@0.1.0', 'FAILED', false],
      ['@acme/inngest@1.0.0', 'COMPLETED', false],
      [null, 'COMPLETED', false],
    ])('dims a row from origin %s with status %s: %s', (origin, status, dimmed) => {
      const trace = createTrace({ isRoot: true, origin, status });
      const result = traceToTimelineData(trace, { runID: 'run-1' });

      expect(result.bars[0]?.dimmed).toBe(dimmed);
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

    it('gives span groups their own style, with no timing breakdown', () => {
      const trace = createTrace({
        isRoot: true,
        childrenSpans: [
          createTrace({
            spanID: 'group',
            stepID: null,
            stepOp: null,
            stepType: 'SPAN_GROUP',
            childrenSpans: [
              createTrace({ spanID: 'run' }),
              createTrace({ spanID: 'wait', stepOp: 'WAIT_FOR_SIGNAL' }),
            ],
          }),
        ],
      });
      const group = traceToTimelineData(trace, { runID: 'run-1' }).bars[0]?.children?.[0];

      expect(group?.style).toBe('span.group');
      expect(group?.timingBreakdown).toBeUndefined();
      expect(group?.inngestBreakdown).toBeUndefined();
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

    it("counts only span groups' step execution in root bar execution time", () => {
      const group = (childrenSpans: Trace[]) => {
        return createTrace({ stepID: null, stepOp: null, stepType: 'SPAN_GROUP', childrenSpans });
      };
      const trace = createTrace({
        isRoot: true,
        queuedAt: '2024-01-01T00:00:00Z',
        startedAt: '2024-01-01T00:00:01Z',
        endedAt: '2024-01-01T00:00:20Z',
        childrenSpans: [
          group([
            createTrace({
              queuedAt: '2024-01-01T00:00:01Z',
              startedAt: '2024-01-01T00:00:02Z',
              endedAt: '2024-01-01T00:00:03Z',
            }),
            group([
              createTrace({
                queuedAt: '2024-01-01T00:00:10Z',
                startedAt: '2024-01-01T00:00:11Z',
                endedAt: '2024-01-01T00:00:13Z',
              }),
            ]),
          ]),
        ],
      });
      const rootBar = traceToTimelineData(trace, { runID: 'run-1' }).bars[0];

      // 1s + 2s of steps, not the group's 12s wall clock
      expect(rootBar?.timingBreakdown?.executionMs).toBe(3000);
      expect(rootBar?.timingBreakdown?.inngestMs).toBe(17000);
      expect(rootBar?.runInngestBreakdown?.finalizationMs).toBe(7000);
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

  describe('warnings', () => {
    const warningsMd = (message: string): NonNullable<Trace['metadata']> => {
      return [
        {
          scope: 'step',
          kind: 'inngest.warnings',
          updatedAt: '2024-01-01T00:00:03Z',
          values: { w: message },
        },
      ] as NonNullable<Trace['metadata']>;
    };

    it('exposes inngest.warnings metadata as bar.warnings', () => {
      const root = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ spanID: 'c1', metadata: warningsMd('careful') })],
      });
      const result = traceToTimelineData(root, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.warnings).toEqual([{ key: 'w', message: 'careful' }]);
    });

    it('exposes per-code inngest.warning.<code> metadata as bar.warnings (icon source)', () => {
      const perCode = [
        {
          scope: 'step',
          kind: 'inngest.warning.dynamic_step',
          updatedAt: '2024-01-01T00:00:03Z',
          values: { dynamic_step: 'per-code message' },
        },
        {
          scope: 'step',
          kind: 'inngest.warnings',
          updatedAt: '2024-01-01T00:00:01Z',
          values: { legacy_code: 'legacy message' },
        },
      ] as NonNullable<Trace['metadata']>;
      const root = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ spanID: 'c1', metadata: perCode })],
      });
      const result = traceToTimelineData(root, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.warnings).toEqual([
        { key: 'dynamic_step', message: 'per-code message' },
        { key: 'legacy_code', message: 'legacy message' },
      ]);
    });

    it('leaves bar.warnings undefined without the metadata', () => {
      const root = createTrace({
        isRoot: true,
        childrenSpans: [createTrace({ spanID: 'c1' })],
      });
      const result = traceToTimelineData(root, { runID: 'run-1' });

      expect(result.bars[0]?.children?.[0]?.warnings).toBeUndefined();
    });

    it('carries the last attempt warnings on a rolled-up step', () => {
      const attempt0 = createTrace({
        spanID: 'a0',
        stepID: 'step-1',
        attempts: 0,
        queuedAt: '2024-01-01T00:00:00Z',
        metadata: warningsMd('first attempt'),
      });
      const attempt1 = createTrace({
        spanID: 'a1',
        stepID: 'step-1',
        attempts: 1,
        queuedAt: '2024-01-01T00:00:02Z',
        metadata: warningsMd('last attempt'),
      });
      const root = createTrace({ isRoot: true, childrenSpans: [attempt0, attempt1] });
      const result = traceToTimelineData(traceRollup(root), { runID: 'run-1' });

      const rollupBar = result.bars[0]?.children?.[0];
      expect(rollupBar?.warnings).toEqual([{ key: 'w', message: 'last attempt' }]);
    });

    it("keeps an earlier attempt's warning when the retry that succeeds has none", () => {
      const attempt0 = createTrace({
        spanID: 'a0',
        stepID: 'step-1',
        attempts: 0,
        status: 'FAILED',
        queuedAt: '2024-01-01T00:00:00Z',
        metadata: warningsMd('built while this job waited'),
      });
      const attempt1 = createTrace({
        spanID: 'a1',
        stepID: 'step-1',
        attempts: 1,
        queuedAt: '2024-01-01T00:00:02Z',
      });
      const root = createTrace({ isRoot: true, childrenSpans: [attempt0, attempt1] });
      const result = traceToTimelineData(traceRollup(root), { runID: 'run-1' });

      const rollupBar = result.bars[0]?.children?.[0];
      expect(rollupBar?.warnings).toEqual([{ key: 'w', message: 'built while this job waited' }]);
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

    it("keeps a retried step's origin, so a library's step stays dimmed and credited", () => {
      const origin = '@inngest/ci@0.1.0';

      const attempts = [0, 1].map((attempt) => {
        return createTrace({
          spanID: `a${attempt}`,
          stepID: 'step-1',
          attempts: attempt,
          origin,
          status: attempt === 0 ? 'FAILED' : 'COMPLETED',
        });
      });

      const fin = [0, 1].map((attempt) => {
        return createTrace({
          spanID: `fin-${attempt}`,
          stepID: null,
          groupID: 'g-final',
          attempts: attempt,
          outputID: 'out-fin',
          origin,
          queuedAt: '2024-01-01T00:00:11Z',
          endedAt: '2024-01-01T00:00:12Z',
        });
      });

      const root = createTrace({ isRoot: true, childrenSpans: [...attempts, ...fin] });

      const result = traceRollup(root);

      expect(result.childrenSpans?.map((c) => c.spanID)).toEqual(['step-1-rollup', 'final-rollup']);
      expect(result.childrenSpans?.map((c) => c.origin)).toEqual([origin, origin]);

      const timeline = traceToTimelineData(result, { runID: 'run-1' });

      expect(timeline.bars[0]?.children?.[0]?.dimmed).toBe(true);
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

  describe('traceRollup — span groups', () => {
    const ts = (secs: number) => `2024-01-01T00:00:${String(secs).padStart(2, '0')}Z`;
    const attempt = (spanID: string, stepID: string, attempts: number, secs: number) =>
      createTrace({
        spanID,
        stepID,
        attempts,
        name: stepID,
        queuedAt: ts(secs),
        startedAt: ts(secs),
        endedAt: ts(secs + 1),
        status: attempts === 0 ? 'FAILED' : 'COMPLETED',
      });

    const group = (spanID: string, childrenSpans: Trace[]) =>
      createTrace({
        spanID,
        name: spanID,
        stepID: null,
        stepOp: null,
        stepType: 'SPAN_GROUP',
        queuedAt: childrenSpans[0]?.queuedAt ?? ts(0),
        // The server ends a group with its last child
        endedAt: childrenSpans[childrenSpans.length - 1]?.endedAt ?? null,
        childrenSpans,
      });

    // A retried step, an ordinary step and a trailing finalization
    const runChildren = () => [
      attempt('a0', 'retried', 0, 1),
      attempt('a1', 'retried', 1, 3),
      attempt('b0', 'plain', 0, 5),
      createTrace({
        spanID: 'final',
        stepID: null,
        stepOp: null,
        groupID: 'g-final',
        attempts: 0,
        outputID: 'o-final',
        queuedAt: '2024-01-01T00:00:08Z',
      }),
    ];

    it('leaves the rest of the run exactly as it is without groups', () => {
      const without = traceRollup(createTrace({ isRoot: true, childrenSpans: runChildren() }));
      const withGroup = traceRollup(
        createTrace({
          isRoot: true,
          // Grouped work that ends before the run's last step does
          childrenSpans: [...runChildren(), group('g', [attempt('c0', 'grouped', 0, 4)])],
        })
      );

      expect(withGroup.childrenSpans?.filter((c) => c.spanID !== 'g')).toEqual(
        without.childrenSpans
      );
    });

    it('keeps groups and rolls up retries inside them, at any depth', () => {
      const steps = () => [attempt('a0', 'retried', 0, 1), attempt('a1', 'retried', 1, 3)];
      const atRoot = traceRollup(createTrace({ isRoot: true, childrenSpans: steps() }));
      const nested = traceRollup(
        createTrace({
          isRoot: true,
          childrenSpans: [group('outer', [group('inner', steps())])],
        })
      );

      const outer = nested.childrenSpans?.[0];
      expect(outer?.spanID).toBe('outer');
      const inner = outer?.childrenSpans?.[0];
      expect(inner?.spanID).toBe('inner');
      expect(inner?.childrenSpans).toEqual(atRoot.childrenSpans);
      expect(inner?.childrenSpans?.[0]?.childrenSpans?.map((c) => c.name)).toEqual([
        'Attempt 0',
        'Attempt 1',
      ]);
    });

    it('orders groups among the steps by queue time', () => {
      const result = traceRollup(
        createTrace({
          isRoot: true,
          childrenSpans: [
            attempt('late', 'late', 1, 9),
            group('g', [attempt('c0', 'grouped', 1, 4)]),
            attempt('early', 'early', 1, 1),
          ],
        })
      );

      expect(result.childrenSpans?.map((c) => c.spanID)).toEqual(['early', 'g', 'late']);
    });

    describe('finalization start', () => {
      // A step's span; `to` of null is a step that hasn't ended
      const work = (spanID: string, from: number, to: number | null, attempts = 0) =>
        createTrace({
          spanID,
          stepID: spanID,
          attempts,
          name: spanID,
          queuedAt: ts(from),
          startedAt: ts(from),
          endedAt: to === null ? null : ts(to),
        });

      // A group ending at `to`, however its children end
      const endingAt = (to: number | null, trace: Trace): Trace => {
        trace.endedAt = to === null ? null : ts(to);

        return trace;
      };

      // Queued at 5s and started at 6s, while the work above is still going
      const finalization = () =>
        createTrace({
          spanID: 'fin',
          stepID: null,
          stepOp: null,
          groupID: 'g-final',
          attempts: 0,
          outputID: 'o-final',
          queuedAt: ts(5),
          startedAt: ts(6),
          endedAt: ts(20),
        });

      const cases: {
        name: string;
        children: () => Trace[];
        /** Seconds the finalization is expected to queue and start at */
        expected: [number, number];
      }[] = [
        {
          name: 'no groups: waits for the last step',
          children: () => [work('s1', 0, 2), work('s2', 3, 9)],
          expected: [9, 9],
        },
        {
          name: 'no groups: leaves a finalization that starts after the last step alone',
          children: () => [work('s1', 0, 4)],
          expected: [5, 6],
        },
        {
          name: 'mixed: ungrouped last step ends after the grouped work',
          children: () => [work('s1', 0, 9), group('g', [work('c1', 3, 6)])],
          expected: [9, 9],
        },
        {
          name: 'mixed: ungrouped last step ends before the grouped work',
          children: () => [work('s1', 0, 2), group('g', [work('c1', 3, 10)])],
          expected: [10, 10],
        },
        {
          name: 'all steps grouped: one group',
          children: () => [group('g', [work('c1', 3, 10)])],
          expected: [10, 10],
        },
        {
          name: 'all steps grouped: sibling groups, the latest end wins',
          children: () => [
            group('g1', [work('c1', 1, 10)]),
            group('g2', [work('c2', 2, 12)]),
            group('g3', [work('c3', 3, 7)]),
          ],
          expected: [12, 12],
        },
        {
          name: 'all steps grouped: a group listed first can end last',
          children: () => [group('g1', [work('c1', 1, 12)]), group('g2', [work('c2', 2, 7)])],
          expected: [12, 12],
        },
        {
          name: 'nested groups: the deepest step ends latest',
          children: () => [
            endingAt(
              8,
              group('outer', [
                work('c1', 1, 4),
                endingAt(6, group('inner', [work('c2', 2, 5), work('c3', 3, 10)])),
              ])
            ),
          ],
          expected: [10, 10],
        },
        {
          name: 'a group ending later than its children counts as work',
          children: () => [endingAt(11, group('g', [work('c1', 3, 8)]))],
          expected: [11, 11],
        },
        {
          name: 'in progress: a child that has not ended is ignored, not treated as the start of time',
          children: () => [endingAt(null, group('g', [work('c1', 1, 8), work('c2', 2, null)]))],
          expected: [8, 8],
        },
        {
          name: 'in progress: nothing has ended, so the finalization is untouched',
          children: () => [endingAt(null, group('g', [work('c1', 1, null)]))],
          expected: [5, 6],
        },
        {
          name: 'retries in a group: a failed attempt ending latest counts',
          children: () => [
            endingAt(
              8,
              group('g', [
                // The rollup ends with attempt 1, but attempt 0 ended later
                work('c1', 2, 9, 0),
                work('c1', 3, 7, 1),
              ])
            ),
          ],
          expected: [9, 9],
        },
        {
          name: 'an empty group with no end is ignored',
          children: () => [endingAt(null, group('g', [])), work('s1', 0, 4)],
          expected: [5, 6],
        },
        {
          name: 'an empty group with an end counts',
          children: () => [endingAt(8, group('g', []))],
          expected: [8, 8],
        },
      ];

      it.each(cases)('$name', ({ children, expected }) => {
        const result = traceRollup(
          createTrace({ isRoot: true, childrenSpans: [...children(), finalization()] })
        );
        const fin = result.childrenSpans?.find((c) => c.spanID === 'fin');

        expect(fin?.name).toBe('Finalization');
        expect(fin?.queuedAt).toBe(ts(expected[0]));
        expect(fin?.startedAt).toBe(ts(expected[1]));
        expect(fin?.endedAt).toBe(ts(20));
      });

      it('clamps every attempt of a multi-attempt finalization to grouped work', () => {
        const attempts = [0, 1].map((n) => {
          return createTrace({
            spanID: `fin-${n}`,
            stepID: null,
            stepOp: null,
            groupID: 'g-final',
            attempts: n,
            outputID: 'o-final',
            queuedAt: ts(5 + n),
            startedAt: ts(6 + n),
            endedAt: ts(20),
          });
        });
        const result = traceRollup(
          createTrace({
            isRoot: true,
            childrenSpans: [group('g', [work('c1', 3, 10)]), ...attempts],
          })
        );
        const fin = result.childrenSpans?.find((c) => c.spanID === 'final-rollup');

        expect(fin?.queuedAt).toBe(ts(10));
        expect(fin?.startedAt).toBe(ts(10));
      });

      it('has no finalization to place when the run has none', () => {
        const result = traceRollup(
          createTrace({ isRoot: true, childrenSpans: [group('g', [work('c1', 3, 10)])] })
        );

        expect(result.childrenSpans?.map((c) => c.spanID)).toEqual(['g']);
      });
    });
  });
});
