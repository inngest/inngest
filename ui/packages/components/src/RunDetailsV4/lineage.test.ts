/**
 * Checked against `emit.json` and `invoke.json`, both real captures. The point
 * of these tests is the identification rule: the brief said a sent event is
 * `stepOp == SEND_EVENT`, and it is not — it is `stepType == 'step.sendEvent'`,
 * on a span whose `stepOp` is an ordinary `RUN`. That is exactly the kind of
 * thing that rots silently, so it is asserted against the payload.
 */
import { describe, expect, it } from 'vitest';

import emit from './canvas/__fixtures__/emit.json';
import invoke from './canvas/__fixtures__/invoke.json';
import step from './canvas/__fixtures__/step.json';
import { findEmittingSteps, findInvokedRuns, parseEmittedEventIDs } from './lineage';
import type { Trace } from './types';

const traceOf = (fixture: unknown) => (fixture as { run: { trace: Trace } }).run.trace;

describe('findEmittingSteps', () => {
  it('finds the step that sent events', () => {
    const found = findEmittingSteps(traceOf(emit));
    expect(found).toHaveLength(1);
    expect(found[0]!.label).toBe('fan-out');
    expect(found[0]!.outputID).toBeTruthy();
  });

  it('is not fooled into thinking every step sent something', () => {
    expect(findEmittingSteps(traceOf(step))).toHaveLength(0);
  });

  it('does not key on stepOp, which reports RUN for a sendEvent', () => {
    // The guard against reintroducing the brief's mistake. If this span ever
    // gains a SEND_EVENT stepOp the rule can change — but it should change
    // deliberately, not because someone assumed.
    let sendEventOps = 0;
    const walk = (span: Trace) => {
      if (span.stepOp === 'SEND_EVENT') sendEventOps += 1;
      span.childrenSpans?.forEach(walk);
    };
    walk(traceOf(emit));

    expect(sendEventOps).toBe(0);
    expect(findEmittingSteps(traceOf(emit))).toHaveLength(1);
  });
});

describe('findInvokedRuns', () => {
  it('finds the child run an invoke started', () => {
    const found = findInvokedRuns(traceOf(invoke));
    expect(found.length).toBeGreaterThan(0);
    expect(found[0]!.runID).toMatch(/^[0-9A-HJKMNP-TV-Z]{26}$/);
  });

  it('finds nothing in a run that invoked nothing', () => {
    expect(findInvokedRuns(traceOf(step))).toHaveLength(0);
  });
});

describe('parseEmittedEventIDs', () => {
  it('reads the ids a sendEvent reported', () => {
    // The real payload shape, verified against a live Dev Server.
    expect(
      parseEmittedEventIDs('{"ids":["01M1Q00QZAD5YFMY4TP9PH0YZ3","01M1Q00QZAD5YFMY4TP9PH0YZ4"]}')
    ).toEqual(['01M1Q00QZAD5YFMY4TP9PH0YZ3', '01M1Q00QZAD5YFMY4TP9PH0YZ4']);
  });

  it('says "cannot tell" rather than "sent nothing" for anything else', () => {
    // The distinction the whole module exists to preserve: an unreadable
    // payload must never be reported as an emitter that sent no events.
    for (const bad of [null, undefined, '', 'not json', '{}', '{"ids":"nope"}', '{"ids":[1,2]}']) {
      expect(parseEmittedEventIDs(bad), String(bad)).toBeNull();
    }
  });

  it('reports an genuinely empty send as empty, not unknown', () => {
    expect(parseEmittedEventIDs('{"ids":[]}')).toEqual([]);
  });
});
