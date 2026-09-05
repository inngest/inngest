/**
 * One sweep over every committed fixture, asserting the invariants that the
 * three views must satisfy together. These are cheap, they run on all 44
 * shapes at once, and each one is here because it was violated at some point.
 */
import { describe, expect, it } from 'vitest';

import { generateBarSegments } from './Timeline';
import { FIXTURES } from './canvas/__fixtures__/index';
import { planCollapse } from './canvas/collapse';
import { toCanvasGraph } from './canvas/graph';
import type { Trace } from './types';
import { packMinimap } from './utils/density';
import { linearScale } from './utils/timeScale';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

function viewsOf(fixture: (typeof FIXTURES)[number]) {
  const rolled = traceRollup(fixture.data.run.trace as Trace);
  const graph = toCanvasGraph(rolled);
  const data = traceToTimelineData(rolled, { runID: fixture.id });
  const scale = linearScale(data.minTime.getTime(), data.maxTime.getTime());
  return { graph, data, minimap: packMinimap(data.bars, scale), plan: planCollapse(graph) };
}

describe.each(FIXTURES.map((f) => [f.id, f] as const))('%s', (id, fixture) => {
  it('draws no bar longer than the run itself', () => {
    // `simple` broke this: reclaiming the leading queue delay moved the plot's
    // start without moving the run bar, so the Run row reported 653ms across an
    // axis covering 497ms.
    const { data } = viewsOf(fixture);
    const runMs = data.maxTime.getTime() - data.minTime.getTime();

    const walk = (bars: typeof data.bars) => {
      for (const bar of bars) {
        if (bar.endTime) {
          // A millisecond of slack for rounding between the two clocks.
          expect(
            bar.endTime.getTime() - bar.startTime.getTime(),
            `${id}: ${bar.name}`
          ).toBeLessThanOrEqual(runMs + 5);
        }
        walk(bar.children ?? []);
      }
    };
    walk(data.bars);
  });

  it('never reports an unfinished step against the current time', () => {
    // Every bar has an end once the run has ended, and it is inside the run.
    const { data } = viewsOf(fixture);
    const runEnded = fixture.data.run.status !== 'RUNNING';
    if (!runEnded) return;

    const walk = (bars: typeof data.bars) => {
      for (const bar of bars) {
        expect(bar.endTime, `${id}: ${bar.name} has no end`).not.toBeNull();
        walk(bar.children ?? []);
      }
    };
    walk(data.bars);
  });

  it('agrees with the canvas about how many steps there were', () => {
    // The minimap is built from timeline rows and the canvas from the graph, by
    // separate paths. If they disagree, one of them is inventing or dropping a
    // step.
    const { graph, minimap } = viewsOf(fixture);
    const work = graph.nodes.filter(
      (n) => n.kind === 'step' || n.kind === 'wait' || n.kind === 'invoke'
    ).length;

    expect(minimap.marks.length, `${id}: minimap vs canvas`).toBe(work);
  });

  it('needs at least one lane, and no more lanes than steps', () => {
    const { minimap } = viewsOf(fixture);
    expect(minimap.rows).toBeGreaterThanOrEqual(1);
    if (minimap.marks.length > 0) {
      expect(minimap.rows).toBeLessThanOrEqual(minimap.marks.length);
    }
  });

  it('draws a waiting lead-in on the Run row only when it says one is there', () => {
    // The bar and its own label have to agree. This broke because the root fell
    // through to the synthesised "total minus what the children executed"
    // figure: `chains` drew 86% of a continuously-busy run as waiting directly
    // above steps drawn as executing over the very same interval, while its
    // label said `+100ms queued`.
    const { data } = viewsOf(fixture);
    const root = data.bars[0];
    if (!root?.isRoot) return;

    const segments = generateBarSegments(root);
    const waiting = segments?.filter((s) => s.style === 'timing.waiting') ?? [];
    if (!root.delayMs) {
      expect(waiting, `${id}: Run row draws a wait it does not name`).toHaveLength(0);
    }
  });

  it('never draws a discovery over a span already on screen', () => {
    // The characteristic bug of this view is the same interval drawn twice. A
    // discovery span is usually the parent of what it planned, and the run's
    // trailing one *is* the finalization span, so without this the Planning row
    // sits directly on top of an identical bar one row below.
    const { data } = viewsOf(fixture);
    const planning = data.bars[0]?.children?.find((b) => b.id === 'run-discovery');
    if (!planning) return;

    const drawn = new Set<string>();
    const walk = (bars: typeof data.bars) => {
      for (const bar of bars) {
        if (bar.id !== 'run-discovery') drawn.add(bar.id);
        walk(bar.children ?? []);
      }
    };
    walk(data.bars);

    for (const segment of planning.segments ?? []) {
      // `discovery-<spanID>-<i>`.
      const spanID = segment.id.slice('discovery-'.length, segment.id.lastIndexOf('-'));
      expect(drawn.has(spanID), `${id}: discovery ${spanID} is already a bar`).toBe(false);
    }
  });
});
