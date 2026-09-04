/**
 * Adversarial shapes, captured from real v4 runs.
 *
 * These exist to pin down where the canvas is honest and where it cannot be.
 * Each case records what the *code* did and what the canvas derives, so a
 * regression that quietly changes the answer shows up as a failing expectation
 * rather than as a subtly wrong picture.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import dupeNames from './__fixtures__/gnarly-dupe-names.json';
import dynamic from './__fixtures__/gnarly-dynamic.json';
import foreignAsync from './__fixtures__/gnarly-foreign-async.json';
import mixed from './__fixtures__/gnarly-mixed.json';
import nested from './__fixtures__/gnarly-nested.json';
import race from './__fixtures__/gnarly-race.json';
import sleepBranch from './__fixtures__/gnarly-sleep-branch.json';
import unbalanced from './__fixtures__/gnarly-unbalanced.json';
import { toCanvasGraph } from './graph';
import type { CanvasGraph } from './graph.types';

function build(fixture: unknown): CanvasGraph {
  return toCanvasGraph(traceRollup((fixture as { run: { trace: Trace } }).run.trace));
}

function shape(graph: CanvasGraph): string[][] {
  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  return graph.levels
    .map((ids) =>
      ids
        .map((id) => byID.get(id)!)
        .filter((n) => n.kind === 'step' || n.kind === 'invoke' || n.kind === 'wait')
        .map((n) => n.label)
        .sort()
    )
    .filter((l) => l.length > 0);
}

describe('adversarial shapes', () => {
  it('handles branches of unequal depth', () => {
    // Promise.all([ deep-1 -> deep-2 -> deep-3, shallow-1 ])
    // The shallow branch ends after one step; the deep branch keeps going alone.
    expect(shape(build(unbalanced))).toEqual([['deep-1', 'shallow-1'], ['deep-2'], ['deep-3']]);
  });

  it('flattens nested Promise.all, because the runtime does too', () => {
    // Promise.all([ Promise.all([a1, a2]), b ]) — the SDK plans all three in one
    // response, so there is no nesting left to represent. Not a loss: the
    // nesting is a source-code construct with no runtime distinction.
    expect(shape(build(nested))).toEqual([['a1', 'a2', 'b'], ['collect']]);
  });

  it('groups a dynamic fan-out whose width came from a previous step', () => {
    expect(shape(build(dynamic))).toEqual([
      ['decide'],
      ['dyn-0', 'dyn-1', 'dyn-2', 'dyn-3'],
      ['finish'],
    ]);
  });

  it('keeps a failed branch red without colouring its siblings', () => {
    const graph = build(mixed);
    expect(shape(graph)).toEqual([['will-fail', 'will-succeed'], ['after-mixed']]);
    expect(graph.nodes.find((n) => n.label === 'will-fail')!.status).toBe('FAILED');
    expect(graph.nodes.find((n) => n.label === 'will-succeed')!.status).toBe('COMPLETED');
  });

  it('places a sleep inside a parallel level when it was planned with a sibling', () => {
    // napping: before-nap -> sleep("nap") -> after-nap
    // busy:    busy-1 -> busy-2
    // The sleep and busy-2 really were planned in the same response.
    expect(shape(build(sleepBranch))).toEqual([
      ['before-nap', 'busy-1'],
      ['busy-2', 'nap'],
      ['after-nap'],
    ]);
  });

  it('draws Promise.race as waiting for every branch, which is what v4 does', () => {
    // Not a lie, despite appearances. Under v4's optimized parallelism
    // `Promise.race` waits for all parallel steps to settle before resolving —
    // the correct winner is returned, but not early. See
    // website/pages/docs/guides/step-parallelism.mdx. Early resolution is opt-in
    // via `group.parallel({ mode: 'race' })`, which is the case still untested.
    expect(shape(build(race))).toEqual([['fast', 'slow'], ['after-race']]);
  });

  it('cannot group a Promise.all whose branches resolve via non-Inngest async', () => {
    // KNOWN LIMIT, and it fails honestly rather than guessing. One branch awaits
    // a plain 250ms timer before calling step.run, so the SDK reports `early`
    // and `late` in separate responses. There is no batch, and the canvas says
    // so via parallelismSource rather than inventing one.
    const graph = build(foreignAsync);
    expect(graph.parallelismSource).toBe('none');
    expect(shape(graph)).toEqual([['early'], ['late']]);
  });

  it('disambiguates duplicate step names with the SDK collision index', () => {
    // Both branches call step.run("work"). The SDK resolves the collision with
    // an auto-incremented index; it was recorded on the span but never exposed,
    // so the trace still shows two identical rows. The canvas surfaces it.
    const graph = build(dupeNames);
    expect(shape(graph)).toEqual([
      ['work', 'work'],
      ['finish', 'finish'],
    ]);
    const works = graph.nodes.filter((n) => n.label === 'work');
    expect(works).toHaveLength(2);
    expect(works[0]!.stepID).not.toBe(works[1]!.stepID);
    // Numbered 1 and 2, so the nodes read as "work #1" / "work #2".
    expect(works.map((n) => n.duplicateIndex).sort()).toEqual([1, 2]);
  });

  it('does not number steps whose IDs are unique', () => {
    // The SDK reports index 0 for unique steps too, so a naive read of the
    // index would badge everything.
    for (const node of build(unbalanced).nodes) {
      expect(node.duplicateIndex ?? null).toBeNull();
    }
  });
});
