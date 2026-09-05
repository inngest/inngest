/**
 * Table-driven tests over payloads captured from real Dev Server runs against
 * the `tests/js` fixture app. See __fixtures__/README.md for how to recapture.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import chains from './__fixtures__/chains.json';
import failure from './__fixtures__/failure.json';
import inflight from './__fixtures__/inflight.json';
import invoke from './__fixtures__/invoke.json';
import loop40 from './__fixtures__/loop40.json';
import loop from './__fixtures__/loop.json';
import noretry from './__fixtures__/noretry.json';
import parallelInferred from './__fixtures__/parallel-inferred.json';
import parallel from './__fixtures__/parallel.json';
import retry from './__fixtures__/retry.json';
import simple from './__fixtures__/simple.json';
import step from './__fixtures__/step.json';
import v4chains from './__fixtures__/v4chains.json';
import v4parallel from './__fixtures__/v4parallel.json';
import v4sequential from './__fixtures__/v4sequential.json';
import wait from './__fixtures__/wait.json';
import waitmatched from './__fixtures__/waitmatched.json';
import waittimeout from './__fixtures__/waittimeout.json';
import wide from './__fixtures__/wide.json';
import { RESULT_ID, eventID, toCanvasGraph } from './graph';
import type { CanvasGraph } from './graph.types';

type Fixture = { run: { status: string; trace: unknown } };

function build(fixture: unknown): CanvasGraph {
  const root = (fixture as Fixture).run.trace as Trace;
  return toCanvasGraph(traceRollup(root));
}

/** Work labels per level, excluding the synthetic event/join/result nodes. */
function shape(graph: CanvasGraph): string[][] {
  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  const WORK = new Set(['step', 'invoke', 'wait']);
  return graph.levels
    .map((ids) =>
      ids
        .map((id) => byID.get(id)!)
        .filter((n) => WORK.has(n.kind))
        .map((n) => n.label)
        .sort()
    )
    .filter((level) => level.length > 0);
}

