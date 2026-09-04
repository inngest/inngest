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

/** Fixture steps are named `<branch>-<n>`, so an edge can be checked. */
const staysInBranch = (g: CanvasGraph): boolean => {
  const byID = new Map(g.nodes.map((n) => [n.id, n]));
  return g.nodes
    .filter((n) => n.parentNodeID)
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
    expect(edges(graph)).toEqual(['left-1 -> left-2', 'right-1 -> right-2']);
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

  it('shows an unobserved join member as an unconfirmed edge', () => {
    // `const [x] = await Promise.all([fork, bystander]); Promise.all([kid1, kid2])`
    // This fixture predates the SDK reporting a set, so it names one member —
    // true, since both really did gate the kids, but partial. `fork` is the
    // unnamed one, and rather than being dropped it is drawn broken: the
    // ordering allows it to have gated them, and nothing says it did.
    const graph = build(pathological);
    const byID = new Map(graph.nodes.map((n) => [n.id, n]));
    const fork = graph.nodes.find((n) => n.label === 'fork')!;
    const kids = graph.nodes.filter((n) => n.label.startsWith('fork-kid'));

    expect(kids.length).toBeGreaterThan(0);

    for (const kid of kids) {
      // The kids share a convergence, so fork reaches them through it.
      const viaJunction = graph.edges
        .filter((e) => e.to === kid.id && byID.get(e.from)?.kind === 'join')
        .flatMap((e) => graph.edges.filter((inner) => inner.to === e.from));

      const fromFork = [
        ...graph.edges.filter((e) => e.from === fork.id && e.to === kid.id),
        ...viaJunction,
      ].find((e) => e.from === fork.id);

      expect(fromFork?.kind).toBe('unconfirmed');

      // Whatever is drawn solid must be a real dependency. Every member of the
      // join is one, so the named parent is true even though it is not the
      // whole set.
      const solid = [...graph.edges.filter((e) => e.to === kid.id), ...viaJunction].filter(
        (e) => e.kind === 'sequence' || e.kind === 'fanIn'
      );
      for (const edge of solid) {
        const label = byID.get(edge.from)!.label;
        if (byID.get(edge.from)!.kind === 'join') continue;
        expect(['fork', 'bystander']).toContain(label);
      }
    }
  });

  it('never draws an edge that crosses branches, on any fixture', () => {
    // The safety property. Shapes whose steps are branch-prefixed must never
    // produce a cross-branch edge; the rest must simply not crash.
    for (const fixture of [balanced, v4chains, ragged, nestedInBranch, v4parallel]) {
      expect(staysInBranch(build(fixture))).toBe(true);
    }
  });

  it('has nothing to attribute for the first fan-out of a run', () => {
    // a, b and c wait on the trigger, not on a step, so the SDK reports no
    // parent for them.
    expect(build(v4parallel).branchesResolved).toBe(0);
  });
});
