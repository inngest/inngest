/**
 * Branch membership: which step each step continues.
 *
 * `parentStepID` is reported by the SDK (`opts.discoveredAfter`), not inferred
 * downstream — see the note on resolveBranches in graph.ts for the two
 * inference attempts that came before and why both were unsound.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import nestedInBranch from './__fixtures__/gnarly-nested-in-branch.json';
import parallelInferred from './__fixtures__/parallel-inferred.json';
import balanced from './__fixtures__/v4branches-balanced.json';
import ragged from './__fixtures__/v4branches-ragged.json';
import v4chains from './__fixtures__/v4chains.json';
import v4parallel from './__fixtures__/v4parallel.json';
import pathological from './__fixtures__/v4pathological.json';
import { toCanvasGraph } from './graph';
import type { CanvasGraph } from './graph.types';

function build(fixture: unknown): CanvasGraph {
  return toCanvasGraph(traceRollup((fixture as { run: { trace: Trace } }).run.trace));
}

const edges = (g: CanvasGraph): string[] => {
  const byID = new Map(g.nodes.map((n) => [n.id, n]));
  return g.nodes
    .filter((n) => n.parentNodeID)
    .map((n) => `${byID.get(n.parentNodeID!)!.label} -> ${n.label}`)
    .sort();
};

/**
 * Fixture steps in a branch are named `<branch>-<n>`, so an edge can be checked.
 *
 * Only pairs where BOTH ends are branch-prefixed. A branch converging into a
 * join is named for the join, not the branch, so `right-2 -> join` and `b -> d`
 * are correct edges that a bare `split('-')` reads as crossing.
 */
const BRANCHED = /^[a-z]+\d+-\d+$/;
const staysInBranch = (g: CanvasGraph): boolean => {
  const byID = new Map(g.nodes.map((n) => [n.id, n]));
  return g.nodes
    .filter((n) => n.parentNodeID)
    .filter((n) => BRANCHED.test(n.label) && BRANCHED.test(byID.get(n.parentNodeID!)!.label))
    .every((n) => byID.get(n.parentNodeID!)!.label.split('-')[0] === n.label.split('-')[0]);
};

describe('branch membership', () => {
  it('follows each branch of a balanced fan-out', () => {
    const graph = build(balanced);
    expect(graph.branchesResolved).toBeGreaterThan(0);
    expect(staysInBranch(graph)).toBe(true);
  });

  it('follows parallel chains', () => {
    const graph = build(v4chains);
    // Recaptured against a current server, the join's own parent is reported
    // too — one more resolved edge than when this fixture was first taken.
    expect(edges(graph)).toEqual(['left-1 -> left-2', 'right-1 -> right-2', 'right-2 -> join']);
  });

  it('draws a fan-out from inside a branch as a real DAG', () => {
    // One branch does Promise.all([f-2a, f-2b]) while the other runs p-2. The
    // level maps two children to one parent and one child to another — not a
    // bijection, and correct. A single-parent-per-level model could not express
    // this at all.
    expect(edges(build(nestedInBranch))).toEqual(['f-1 -> f-2a', 'f-1 -> f-2b', 'p-1 -> p-2']);
  });

  it('draws nothing out of a branch that simply ended', () => {
    // br2 stops while five other branches continue one step each. Nothing
    // reported it as a parent and the level does not converge, so it is a
    // branch that ended — not a join we failed to observe — and it gets no
    // outgoing edge of any kind.
    const graph = build(ragged);
    const byID = new Map(graph.nodes.map((n) => [n.id, n]));
    const ended = graph.nodes.find((n) => n.label === 'br2-1')!;

    expect(graph.edges.filter((e) => e.from === ended.id)).toEqual([]);
    expect(graph.edges.some((e) => byID.get(e.to)?.label === 'br1-2' && e.from === ended.id)).toBe(
      false
    );
  });

  it('never draws an edge that crosses branches, on any fixture', () => {
    // The safety property. Shapes whose steps are branch-prefixed must never
    // produce a cross-branch edge; the rest must simply not crash.
    for (const fixture of [balanced, v4chains, ragged, nestedInBranch, v4parallel]) {
      expect(staysInBranch(build(fixture))).toBe(true);
    }
  });

  it('has nothing to attribute when the SDK reported no parents', () => {
    // `parallel-inferred` is kept for this. `v4parallel` used to serve here and
    // no longer can: recaptured against a current server it reports a parent
    // for the join, which is the point of recapturing it.
    expect(build(parallelInferred).branchesResolved).toBe(0);
  });
});
