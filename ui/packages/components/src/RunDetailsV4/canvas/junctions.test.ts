/**
 * Junctions: the points where the run changes shape.
 *
 * A junction is drawn wherever an edge bundle is not one-to-one — one step
 * becoming several, or several converging into one. Each is a real SDK request
 * (the executor asking what to run next), so where one can be identified the
 * circle carries it and can be clicked.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import chains from './__fixtures__/t19-chains.json';
import deadend from './__fixtures__/t19-deadend.json';
import parallel from './__fixtures__/t19-parallel.json';
import race from './__fixtures__/t19-race.json';
import { toCanvasGraph } from './graph';
import type { CanvasGraph } from './graph.types';

type Fixture = { run: { trace: Trace } };

const build = (fixture: unknown): CanvasGraph =>
  toCanvasGraph(traceRollup((fixture as Fixture).run.trace));

const junctions = (graph: CanvasGraph) => graph.nodes.filter((node) => node.kind === 'join');

describe('junctions', () => {
  it('marks both ends of a parallel section', () => {
    const graph = build(parallel);
    const [out, back] = junctions(graph).sort((a, b) => a.level - b.level);

    // Widening: one request planned all three.
    expect(graph.edges.filter((e) => e.from === out!.id)).toHaveLength(3);
    expect(out!.discovery?.next.sort()).toEqual(['a', 'b', 'c']);

    // Narrowing: the request that ran the step after them.
    expect(graph.edges.filter((e) => e.to === back!.id)).toHaveLength(3);
    expect(back!.discovery?.next).toEqual(['d']);
  });

  it('sits between the levels it joins rather than on one of them', () => {
    for (const junction of junctions(build(parallel))) {
      expect(Number.isInteger(junction.level)).toBe(false);
    }
  });

  it('carries the timing of the request it stands for', () => {
    const junction = junctions(build(parallel)).find((j) => j.discovery)!;
    expect(junction.spanID).not.toBe('');
    expect(junction.discovery!.endedAt).toBeGreaterThan(junction.discovery!.startedAt!);
  });

  it('leaves a one-to-one hop alone', () => {
    // The two chains continue on their own, so the only junctions are where the
    // run widened at the start and came back together at the end.
    const graph = build(chains);
    expect(junctions(graph)).toHaveLength(2);

    const byID = new Map(graph.nodes.map((n) => [n.id, n]));
    const midHop = graph.edges.filter(
      (e) => byID.get(e.from)?.label === 'left-1' || byID.get(e.from)?.label === 'right-1'
    );
    expect(midHop.every((e) => byID.get(e.to)?.kind !== 'join')).toBe(true);
  });

  it('routes an unproven contributor through the junction, still broken', () => {
    // It finished in time to have taken part, so the convergence is true as an
    // ordering. The line stays broken to say we cannot show it was waited for.
    const graph = build(deadend);
    const byID = new Map(graph.nodes.map((n) => [n.id, n]));

    const unconfirmed = graph.edges.filter((e) => e.kind === 'unconfirmed');
    expect(unconfirmed).toHaveLength(1);
    expect(byID.get(unconfirmed[0]!.from)?.label).toBe('b-deadend');
    expect(byID.get(unconfirmed[0]!.to)?.kind).toBe('join');

    // And the junction says so rather than claiming what was waited for.
    const converge = junctions(graph).find((j) => j.junction?.direction === 'converge')!;
    expect(converge.junction?.unproven).toBe(1);
  });

  it('keeps a race loser out of the convergence', () => {
    // Unlike an unproven edge, this one is known *not* to have contributed, so
    // routing it through the circle would say the opposite of what happened.
    const graph = build(race);
    const byID = new Map(graph.nodes.map((n) => [n.id, n]));
    const alternate = graph.edges.find((e) => e.kind === 'alternate')!;

    expect(byID.get(alternate.to)?.kind).not.toBe('join');
  });

  it('stays a plain circle when no single request explains it', () => {
    // A race schedules a discovery per branch, so no one request planned both
    // and there is nothing honest to attach.
    const junction = junctions(build(race))[0]!;
    expect(junction.discovery).toBeFalsy();
    expect(junction.spanID).toBe('');
  });
});

describe('what a junction says it is', () => {
  it('names the widening and the coalesce separately', () => {
    const [out, back] = junctions(build(parallel)).sort((a, b) => a.level - b.level);

    expect(out!.junction).toEqual({ direction: 'diverge', from: 1, unproven: 0, to: 3 });
    expect(back!.junction).toEqual({ direction: 'converge', from: 3, unproven: 0, to: 1 });
  });

  it('knows its shape even when no request could be matched', () => {
    // A race schedules a discovery per branch, so there is no single request —
    // but it still fanned out, and the circle can still say so.
    const junction = junctions(build(race))[0]!;
    expect(junction.discovery).toBeFalsy();
    expect(junction.junction?.direction).toBe('diverge');
  });
});
