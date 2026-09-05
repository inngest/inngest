/**
 * One test, over every committed fixture.
 *
 * This was five invariants times forty-four fixtures — 220 cases for what is
 * really one question: does anything the three views say contradict anything
 * else they say? Written as a sweep that collects violations and reports them
 * together, the coverage is identical and the suite is one line instead of a
 * screenful, which matters when the whole thing is a proof of concept.
 *
 * Each check is here because it was violated at some point, and every message
 * names its fixture so a failure still says where to look.
 */
import { describe, expect, it } from 'vitest';

import { generateBarSegments } from './Timeline';
import type { TimelineBarData } from './TimelineBar.types';
import { FIXTURES } from './canvas/__fixtures__/index';
import { toCanvasGraph } from './canvas/graph';
import type { Trace } from './types';
import { packMinimap } from './utils/density';
import { linearScale } from './utils/timeScale';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

describe('the three views agree', () => {
  it('on every committed fixture', () => {
    const problems: string[] = [];

    for (const fixture of FIXTURES) {
      const id = fixture.id;
      const rolled = traceRollup(fixture.data.run.trace as Trace);
      const graph = toCanvasGraph(rolled);
      const data = traceToTimelineData(rolled, { runID: id });
      const minimap = packMinimap(
        data.bars,
        linearScale(data.minTime.getTime(), data.maxTime.getTime())
      );

      const runMs = data.maxTime.getTime() - data.minTime.getTime();
      const ended = fixture.data.run.status !== 'RUNNING';

      const walk = (bars: TimelineBarData[]) => {
        for (const bar of bars) {
          // A bar longer than the run means the plot and the number disagree.
          // `simple` did this when the leading queue delay was reclaimed.
          if (bar.endTime && bar.endTime.getTime() - bar.startTime.getTime() > runMs + 5) {
            problems.push(`${id}: ${bar.name} is longer than the run`);
          }
          // An unfinished span measured against `now` grows every time the page
          // is opened. `cancelled` reported 29 minutes on a 35-second run.
          if (ended && !bar.endTime) {
            problems.push(`${id}: ${bar.name} has no end in a finished run`);
          }
          walk(bar.children ?? []);
        }
      };
      walk(data.bars);

      // The minimap comes from timeline rows and the canvas from the graph, by
      // separate paths. Disagreeing means one is inventing or dropping a step.
      const work = graph.nodes.filter(
        (n) => n.kind === 'step' || n.kind === 'wait' || n.kind === 'invoke'
      ).length;
      if (minimap.marks.length !== work) {
        problems.push(`${id}: minimap has ${minimap.marks.length} steps, canvas has ${work}`);
      }

      const root = data.bars[0];
      if (root?.isRoot && root.endTime) {
        // The Run row is the ruler everything else is read against. Reporting
        // the run's whole life while the axis spanned execution only had
        // `simple` drawing a full-width bar labelled 165ms over a 0→32ms axis.
        const printed = root.endTime.getTime() - root.startTime.getTime();
        if (Math.abs(printed - runMs) > 2) {
          problems.push(`${id}: Run says ${printed}ms, axis covers ${runMs}ms`);
        }

        // And it draws a waiting lead-in only when it says one is there.
        const waiting = (generateBarSegments(root) ?? []).filter(
          (s) => s.style === 'timing.waiting'
        );
        if (!root.delayMs && waiting.length) {
          problems.push(`${id}: Run row draws a wait it does not name`);
        }
      }
    }

    expect(problems, `\n${problems.join('\n')}\n`).toEqual([]);
  });
});
