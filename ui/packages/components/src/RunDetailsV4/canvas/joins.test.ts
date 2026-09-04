/**
 * Joins: a step that waited for several others.
 *
 * `parentStepIDs` is reported by the SDK (`opts.discoveredAfter`), which sees
 * the promise combinator being built and names every member of it. Before that,
 * only a single parent was reported — true but partial on a join, since
 * `await Promise.all([a, b]); step.run(c)` names whichever of a and b finished
 * last and leaves the other looking like a dead end.
 *
 * `v4parallel` is a real captured run of exactly that shape: a, b and c in
 * parallel, then d. It was captured before the SDK could report the join, so d
 * carries no parents at all — which makes it the honest fixture to test both
 * what the new data draws and how the old data still degrades.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import v4parallel from './__fixtures__/v4parallel.json';
import { toCanvasGraph } from './graph';
import type { CanvasGraph } from './graph.types';

type Fixture = { run: { trace: Trace } };

const stepIDs: Record<string, string> = {
  a: '86f7e437faa5a7fce15d1ddcb9eaeaea377667b8',
  b: 'e9d71f5ee7c92d6dc9e92ffdad17b8bd49418f98',
  c: '84a516841ba77a5b4648de2cd0dfcb30ea46dbb4',
};

/**
 * Copies the fixture and fills in the lineage the current SDK would report for
 * `d`. Everything else about the payload is left exactly as captured.
 */
function withLineage(parents: string[], alternates?: string[]): CanvasGraph {
  const fixture = structuredClone(v4parallel as unknown as Fixture);

  const stamp = (span: Trace) => {
    if (span.name === 'd') {
      span.parentStepIDs = parents.map((name) => stepIDs[name]!);
      if (alternates) {
        span.parentAlternateStepIDs = alternates.map((name) => stepIDs[name]!);
      }
    }
    for (const child of span.childrenSpans ?? []) stamp(child as Trace);
  };

  stamp(fixture.run.trace);

  return toCanvasGraph(traceRollup(fixture.run.trace));
}

const labelled = (graph: CanvasGraph) => {
  const byID = new Map(graph.nodes.map((node) => [node.id, node]));
  const name = (id: string) => {
    const node = byID.get(id);
    if (!node) return id;
    return node.kind === 'join' ? 'JUNCTION' : node.label;
  };

  return {
    /** Edges arriving at a node, by the label of where they came from. */
    edgesInto: (label: string) => {
      const target = graph.nodes.find((node) => node.label === label);
      return graph.edges
        .filter((edge) => edge.to === target?.id)
        .map((edge) => `${name(edge.from)} -${edge.kind}->`)
        .sort();
    },
    joins: graph.nodes.filter((node) => node.kind === 'join').length,
    /** Edges feeding a node, following through a junction, keeping each kind. */
    contributorsOf: (label: string) => {
      const target = graph.nodes.find((node) => node.label === label);
      return graph.edges
        .filter((edge) => edge.to === target?.id)
        .flatMap((edge) =>
          byID.get(edge.from)?.kind === 'join'
            ? graph.edges
                .filter((inner) => inner.to === edge.from)
                .map((inner) => `${name(inner.from)} -${inner.kind}->`)
            : [`${name(edge.from)} -${edge.kind}->`]
        )
        .sort();
    },
    /** Labels feeding a node, following through a junction when there is one. */
    sourcesOf: (label: string) => {
      const target = graph.nodes.find((node) => node.label === label);
      const incoming = graph.edges.filter((edge) => edge.to === target?.id);
      const resolved = incoming.flatMap((edge) =>
        byID.get(edge.from)?.kind === 'join'
          ? graph.edges.filter((e) => e.to === edge.from).map((e) => name(e.from))
          : [name(edge.from)]
      );
      return resolved.sort();
    },
    parentsOf: (label: string) =>
      (graph.nodes.find((node) => node.label === label)?.parentNodeIDs ?? []).map(name).sort(),
  };
};

describe('joins', () => {
  it('draws an edge from every member of the join', () => {
    const graph = labelled(withLineage(['a', 'b', 'c']));

    expect(graph.parentsOf('d')).toEqual(['a', 'b', 'c']);

    // All three converge through one junction rather than three lines meeting
    // at the node itself.
    expect(graph.sourcesOf('d')).toEqual(['a', 'b', 'c']);
    expect(graph.edgesInto('d')).toEqual(['JUNCTION -sequence->']);
  });

  it('draws a race as one real edge and dashed alternates', () => {
    const graph = labelled(withLineage(['a'], ['b', 'c']));

    // b and c could have unblocked d and did not, so they are not dependencies
    // and must not appear as parents.
    expect(graph.parentsOf('d')).toEqual(['a']);
    expect(graph.edgesInto('d')).toEqual(['a -sequence->', 'b -alternate->', 'c -alternate->']);
  });

  it('draws the unnamed half of a partial join as unconfirmed', () => {
    // An SDK that reports one parent names only whichever finished last. That
    // one is a real dependency, so it is solid; b and c finished in time to
    // have gated d too and nothing says they did, so they are drawn broken
    // rather than dropped — which would leave them looking like dead ends.
    const graph = labelled(withLineage(['a']));

    expect(graph.parentsOf('d')).toEqual(['a']);

    // All three arrive through the convergence — they did all finish first —
    // but only a is drawn as a dependency.
    expect(graph.contributorsOf('d')).toEqual([
      'a -fanIn->',
      'b -unconfirmed->',
      'c -unconfirmed->',
    ]);
  });

  it('draws no edge from a step that was still running', () => {
    // The dead end: `const pa = step.run(a), pb = step.run(b); await pa;
    // step.run(c); await pb`. Here b is made to still be running when d starts,
    // which rules it out as a gate — so it gets no edge, broken or otherwise.
    const fixture = structuredClone(v4parallel as unknown as Fixture);
    const stamp = (span: Trace) => {
      if (span.name === 'd') span.parentStepIDs = [stepIDs.a!];
      if (span.name === 'b') span.endedAt = null;
      for (const child of span.childrenSpans ?? []) stamp(child as Trace);
    };
    stamp(fixture.run.trace);

    const graph = labelled(toCanvasGraph(traceRollup(fixture.run.trace)));

    expect(graph.edgesInto('d')).not.toContain('b -unconfirmed->');
  });

  it('leaves a run with no lineage at all untouched', () => {
    // The fixture as captured: no parents anywhere past the first level.
    const graph = labelled(
      toCanvasGraph(traceRollup((v4parallel as unknown as Fixture).run.trace))
    );

    expect(graph.parentsOf('d')).toEqual([]);

    // The fallback junction for the unresolved hop, plus the one where the run
    // widened out of the trigger.
    expect(graph.joins).toBe(2);
  });
});