describe('toCanvasGraph', () => {
  describe('shape', () => {
    const cases: Array<{ name: string; fixture: unknown; expected: string[][] }> = [
      {
        name: 'a linear run puts every step on its own level',
        fixture: step,
        expected: [['first step'], ['for 2s'], ['second step']],
      },
      {
        name: 'a fan-out puts concurrent steps on one level and the join after it',
        fixture: parallel,
        expected: [['a', 'b', 'c'], ['d']],
      },
      {
        name: 'parallel branches of chains level up pairwise',
        fixture: chains,
        expected: [['left-1', 'right-1'], ['left-2', 'right-2'], ['join']],
      },
      {
        name: 'a wide fan-out stays a single level',
        fixture: wide,
        expected: [
          ['w0', 'w1', 'w10', 'w11', 'w2', 'w3', 'w4', 'w5', 'w6', 'w7', 'w8', 'w9'],
          ['collect'],
        ],
      },
      {
        name: 'a retry collapses to one node',
        fixture: retry,
        expected: [['first step']],
      },
      {
        name: 'a failure mid-run stops at the failed step',
        fixture: failure,
        expected: [['ok step'], ['doomed step']],
      },
      {
        name: 'an invoke is its own node between its neighbours',
        fixture: invoke,
        expected: [['before'], ['call child'], ['after']],
      },
      {
        name: 'a run with no steps has no work levels',
        fixture: simple,
        expected: [],
      },
      {
        name: 'a single failed step with no retry is one level',
        fixture: noretry,
        expected: [['first step']],
      },
      {
        name: 'a waitForEvent is its own node',
        fixture: wait,
        expected: [['test/resume']],
      },
      {
        name: 'a timed-out waitForEvent is its own node',
        fixture: waittimeout,
        expected: [['never arrives']],
      },
    ];

    for (const { name, fixture, expected } of cases) {
      it(name, () => {
        expect(shape(build(fixture))).toEqual(expected);
      });
    }
  });

  describe('endpoints', () => {
    it('always starts with a trigger and ends with a result', () => {
      for (const fixture of [simple, step, parallel, failure, inflight, wait]) {
        const graph = build(fixture);
        expect(graph.levels[0]).toEqual([eventID(0)]);
        expect(graph.levels.at(-1)).toEqual([RESULT_ID]);
      }
    });

    it('carries the run status onto the result node', () => {
      expect(build(failure).nodes.find((n) => n.id === RESULT_ID)!.status).toBe('FAILED');
      expect(build(parallel).nodes.find((n) => n.id === RESULT_ID)!.status).toBe('COMPLETED');
      expect(build(inflight).nodes.find((n) => n.id === RESULT_ID)!.status).toBe('RUNNING');
    });

    it('points synthetic nodes at the run root so they open the run panel', () => {
      const graph = build(parallel);
      const rootSpanID = (parallel as Fixture).run.trace as Trace;
      for (const id of [eventID(0), RESULT_ID]) {
        expect(graph.nodes.find((n) => n.id === id)!.spanID).toBe(rootSpanID.spanID);
      }
    });
  });

  describe('waits are nodes, not edges', () => {
    it('models a sleep as its own node between its neighbours', () => {
      const graph = build(step);
      const sleep = graph.nodes.find((n) => n.kind === 'wait')!;
      expect(sleep.label).toBe('for 2s');
      expect(sleep.waitKind).toBe('sleep');
      expect(sleep.stepOp).toBe('SLEEP');
      // The fixture sleeps for 2s; allow slack for scheduling.
      const duration = sleep.endedAt! - (sleep.startedAt ?? sleep.queuedAt);
      expect(duration).toBeGreaterThanOrEqual(1900);
      expect(duration).toBeLessThan(3000);
    });

    it('marks a matched waitForEvent as not timed out', () => {
      const wf = build(waitmatched).nodes.find((n) => n.kind === 'wait')!;
      expect(wf.waitKind).toBe('waitForEvent');
      expect(wf.waitTimedOut).toBe(false);
    });

    it('marks an expired waitForEvent as timed out', () => {
      const wf = build(waittimeout).nodes.find((n) => n.kind === 'wait')!;
      expect(wf.waitTimedOut).toBe(true);
    });
  });

  describe('timeouts are neither success nor failure', () => {
    it('reports an expired wait as timed out while its span status stays COMPLETED', () => {
      // The trace draws no distinction here — the span really is COMPLETED — so
      // the canvas carries the timeout separately and greys the node.
      const wf = build(waittimeout).nodes.find((n) => n.kind === 'wait')!;
      expect(wf.status).toBe('COMPLETED');
      expect(wf.waitTimedOut).toBe(true);
    });
  });

  describe('fan-out topology', () => {
    it('puts a junction at each end of a parallel level', () => {
      // One where the run widens and one where it narrows: the two points where
      // the SDK made a discovery and the shape of the run changed.
      const graph = build(parallel);
      expect(graph.edges.filter((e) => e.kind === 'fanOut')).toHaveLength(3);
      expect(graph.edges.filter((e) => e.kind === 'fanIn')).toHaveLength(3);

      const junctions = graph.nodes.filter((n) => n.kind === 'join');
      expect(junctions).toHaveLength(2);

      // One that the run widens through, one it narrows through.
      const widens = junctions.filter((j) => graph.edges.filter((e) => e.from === j.id).length > 1);
      const narrows = junctions.filter((j) => graph.edges.filter((e) => e.to === j.id).length > 1);
      expect(widens).toHaveLength(1);
      expect(narrows).toHaveLength(1);

      // The widening one sits between the trigger and the parallel level rather
      // than on either of them.
      expect(Number.isInteger(widens[0]!.level)).toBe(false);
    });

    it('does not emit a join for a linear run', () => {
      expect(build(step).nodes.filter((n) => n.kind === 'join')).toHaveLength(0);
    });

    it('every edge references nodes that exist', () => {
      for (const fixture of [step, parallel, chains, wide, invoke, loop, failure, inflight]) {
        const graph = build(fixture);
        const ids = new Set(graph.nodes.map((n) => n.id));
        for (const edge of graph.edges) {
          expect(ids.has(edge.from), `${edge.id} from`).toBe(true);
          expect(ids.has(edge.to), `${edge.id} to`).toBe(true);
        }
      }
    });

    it('warns only when a parallel level was inferred', () => {
      expect(build(parallelInferred).warnings.join(' ')).toMatch(
        /inferred from observed execution overlap/
      );
      expect(build(step).warnings.join(' ')).not.toMatch(/inferred/);
    });
  });

  describe('authoritative parallelism (SDK execution version 2+)', () => {
    it('groups a fan-out from the SDK plan rather than from overlap', () => {
      const graph = build(v4parallel);
      expect(graph.parallelismSource).toBe('sdk');
      expect(shape(graph)).toEqual([['a', 'b', 'c'], ['d']]);
      expect(graph.warnings.join(' ')).not.toMatch(/inferred/);
    });

    it('groups parallel chains pairwise from the plans', () => {
      const graph = build(v4chains);
      expect(graph.parallelismSource).toBe('sdk');
      expect(shape(graph)).toEqual([['left-1', 'right-1'], ['left-2', 'right-2'], ['join']]);
    });

    it('reports no parallelism for a sequential v4 run, and does not warn', () => {
      const graph = build(v4sequential);
      expect(graph.parallelismSource).toBe('none');
      expect(shape(graph)).toEqual([['first step'], ['for 2s'], ['second step']]);
      expect(graph.warnings).toEqual([]);
    });

    it('falls back to inference, and says so, when no plan was reported', () => {
      // `parallel-inferred` is deliberately a pre-loader capture: once the
      // loader began lifting plannedSteps off discovery spans, even v1-SDK runs
      // started reporting exact groups, and nothing else in the set exercises
      // the fallback any more.
      const graph = build(parallelInferred);
      expect(graph.parallelismSource).toBe('inferred');
      expect(shape(graph)).toEqual([['a', 'b', 'c'], ['d']]);
      expect(graph.warnings.join(' ')).toMatch(/inngest-js v4\+/);
    });
  });

  describe('retries', () => {
    it('keeps the attempt count on the node instead of adding nodes', () => {
      const graph = build(retry);
      const node = graph.nodes.find((n) => n.label === 'first step')!;
      expect(node.attempts).toBe(1);
      expect(node.status).toBe('COMPLETED');
    });
  });

  describe('in-flight runs', () => {
    it('models a running run with a null maxTime and unfinished nodes', () => {
      const graph = build(inflight);
      expect(graph.runStatus).toBe('RUNNING');
      expect(graph.maxTime).toBeNull();
      expect(graph.nodes.some((n) => n.endedAt === null && n.kind === 'step')).toBe(true);
    });
  });

  describe('loops', () => {
    it('renders an 8-iteration agent loop as 16 sequential levels', () => {
      // Recorded so the readability problem is visible in the test output, not
      // just on screen. See the loops note in tasks/canvas/todo.md.
      expect(shape(build(loop))).toHaveLength(16);
    });

    it('renders a 40-iteration agent loop as 80 sequential levels', () => {
      expect(shape(build(loop40))).toHaveLength(80);
    });
  });
});

describe('what recapturing changed', () => {
  it('still has one capture that exercises the inference fallback', () => {
    // Recapturing the fixtures against a current Dev Server made every v3 run
    // report exact groups, because the loader now lifts `plannedSteps` off
    // discovery spans. That is better for users and worse for coverage: the
    // inference path in `toLevels` lost its last fixture.
    //
    // `parallel-inferred` is the preserved pre-loader capture. If this ever
    // fails, the fallback is untested and something has to be recaptured or
    // synthesised deliberately rather than noticed later.
    expect(build(parallelInferred).parallelismSource).toBe('inferred');
  });

  it('reports exact grouping for a v3 run now that plans reach the client', () => {
    // The improvement, asserted so it is not mistaken for a regression: this
    // run used to be 'inferred' and is now 'sdk'.
    expect(build(parallel).parallelismSource).toBe('sdk');
    expect(shape(build(parallel))).toEqual([['a', 'b', 'c'], ['d']]);
  });
});
