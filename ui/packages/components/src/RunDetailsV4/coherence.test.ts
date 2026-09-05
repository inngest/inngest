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
import { leadInMs, traceRollup, traceToTimelineData } from './utils/traceConversion';

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

  it('puts its rows in the order the steps started', () => {
    // The spans arrive in the order the SDK reported them, which §3.5 says is
    // non-deterministic under parallelism: `t19-parallel` listed its three
    // parallel steps b, c, a. A waterfall whose rows are not in time order is
    // not a waterfall.
    const { data } = viewsOf(fixture);

    const walk = (bars: typeof data.bars) => {
      let previous = -Infinity;
      for (const bar of bars) {
        const began =
          (bar.reportedMs !== undefined && bar.endTime
            ? bar.endTime.getTime() - bar.reportedMs
            : bar.startTime.getTime()) + (bar.delayMs ?? 0);
        expect(began, `${id}: ${bar.name} is out of order`).toBeGreaterThanOrEqual(previous);
        previous = began;
        walk(bar.children ?? []);
      }
    };
    walk(data.bars[0]?.children ?? []);
  });

  it('labels the Run row with the width it actually occupies', () => {
    // The ruler everything else is read against.
    //
    // This is the assertion that was missing, and its absence let a regression
    // through on 43 of 45 fixtures: making the Run row report the run's whole
    // life moved the NUMBER without moving the PLOT, so `simple` drew a
    // full-width bar labelled 165ms against an axis reading 0ms → 32ms.
    //
    // The test above it compared the bar's own span to the axis and passed
    // throughout, because the regression was in the rendered label rather than
    // in the geometry — the same shape of miss as asserting a tooltip's model
    // while it never reached the screen. So this one asserts what is printed.
    const { data } = viewsOf(fixture);
    const root = data.bars[0];
    // A run still in flight measures both its bar and its axis against `now`,
    // so they move together and there is nothing fixed to compare.
    if (!root?.isRoot || !root.endTime) return;

    const printed = root.endTime.getTime() - root.startTime.getTime();
    const axis = data.maxTime.getTime() - data.minTime.getTime();

    expect(
      Math.abs(printed - axis),
      `${id}: Run says ${printed}, axis covers ${axis}`
    ).toBeLessThan(2);
  });
});
