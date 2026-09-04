/**
 * `index.ts` is hand-maintained, so the thing that can go wrong is a fixture
 * being captured, committed, and then never rendered anywhere because nobody
 * added it to the list. These tests make that a failure rather than a silence.
 */
import { readdirSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

import type { Trace } from '../../types';
import { traceRollup } from '../../utils/traceConversion';
import { toCanvasGraph } from '../graph';
import { FIXTURES, FIXTURES_BY_ID } from './index';

// Resolved from the vitest root rather than `import.meta.url`, which is not a
// file: URL under the jsdom environment this package tests in. If the root ever
// moves, `readdirSync` throws rather than quietly returning nothing.
const here = resolve(process.cwd(), 'src/RunDetailsV4/canvas/__fixtures__');

const onDisk = readdirSync(here)
  .filter((name) => name.endsWith('.json'))
  .map((name) => name.replace(/\.json$/, ''))
  .sort();

describe('the fixture index', () => {
  it('lists every fixture on disk', () => {
    expect(FIXTURES.map((f) => f.id).sort()).toEqual(onDisk);
  });

  it('gives every fixture a unique id', () => {
    expect(new Set(FIXTURES.map((f) => f.id)).size).toBe(FIXTURES.length);
    expect(FIXTURES_BY_ID.size).toBe(FIXTURES.length);
  });

  it('describes every fixture', () => {
    for (const fixture of FIXTURES) {
      expect(fixture.title, `${fixture.id} title`).not.toBe('');
      // A note that does not say why the fixture is worth keeping is not a
      // note. The floor is arbitrary; the point is that it cannot be a word.
      expect(fixture.note.length, `${fixture.id} note`).toBeGreaterThan(30);
    }
  });

  it('does not claim a synthetic fixture is a capture', () => {
    // Nothing is synthetic yet. When something is, it must say so — this is the
    // guard that a post-processed payload is never mistaken for evidence.
    for (const fixture of FIXTURES) {
      if (fixture.synthetic !== undefined) {
        expect(fixture.synthetic.length, `${fixture.id} synthetic`).toBeGreaterThan(20);
      }
    }
  });
});

describe('every listed fixture', () => {
  it.each(FIXTURES.map((f) => [f.id, f] as const))('%s builds a graph', (_id, fixture) => {
    const graph = toCanvasGraph(traceRollup(fixture.data.run.trace as Trace));

    // The gallery renders these unguarded, so "it derives without throwing and
    // has a beginning and an end" is the bar every one of them has to clear.
    expect(graph.nodes.length).toBeGreaterThan(0);
    expect(graph.levels.length).toBeGreaterThan(0);
    expect(graph.nodes.some((n) => n.kind === 'event')).toBe(true);
    expect(graph.nodes.some((n) => n.kind === 'result')).toBe(true);
  });
});
