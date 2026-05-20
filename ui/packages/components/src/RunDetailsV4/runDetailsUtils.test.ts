import { describe, expect, it } from 'vitest';

import { collectInvokedRuns } from './runDetailsUtils';
import type { StepInfoInvoke, Trace } from './types';

function makeTrace(overrides: Partial<Trace> = {}): Trace {
  return {
    attempts: null,
    endedAt: null,
    isRoot: false,
    name: 'test-step',
    outputID: null,
    queuedAt: '2026-01-01T00:00:00Z',
    spanID: 'span-1',
    startedAt: null,
    status: 'COMPLETED',
    stepInfo: null,
    userlandSpan: null,
    isUserland: false,
    ...overrides,
  };
}

function makeInvokeStepInfo(overrides: Partial<StepInfoInvoke> = {}): StepInfoInvoke {
  return {
    triggeringEventID: '01TRIGGER',
    functionID: 'invoked-fn',
    timeout: '2026-12-01T00:00:00Z',
    returnEventID: null,
    runID: '01INVOKED01',
    timedOut: false,
    ...overrides,
  };
}

describe('collectInvokedRuns', () => {
  it('returns [] for undefined trace', () => {
    expect(collectInvokedRuns(undefined)).toEqual([]);
  });

  it('returns [] when there are no invoke spans', () => {
    const trace = makeTrace({
      isRoot: true,
      childrenSpans: [makeTrace({ spanID: 'a' }), makeTrace({ spanID: 'b' })],
    });
    expect(collectInvokedRuns(trace)).toEqual([]);
  });

  it('collects nested invokes', () => {
    const trace = makeTrace({
      isRoot: true,
      childrenSpans: [
        makeTrace({
          spanID: 'parent',
          childrenSpans: [
            makeTrace({
              spanID: 'invoke-1',
              name: 'parent.step.invoke',
              status: 'COMPLETED',
              stepInfo: makeInvokeStepInfo({
                runID: '01INVOKED01',
                functionID: 'invoked-fn-1',
              }),
            }),
            makeTrace({
              spanID: 'wrapper',
              childrenSpans: [
                makeTrace({
                  spanID: 'invoke-2',
                  name: 'nested.step.invoke',
                  status: 'RUNNING',
                  stepInfo: makeInvokeStepInfo({
                    runID: '01INVOKED02',
                    functionID: 'invoked-fn-2',
                  }),
                }),
              ],
            }),
          ],
        }),
      ],
    });

    const got = collectInvokedRuns(trace);
    expect(got).toHaveLength(2);
    expect(got).toEqual([
      {
        spanID: 'invoke-1',
        invokerName: 'parent.step.invoke',
        functionID: 'invoked-fn-1',
        runID: '01INVOKED01',
        status: 'COMPLETED',
      },
      {
        spanID: 'invoke-2',
        invokerName: 'nested.step.invoke',
        functionID: 'invoked-fn-2',
        runID: '01INVOKED02',
        status: 'RUNNING',
      },
    ]);
  });

  it('skips invokes without a runID', () => {
    const trace = makeTrace({
      isRoot: true,
      childrenSpans: [
        makeTrace({
          spanID: 'invoke-no-run',
          stepInfo: makeInvokeStepInfo({ runID: null }),
        }),
        makeTrace({
          spanID: 'invoke-with-run',
          stepInfo: makeInvokeStepInfo({ runID: '01HASRUN01' }),
        }),
      ],
    });

    const got = collectInvokedRuns(trace);
    expect(got).toHaveLength(1);
    expect(got[0]?.spanID).toBe('invoke-with-run');
  });
});
