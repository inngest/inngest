/**
 * Collapsing is judged against the fixtures it exists for: `loop40` (40
 * iterations), `tall500` (500 sequential steps) and `wide` (a 12-wide fan-out).
 * The acceptance criterion from the brief is a number — no run draws more than
 * ~40 nodes at rest — so it is asserted as one.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup, traceToTimelineData } from '../utils/traceConversion';
import chains from './__fixtures__/chains.json';
import failure from './__fixtures__/failure.json';
import loop40 from './__fixtures__/loop40.json';
import loop from './__fixtures__/loop.json';
import parallel from './__fixtures__/parallel.json';
import retry from './__fixtures__/retry.json';
import simple from './__fixtures__/simple.json';
import step from './__fixtures__/step.json';
import tall500 from './__fixtures__/tall500.json';
import v4branchesBalanced from './__fixtures__/v4branches-balanced.json';
import v4branchesRagged from './__fixtures__/v4branches-ragged.json';
import v4pathological from './__fixtures__/v4pathological.json';
import wide from './__fixtures__/wide.json';
import {
  applyCollapse,
  applyCollapseToBars,
  groupTitle,
  planCollapse,
  shapeKey,
  shouldAggregate,
} from './collapse';
import { toCanvasGraph } from './graph';
import type { CanvasGraph, CanvasNode } from './graph.types';

type Fixture = { run: { status: string; trace: unknown } };

function build(fixture: unknown): CanvasGraph {
  return toCanvasGraph(traceRollup((fixture as Fixture).run.trace as Trace));
}

const NODE_BUDGET = 40;

describe('shapeKey', () => {
  const node = (over: Partial<CanvasNode>): CanvasNode =>
    ({
      id: 'x',
      kind: 'step',
      label: 'a',
      status: 'COMPLETED',
      level: 1,
      lane: 0,
      spanID: 'x',
      stepID: null,
      stepOp: 'RUN',
      attempts: 0,
      queuedAt: 0,
      startedAt: 0,
      endedAt: 1,
      ...over,
    } as CanvasNode);

  it('treats numbered siblings as the same shape', () => {
    expect(shapeKey(node({ userlandStepID: 'think-0' }))).toBe(
      shapeKey(node({ userlandStepID: 'think-39' }))
    );
    expect(shapeKey(node({ userlandStepID: 's0' }))).toBe(
      shapeKey(node({ userlandStepID: 's499' }))
    );
  });

  it('keeps genuinely different names apart', () => {
    expect(shapeKey(node({ userlandStepID: 'think-0' }))).not.toBe(
      shapeKey(node({ userlandStepID: 'act-0' }))
    );
  });

  it('never lets a wait merge with a step', () => {
    expect(shapeKey(node({ userlandStepID: 'a', kind: 'step' }))).not.toBe(
      shapeKey(node({ userlandStepID: 'a', kind: 'wait' }))
    );
  });

  it('ignores the SDK collision suffix, which is not stable across runs', () => {
    expect(shapeKey(node({ userlandStepID: 'dupe:1' }))).toBe(
      shapeKey(node({ userlandStepID: 'dupe:2' }))
    );
  });
});

describe('the node budget', () => {
  const cases: Array<{ name: string; fixture: unknown }> = [
    { name: 'loop40', fixture: loop40 },
    { name: 'tall500', fixture: tall500 },
    { name: 'wide', fixture: wide },
    { name: 'loop', fixture: loop },
  ];

  it.each(cases)('$name draws under the budget at rest', ({ fixture }) => {
    const plan = planCollapse(build(fixture));
    expect(plan.nodesAtRest).toBeLessThanOrEqual(NODE_BUDGET);
  });

  it('leaves a run that is already small entirely alone', () => {
    for (const fixture of [simple, step, parallel, chains]) {
      const graph = build(fixture);
      const plan = planCollapse(graph);
      expect(plan.nodesAtRest).toBe(graph.nodes.length);
      expect(plan.groups).toHaveLength(0);
    }
  });
});

describe('an agent loop', () => {
  const plan = planCollapse(build(loop40));

  it('collapses to the loop body, not to one node per step', () => {
    // think/act alternating: two positions in the body, 40 of each.
    expect(plan.groups).toHaveLength(2);
    expect(plan.groups.map((g) => g.count)).toEqual([40, 40]);
    expect(plan.groups.every((g) => g.kind === 'iteration')).toBe(true);
  });

  it('accounts for every step it collapsed', () => {
    const members = plan.groups.flatMap((g) => g.memberNodeIDs);
    expect(members).toHaveLength(80);
    expect(new Set(members).size).toBe(80);
  });

  it('keeps the durations so the variation is still visible', () => {
    for (const group of plan.groups) {
      expect(group.durationsMs).toHaveLength(group.count);
      expect(group.durationsMs.some((d) => d > 0)).toBe(true);
    }
  });

  it('says that it grouped by name with numbers ignored', () => {
    expect(plan.warnings.join(' ')).toMatch(/numbers ignored/i);
  });
});

describe('500 sequential steps', () => {
  const plan = planCollapse(build(tall500));

  it('become one group', () => {
    expect(plan.groups).toHaveLength(1);
    expect(plan.groups[0]!.count).toBe(500);
    expect(plan.groups[0]!.kind).toBe('iteration');
  });

  it('span every level they occupied', () => {
    expect(plan.groups[0]!.levels).toHaveLength(500);
  });
});

describe('a wide fan-out', () => {
  const graph = build(wide);
  const plan = planCollapse(graph);

  it('collapses the siblings, not the collect step after them', () => {
    const fanOut = plan.groups.find((g) => g.count === 12);
    expect(fanOut).toBeDefined();
    expect(fanOut!.kind).toBe('siblings');
  });

  it('keeps the envelope rather than a percentile', () => {
    const fanOut = plan.groups.find((g) => g.count === 12)!;
    const { firstStartedAt, lastStartedAt, lastEndedAt, staggerMs } = fanOut.envelope;
    expect(firstStartedAt).toBeLessThanOrEqual(lastStartedAt);
    expect(lastEndedAt).toBeGreaterThanOrEqual(lastStartedAt);
    expect(staggerMs).toBe(lastStartedAt - firstStartedAt);
  });
});

describe('carrying the variance', () => {
  it('reports every distinct label, most common first', () => {
    const plan = planCollapse(build(loop40));
    for (const group of plan.groups) {
      const total = group.variants.reduce((n, v) => n + v.count, 0);
      expect(total).toBe(group.count);
      const counts = group.variants.map((v) => v.count);
      expect([...counts].sort((a, b) => b - a)).toEqual(counts);
    }
  });

  it('calls indexed names numbering rather than variance', () => {
    // `think-0 … think-39` is forty distinct labels and no actual variation.
    // Reporting that as "40 variants" would be noise dressed up as signal.
    const plan = planCollapse(build(loop40));
    for (const group of plan.groups) {
      expect(group.variance).toBe('numbered');
      expect(group.labelRange).not.toBeNull();
    }
  });

  it('calls a genuine mix of tools variance', () => {
    // The shape the brief cares about: an agent loop where iteration 7 called a
    // different tool. Relabel members so the names repeat and differ.
    const graph = build(loop40);
    const plan0 = planCollapse(graph);
    const members = plan0.groups[0]!.memberNodeIDs;
    const byID = new Map(graph.nodes.map((n) => [n.id, n]));
    members.forEach((id, i) => {
      const node = byID.get(id)!;
      node.label = i % 20 === 7 ? 'write_file' : 'search';
      node.userlandStepID = 'tool';
    });

    const group = planCollapse(graph).groups.find((g) => g.memberNodeIDs.includes(members[0]!))!;

    expect(group.variance).toBe('mixed');
    expect(group.variants).toEqual([
      { label: 'search', count: 38, nodeIDs: expect.any(Array) },
      { label: 'write_file', count: 2, nodeIDs: expect.any(Array) },
    ]);
  });

  it('never hides a failure behind a green group', () => {
    // Synthesise the case the fixtures do not have: a loop where one iteration
    // failed. A group is drawn as its worst member, so this must not be green.
    const graph = build(loop40);
    const victim = graph.nodes.find((n) => n.kind === 'step')!;
    victim.status = 'FAILED';

    const plan = planCollapse(graph);
    const holding = plan.groups.find((g) => g.memberNodeIDs.includes(victim.id))!;

    expect(holding.status).toBe('FAILED');
    expect(holding.exceptions).toContainEqual({ nodeID: victim.id, reason: 'failed' });
  });

  it('names a retried member even when the run succeeded', () => {
    const graph = build(loop40);
    const victim = graph.nodes.find((n) => n.kind === 'step')!;
    victim.attempts = 2;

    const plan = planCollapse(graph);
    const holding = plan.groups.find((g) => g.memberNodeIDs.includes(victim.id))!;
    expect(holding.exceptions).toContainEqual({ nodeID: victim.id, reason: 'retried' });
  });

  it('names a member that ran long', () => {
    const graph = build(loop40);
    const victim = graph.nodes.find((n) => n.kind === 'step' && n.endedAt !== null)!;
    victim.endedAt = victim.startedAt! + 60_000;

    const plan = planCollapse(graph);
    const holding = plan.groups.find((g) => g.memberNodeIDs.includes(victim.id))!;
    expect(holding.exceptions).toContainEqual({ nodeID: victim.id, reason: 'slow' });
  });
});

describe('what collapsing must not do', () => {
  it('never puts a node in two groups', () => {
    for (const fixture of [loop40, tall500, wide, loop, v4pathological, chains]) {
      const plan = planCollapse(build(fixture));
      const members = plan.groups.flatMap((g) => g.memberNodeIDs);
      expect(new Set(members).size).toBe(members.length);
    }
  });

  it('only ever collapses real work', () => {
    for (const fixture of [loop40, tall500, wide]) {
      const graph = build(fixture);
      const byID = new Map(graph.nodes.map((n) => [n.id, n]));
      const plan = planCollapse(graph);
      for (const id of plan.groups.flatMap((g) => g.memberNodeIDs)) {
        expect(['step', 'wait', 'invoke']).toContain(byID.get(id)!.kind);
      }
    }
  });

  it('collapses nothing when there is nothing repeated', () => {
    for (const fixture of [step, retry, failure, chains]) {
      expect(planCollapse(build(fixture)).groups).toHaveLength(0);
    }
  });

  it('agrees with itself about how many nodes are left', () => {
    for (const fixture of [loop40, tall500, wide, loop]) {
      const graph = build(fixture);
      const plan = planCollapse(graph);
      const collapsed = plan.groups.reduce((n, g) => n + g.count, 0);
      expect(plan.nodesAtRest).toBe(graph.nodes.length - collapsed + plan.groups.length);
      expect(plan.nodesExpanded).toBe(graph.nodes.length);
    }
  });
});

describe('the rewritten graph', () => {
  const cases = [
    { name: 'loop40', fixture: loop40 },
    { name: 'tall500', fixture: tall500 },
    { name: 'wide', fixture: wide },
    { name: 'loop', fixture: loop },
  ];

  it.each(cases)('$name loses every level that was emptied', ({ fixture }) => {
    const graph = build(fixture);
    const { graph: collapsed } = applyCollapse(graph, planCollapse(graph));

    // No empty levels, and levels are numbered contiguously from zero.
    expect(collapsed.levels.every((l) => l.length > 0)).toBe(true);

    // Junctions sit on half-levels and are deliberately not in `levels`, the
    // same as in the graph this was built from. Everything else must be.
    collapsed.nodes
      .filter((n) => n.kind !== 'join')
      .forEach((n) => {
        expect(collapsed.levels[n.level]).toContain(n.id);
      });
  });

  it.each(cases)('$name keeps a beginning and an end', ({ fixture }) => {
    const graph = build(fixture);
    const { graph: collapsed } = applyCollapse(graph, planCollapse(graph));
    expect(collapsed.nodes.some((n) => n.kind === 'event')).toBe(true);
    expect(collapsed.nodes.some((n) => n.kind === 'result')).toBe(true);
  });

  it.each(cases)('$name draws no edge to a node that is gone', ({ fixture }) => {
    const graph = build(fixture);
    const { graph: collapsed } = applyCollapse(graph, planCollapse(graph));
    const ids = new Set(collapsed.nodes.map((n) => n.id));
    for (const edge of collapsed.edges) {
      expect(ids.has(edge.from), `${edge.id} from`).toBe(true);
      expect(ids.has(edge.to), `${edge.id} to`).toBe(true);
      expect(edge.from).not.toBe(edge.to);
    }
  });

  it.each(cases)('$name still connects the trigger to the result', ({ fixture }) => {
    const graph = build(fixture);
    const { graph: collapsed } = applyCollapse(graph, planCollapse(graph));

    // Collapsing must not sever the run: walking forward from the event node
    // has to reach the result, or the graph reads as two disconnected halves.
    const out = new Map<string, string[]>();
    for (const e of collapsed.edges) out.set(e.from, [...(out.get(e.from) ?? []), e.to]);

    const start = collapsed.nodes.find((n) => n.kind === 'event')!;
    const end = collapsed.nodes.find((n) => n.kind === 'result')!;
    const seen = new Set<string>();
    const queue = [start.id];
    while (queue.length) {
      const id = queue.pop()!;
      if (seen.has(id)) continue;
      seen.add(id);
      queue.push(...(out.get(id) ?? []));
    }
    expect(seen.has(end.id)).toBe(true);
  });

  it('matches the node count the plan promised', () => {
    for (const { fixture } of cases) {
      const graph = build(fixture);
      const plan = planCollapse(graph);
      const { graph: collapsed } = applyCollapse(graph, plan);
      expect(collapsed.nodes).toHaveLength(plan.nodesAtRest);
    }
  });

  it('gives every group node its group', () => {
    const graph = build(loop40);
    const plan = planCollapse(graph);
    const { graph: collapsed, groupByNodeID } = applyCollapse(graph, plan);
    const groupNodes = collapsed.nodes.filter((n) => n.kind === 'group');
    expect(groupNodes).toHaveLength(plan.groups.length);
    for (const node of groupNodes) {
      expect(groupByNodeID.get(node.id)).toBeDefined();
    }
  });

  it('expands the groups the user opened, and only those', () => {
    const graph = build(loop40);
    const plan = planCollapse(graph);
    const opened = new Set([plan.groups[0]!.id]);
    const { graph: collapsed } = applyCollapse(graph, plan, opened);

    const kinds = collapsed.nodes.filter((n) => n.kind === 'group');
    expect(kinds).toHaveLength(1);
    expect(kinds[0]!.id).toBe(plan.groups[1]!.id);
    // The opened group's 40 members are back.
    expect(collapsed.nodes.length).toBeGreaterThan(plan.nodesAtRest);
  });

  it('leaves a graph with nothing to collapse untouched', () => {
    const graph = build(step);
    const { graph: collapsed } = applyCollapse(graph, planCollapse(graph));
    expect(collapsed).toBe(graph);
  });

  it('never promotes a broken edge into a dependency', () => {
    // A group's outgoing edge takes the strongest kind any member had, but it
    // may not invent one that no member had.
    for (const { fixture } of cases) {
      const graph = build(fixture);
      const plan = planCollapse(graph);
      const before = new Set(graph.edges.map((e) => e.kind));
      const { graph: collapsed } = applyCollapse(graph, plan);
      for (const edge of collapsed.edges) {
        expect(before.has(edge.kind)).toBe(true);
      }
    }
  });
});

describe('groupTitle', () => {
  it('reads numbered members as a range rather than a truncation', () => {
    const plan = planCollapse(build(loop40));
    expect(plan.groups.map(groupTitle)).toEqual(['think-[0…39]', 'act-[0…39]']);
    expect(groupTitle(planCollapse(build(tall500)).groups[0]!)).toBe('s[0…499]');
  });

  it('names the shape when the members are not just numbered', () => {
    const graph = build(wide);
    const group = planCollapse(graph).groups.find((g) => g.count === 12)!;
    // `w0`..`w11` are numbered too, so this is still a range — the assertion
    // that matters is that it is short and says what the group is.
    expect(groupTitle(group)).toBe('w[0…11]');
  });
});

describe('shouldAggregate', () => {
  it('opens collapsed when the run is too long, too wide, or too many', () => {
    for (const fixture of [loop40, tall500, wide]) {
      const graph = build(fixture);
      expect(shouldAggregate(graph, planCollapse(graph))).toBe(true);
    }
  });

  it('leaves a readable run expanded', () => {
    for (const fixture of [step, parallel, chains, retry]) {
      const graph = build(fixture);
      expect(shouldAggregate(graph, planCollapse(graph))).toBe(false);
    }
  });
});

describe('what the reviewers caught', () => {
  it('calls a member slow only when it is anomalous, not merely the top of a ramp', () => {
    // loop40's think durations are a smooth ramp — …213, 299, 335, 370, 1675.
    // A MAD threshold alone lands at ~335, so 299 is fine and 335 is "slow": an
    // 11% difference deciding it inside a continuous distribution, and the node
    // then reported "3 slow" for one anomaly and a trend.
    const think = planCollapse(build(loop40)).groups[0]!;
    const slow = think.exceptions.filter((e) => e.reason === 'slow');
    expect(slow).toHaveLength(1);

    const flagged = think.memberNodeIDs.indexOf(slow[0]!.nodeID);
    const sorted = [...think.durationsMs].sort((a, b) => a - b);
    const median = sorted[Math.floor(sorted.length / 2)]!;
    expect(think.durationsMs[flagged]!).toBeGreaterThan(median * 4);
  });

  it('does not fuse parallel branches into a single "ran N times"', () => {
    // v4branches-balanced is six parallel branches of four sequential steps,
    // named br0-1 … br5-4. Both digit runs normalise, so all 24 shared a shape
    // and collapsed into one node claiming one step ran 24 times. They are four
    // fan-outs of six, and that is what it should say.
    const plan = planCollapse(build(v4branchesBalanced));
    expect(plan.groups).toHaveLength(4);
    for (const group of plan.groups) {
      expect(group.kind).toBe('siblings');
      expect(group.count).toBe(6);
    }
  });

  it('never labels a range that is not one', () => {
    // `br[0-1…5-4]` reads as a range and is nothing of the sort.
    for (const fixture of [loop40, tall500, wide, v4branchesBalanced, v4branchesRagged]) {
      for (const group of planCollapse(build(fixture)).groups) {
        const title = groupTitle(group);
        const range = /\[([^\]…]*)…([^\]]*)\]/.exec(title);
        if (range) {
          expect(range[1], title).toMatch(/^\d+$/);
          expect(range[2], title).toMatch(/^\d+$/);
        }
      }
    }
  });

  it('starts a group row no later than its own first child', () => {
    // Ordinary bars start at queuedAt; the group row used the envelope's first
    // START, so expanding it revealed children beginning before their parent —
    // an 88ms overhang on `wide` — and dropped the queued phase every sibling
    // row includes.
    const graph = build(wide);
    const plan = planCollapse(graph);
    const data = traceToTimelineData(traceRollup((wide as Fixture).run.trace as Trace), {
      runID: 'test',
    });
    const rows = applyCollapseToBars(data.bars, plan);

    const check = (bars: typeof rows) => {
      for (const bar of bars) {
        for (const child of bar.children ?? []) {
          expect(child.startTime.getTime(), bar.name).toBeGreaterThanOrEqual(
            bar.startTime.getTime()
          );
        }
        check(bar.children ?? []);
      }
    };
    check(rows);
  });
});

describe('a collapsed row draws the shape that happened', () => {
  it('draws a fan-out as concurrent, not as a sequence', () => {
    // `wide`'s twelve steps all started within 14ms of each other. Drawn as
    // evenly-pitched equal ticks with gaps between them they read as twelve
    // sequential 8ms steps — the opposite of the truth, and in direct
    // contradiction of the minimap one strip above, which drew them overlapping
    // across twelve lanes.
    const raw = (wide as { run: { trace: unknown } }).run.trace as Trace;
    const rolled = traceRollup(raw);
    const plan = planCollapse(toCanvasGraph(rolled));
    const data = traceToTimelineData(rolled, { runID: 'wide' });
    const bars = applyCollapseToBars(data.bars, plan);

    const group = bars[0]?.children?.find((b) => b.name.includes('×'));
    expect(group?.segments?.length).toBeGreaterThan(5);

    // Members only. The row also carries the group's collective lead-in, which
    // deliberately does not overlap the first member — it ends where it starts.
    const segments = group!.segments!.filter((s) => s.id.includes('-member-'));
    // Members overlap: every one starts before the one before it has finished.
    let overlapping = 0;
    for (let i = 1; i < segments.length; i++) {
      const previousEnd = segments[i - 1]!.startPercent + segments[i - 1]!.widthPercent;
      if (segments[i]!.startPercent < previousEnd) overlapping += 1;
    }
    expect(overlapping, 'a fan-out drawn as a sequence').toBe(segments.length - 1);

    // ...and they do not all sit on top of each other either: the stagger
    // between their starts is the picture of the concurrency limit, and
    // measuring from queuedAt rather than execution start flattened it to zero.
    const starts = segments.map((s) => s.startPercent);
    expect(Math.max(...starts) - Math.min(...starts)).toBeGreaterThan(0);
  });
});
