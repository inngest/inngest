/**
 * Collapsing repetition.
 *
 * A 40-iteration agent loop is 80 levels and a 500-step run is 502; both are
 * unreadable, and agent loops are the workload that matters. The fix everyone
 * who has confronted this converged on is not a cleverer layout but two modes —
 * an aggregated view where a repeated shape is one node with a count, and an
 * expanded view that is the graph as it is today.
 *
 * This module is the detection, once, over the finished `CanvasGraph`, so both
 * the canvas and the timeline collapse the same things for the same reasons. It
 * is deliberately not inside `graph.ts`: that file is about what the trace
 * *says*, and this one is about how much of it to draw. Nothing here changes
 * the graph, so every existing assertion about the model still holds.
 *
 * ITERATIONS ARE NOT REPEATS. An agent loop's iteration 7 called a different
 * tool, and that is the thing the user opened the trace to find. So a group
 * carries its variance rather than averaging it away: `variants` says
 * "38 x search, 2 x write_file", `exceptions` names the members that failed,
 * retried or ran long, and `status` is the worst status inside — a group holding
 * a failure is red at rest, without anyone having to expand it.
 *
 * The one genuine heuristic is `shapeKey`, which normalises digit runs so that
 * `think-0` and `think-39` are the same shape. That is an inference about what
 * a user meant by a name, so it is reported in `warnings` rather than assumed.
 */
import type { BarSegment, TimelineBarData } from '../TimelineBar.types';
import type { CanvasEdge, CanvasGraph, CanvasNode, CanvasStatus } from './graph.types';

/**
 * How a group came to be.
 *
 *  - 'iteration' consecutive levels repeating the same shape: a loop body.
 *  - 'siblings'  lanes within one level sharing a shape: a fan-out.
 */
export type CollapseKind = 'iteration' | 'siblings';

/** Why a member is worth pointing at individually. */
export type ExceptionReason =
  /** Ended in failure. */
  | 'failed'
  /** Cancelled, or still running when the run was cancelled. */
  | 'cancelled'
  /** Succeeded, but not on the first attempt. */
  | 'retried'
  /** Still going. */
  | 'running'
  /** Took long enough to be outside the band the others sit in. */
  | 'slow';

export type CollapseException = {
  nodeID: string;
  reason: ExceptionReason;
};

/** One distinct label inside a group, so the differences survive collapsing. */
export type CollapseVariant = {
  label: string;
  count: number;
  nodeIDs: string[];
};

/**
 * When the members ran.
 *
 * Percentiles are the obvious summary and the wrong one: in a wide fan-out the
 * stagger between the first and last start *is a picture of the concurrency
 * limit*, and a p50/p99 throws exactly that away.
 */
export type CollapseEnvelope = {
  firstStartedAt: number;
  lastStartedAt: number;
  lastEndedAt: number;
  /** How spread out the starts were. Zero means they all began together. */
  staggerMs: number;
};

/**
 * Whether the differences between member labels mean anything.
 *
 * This distinction is the difference between a useful summary and noise. A loop
 * written as `think-0 … think-39` has forty distinct labels and no variance —
 * they are the same step, numbered. A loop that called `search` 38 times and
 * `write_file` twice has real variance, and that is the thing the user opened
 * the trace to find.
 *
 *  - 'uniform'  every member shares one label
 *  - 'numbered' every label is distinct, i.e. the name carries the index
 *  - 'mixed'    labels repeat *and* differ — genuine variation worth showing
 */
export type CollapseVariance = 'uniform' | 'numbered' | 'mixed';

export type CanvasGroup = {
  id: string;
  kind: CollapseKind;
  /** The normalised shape every member shares. */
  shapeKey: string;
  /** A representative label, from the most common variant. */
  label: string;
  variance: CollapseVariance;
  /**
   * The lowest and highest member label, by a numeric-aware sort — set only when
   * the labels are purely numbering.
   *
   * Deliberately not "first and last in run order": a fan-out's members are in
   * lane order, so `w8` and `w3` would happen to be the ends and would read as
   * the nonsense range `w[8…3]`.
   */
  labelRange: { from: string; to: string } | null;
  /** Member node ids, in run order. */
  memberNodeIDs: string[];
  count: number;
  /** Every level the members occupy, ascending. */
  levels: number[];
  /** Where the collapsed stand-in sits: the first level, and the first lane. */
  level: number;
  lane: number;
  /** Distinct labels inside, most common first. */
  variants: CollapseVariant[];
  /** Member durations in run order — the sparkline, and the shape of the work. */
  durationsMs: number[];
  envelope: CollapseEnvelope;
  /** The worst status inside, so a group holding a failure is red at rest. */
  status: CanvasStatus;
  exceptions: CollapseException[];
};

export type CollapsePlan = {
  groups: CanvasGroup[];
  /** Member node id -> the group that holds it. */
  groupOfNode: ReadonlyMap<string, string>;
  /** How many nodes the canvas draws with every group collapsed. */
  nodesAtRest: number;
  /** How many it draws with none collapsed. */
  nodesExpanded: number;
  /** Inferences made, in the same voice as `CanvasGraph.warnings`. */
  warnings: string[];
};

/** Below this, collapsing costs the reader more than it saves them. */
const MIN_MEMBERS = 3;

/** Push onto a map of lists, creating the list on first use. */
function pushTo<K, V>(map: Map<K, V[]>, key: K, value: V): void {
  const existing = map.get(key);
  if (existing) existing.push(value);
  else map.set(key, [value]);
}

/**
 * Worst-first. A group is drawn as its most alarming member, because the
 * alternative is a green node hiding a red one.
 */
const SEVERITY: Record<CanvasStatus, number> = {
  FAILED: 6,
  CANCELLED: 5,
  RUNNING: 4,
  WAITING: 3,
  QUEUED: 2,
  COMPLETED: 1,
  SKIPPED: 0,
  UNKNOWN: 0,
};

function worstStatus(nodes: CanvasNode[]): CanvasStatus {
  return nodes.reduce<CanvasStatus>(
    (worst, n) => (SEVERITY[n.status] > SEVERITY[worst] ? n.status : worst),
    'UNKNOWN'
  );
}

/** Only these are collapsible; synthetic nodes are structure, not repetition. */
function isWork(node: CanvasNode): boolean {
  return node.kind === 'step' || node.kind === 'wait' || node.kind === 'invoke';
}

/**
 * The shape of a node, for the purpose of deciding whether two of them are the
 * same thing happening twice.
 *
 * Digit runs collapse to `#`, so `think-0` and `think-39` share a shape and
 * `s0`..`s499` are one. This is the inference in this module; `variants` keeps
 * the real labels so nothing is actually lost, and `warnings` says it happened.
 *
 * The SDK's `:N` collision suffix is stripped first and never contributes: it is
 * assigned in encounter order and is not stable across runs under parallelism,
 * so nothing may be keyed on it.
 */
export function shapeKey(node: CanvasNode): string {
  const base = (node.userlandStepID ?? node.label ?? '').replace(/:\d+$/, '');
  const normalised = base.replace(/\d+/g, '#');
  return [node.kind, node.stepOp ?? '', node.waitKind ?? '', normalised].join('|');
}

function durationOf(node: CanvasNode): number {
  if (node.startedAt === null) return 0;
  return Math.max(0, (node.endedAt ?? node.startedAt) - node.startedAt);
}

function median(sorted: number[]): number {
  if (!sorted.length) return 0;
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 ? (sorted[mid - 1]! + sorted[mid]!) / 2 : sorted[mid]!;
}

/**
 * Which members ran long enough to be worth naming.
 *
 * Median absolute deviation rather than a mean and standard deviation: one
 * 30-second outlier in a loop of 50ms steps drags a mean far enough that it
 * stops flagging anything, which is the exact case this needs to catch.
 */
function slowMembers(nodes: CanvasNode[]): Set<string> {
  const durations = nodes.map(durationOf);
  const med = median([...durations].sort((a, b) => a - b));
  const mad = median(durations.map((d) => Math.abs(d - med)).sort((a, b) => a - b));

  // With no spread at all, a fixed multiple of the median is the only sane
  // threshold — and a floor keeps millisecond noise from reading as an outlier.
  const threshold = mad > 0 ? med + 3 * mad : Math.max(med * 3, med + 250);

  // A MAD threshold on its own is not enough, and the failure is a real one.
  // `loop40` simulates a loop that gets steadily slower: 171, 179, …, 213, 299,
  // 335, 370, 1675. The threshold lands at ~335, so 299 is fine and 335 is
  // "slow" — an 11% difference deciding it, inside a continuous ramp. The node
  // then reported "3 slow", naming the top of a trend as if it were three
  // anomalies, when there is one anomaly and a trend.
  //
  // So a member must ALSO be a large multiple of the median. That makes "slow"
  // mean anomalous, which is what the word claims. On `loop40` it now flags
  // only 1675 (12x the median) and says "1 slow", which is true.
  //
  // A monotonic ramp is a real and interesting thing that this does not
  // describe at all — it is simply not described *wrongly* any more.
  const ANOMALY_MULTIPLE = 4;

  const slow = new Set<string>();
  nodes.forEach((node, i) => {
    const d = durations[i]!;
    if (d > 0 && d > threshold && d > med * ANOMALY_MULTIPLE) slow.add(node.id);
  });
  return slow;
}

function buildGroup(
  kind: CollapseKind,
  key: string,
  members: CanvasNode[],
  index: number
): CanvasGroup {
  const byLabel = new Map<string, string[]>();
  for (const node of members) {
    pushTo(byLabel, node.label ?? '', node.id);
  }

  const variants: CollapseVariant[] = [...byLabel.entries()]
    .map(([label, nodeIDs]) => ({ label, count: nodeIDs.length, nodeIDs }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));

  const slow = slowMembers(members);
  const exceptions: CollapseException[] = [];
  for (const node of members) {
    if (node.status === 'FAILED') exceptions.push({ nodeID: node.id, reason: 'failed' });
    else if (node.status === 'CANCELLED') exceptions.push({ nodeID: node.id, reason: 'cancelled' });
    else if (node.status === 'RUNNING' || node.status === 'QUEUED')
      exceptions.push({ nodeID: node.id, reason: 'running' });
    else if (node.attempts > 0) exceptions.push({ nodeID: node.id, reason: 'retried' });
    else if (slow.has(node.id)) exceptions.push({ nodeID: node.id, reason: 'slow' });
  }

  const starts = members.map((n) => n.startedAt ?? n.queuedAt);
  const ends = members.map((n) => n.endedAt ?? n.startedAt ?? n.queuedAt);
  const firstStartedAt = Math.min(...starts);
  const lastStartedAt = Math.max(...starts);

  const levels = [...new Set(members.map((n) => n.level))].sort((a, b) => a - b);

  const variance: CollapseVariance =
    variants.length === 1 ? 'uniform' : variants.length === members.length ? 'numbered' : 'mixed';

  // `numeric: true` so `w2` sorts before `w11` rather than after it.
  const sortedLabels =
    variance === 'numbered'
      ? [...byLabel.keys()].sort((a, b) =>
          a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
        )
      : [];

  return {
    id: `canvas:group:${kind}:${index}`,
    kind,
    shapeKey: key,
    label: variants[0]?.label ?? '',
    variance,
    labelRange: sortedLabels.length
      ? { from: sortedLabels[0]!, to: sortedLabels[sortedLabels.length - 1]! }
      : null,
    memberNodeIDs: members.map((n) => n.id),
    count: members.length,
    levels,
    level: levels[0]!,
    lane: Math.min(...members.map((n) => n.lane)),
    variants,
    durationsMs: members.map(durationOf),
    envelope: {
      firstStartedAt,
      lastStartedAt,
      lastEndedAt: Math.max(...ends),
      staggerMs: lastStartedAt - firstStartedAt,
    },
    status: worstStatus(members),
    exceptions,
  };
}

/**
 * The shape signature of a whole level, used to spot a repeating loop body.
 * Sorted, so a level whose two branches finished in a different order than last
 * time is still recognised as the same iteration.
 */
function levelSignature(nodes: CanvasNode[]): string {
  return nodes.map(shapeKey).sort().join('+');
}

/**
 * Find the loop.
 *
 * Walks the level signatures looking for the shortest period `p` whose pattern
 * repeats at least `MIN_MEMBERS` times in a row. A sequential loop of one step
 * per level has p=1; a think/act agent loop has p=2. Returns the run of levels
 * and the period, or null.
 */
function findRepeat(
  signatures: string[],
  from: number
): { period: number; repeats: number } | null {
  const available = signatures.length - from;
  for (let period = 1; period * MIN_MEMBERS <= available; period++) {
    let repeats = 1;
    while (
      from + (repeats + 1) * period <= signatures.length &&
      Array.from({ length: period }).every(
        (_, k) => signatures[from + k] === signatures[from + repeats * period + k]
      )
    ) {
      repeats += 1;
    }
    if (repeats >= MIN_MEMBERS) return { period, repeats };
  }
  return null;
}

/**
 * Decide what to collapse.
 *
 * Iterations are found first and win: a loop whose body is a fan-out should read
 * as a loop, not as a pile of unrelated fan-outs.
 */
export function planCollapse(graph: CanvasGraph): CollapsePlan {
  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  const workPerLevel: CanvasNode[][] = graph.levels.map((ids) =>
    ids.map((id) => byID.get(id)!).filter(isWork)
  );

  const groups: CanvasGroup[] = [];
  const groupOfNode = new Map<string, string>();
  const warnings: string[] = [];
  const claimed = new Set<string>();

  // --- iterations ---------------------------------------------------------
  // Only levels that actually hold work can take part; a level of nothing but
  // junctions would otherwise break a run of otherwise-identical iterations.
  const signatures = workPerLevel.map(levelSignature);

  let cursor = 0;
  while (cursor < signatures.length) {
    if (!signatures[cursor]) {
      cursor += 1;
      continue;
    }
    const found = findRepeat(signatures, cursor);
    if (!found) {
      cursor += 1;
      continue;
    }

    const { period, repeats } = found;
    // One group per position in the loop body, so a think/act loop draws as
    // `think x 40` then `act x 40` — the body, not 80 separate things.
    for (let k = 0; k < period; k++) {
      const membersByKey = new Map<string, CanvasNode[]>();
      for (let r = 0; r < repeats; r++) {
        for (const node of workPerLevel[cursor + r * period + k] ?? []) {
          pushTo(membersByKey, shapeKey(node), node);
        }
      }
      for (const [key, members] of membersByKey) {
        if (members.length < MIN_MEMBERS) continue;

        // An iteration is a loop body repeating over TIME, so each repetition
        // should be its own level. If a level holds several members of the same
        // shape, they ran in parallel — they are branches, not iterations, and
        // fusing them claims the run did one thing N times when it did N things
        // once each.
        //
        // `v4branches-balanced` is the case: six parallel branches of four
        // sequential steps, named `br0-1` … `br5-4`. Both digit runs normalise,
        // so all 24 shared a shape and collapsed into a single node asserting
        // "one step that ran 24 times". The sibling pass below handles each
        // level properly instead.
        const perLevel = new Map<number, number>();
        for (const node of members) {
          perLevel.set(node.level, (perLevel.get(node.level) ?? 0) + 1);
        }
        if ([...perLevel.values()].some((n) => n > 1)) continue;

        const group = buildGroup('iteration', key, members, groups.length);
        groups.push(group);
        for (const node of members) {
          groupOfNode.set(node.id, group.id);
          claimed.add(node.id);
        }
      }
    }
    cursor += repeats * period;
  }

  // --- siblings -----------------------------------------------------------
  for (const level of workPerLevel) {
    const membersByKey = new Map<string, CanvasNode[]>();
    for (const node of level) {
      if (claimed.has(node.id)) continue;
      pushTo(membersByKey, shapeKey(node), node);
    }
    for (const [key, members] of membersByKey) {
      if (members.length < MIN_MEMBERS) continue;
      const group = buildGroup('siblings', key, members, groups.length);
      groups.push(group);
      for (const node of members) {
        groupOfNode.set(node.id, group.id);
        claimed.add(node.id);
      }
    }
  }

  if (groups.length) {
    const normalised = groups.filter((g) => g.variants.some((v) => /\d/.test(v.label))).length;
    if (normalised > 0) {
      warnings.push(
        'Repeated steps are grouped by name with numbers ignored, so `step-1` and `step-2` count as the same shape. Expand a group to see the individual steps.'
      );
    }
  }

  const nodesExpanded = graph.nodes.length;
  const collapsedAway = groups.reduce((n, g) => n + g.count, 0);

  return {
    groups,
    groupOfNode,
    nodesAtRest: nodesExpanded - collapsedAway + groups.length,
    nodesExpanded,
    warnings,
  };
}

/**
 * A short, readable name for a group.
 *
 * Numbered members share everything but the index, so `think-0` and `think-39`
 * read as `think-[0…39]` rather than as a truncated `think-0 … thin…`. The
 * common prefix and suffix are found rather than assumed, so `run-3-of-8` and
 * `run-91-of-8` still produce something sensible.
 */
export function groupTitle(group: CanvasGroup): string {
  if (!group.labelRange) return group.label;

  const { from: a, to: b } = group.labelRange;
  if (!a || !b || a === b) return group.label;

  let prefix = 0;
  while (prefix < a.length && prefix < b.length && a[prefix] === b[prefix]) prefix++;

  let suffix = 0;
  while (
    suffix < a.length - prefix &&
    suffix < b.length - prefix &&
    a[a.length - 1 - suffix] === b[b.length - 1 - suffix]
  ) {
    suffix++;
  }

  // Only call it a range when what differs really is just a number. On
  // `v4branches-balanced` — six branches of four steps, named `br0-1` …
  // `br5-4` — two digit runs vary independently, and `br[0-1…5-4]` reads as a
  // range while being nothing of the sort.
  const head = a.slice(0, prefix);
  const tail = suffix > 0 ? a.slice(a.length - suffix) : '';
  const from = a.slice(prefix, a.length - suffix);
  const to = b.slice(prefix, b.length - suffix);

  // Nothing in common — fall back to naming the first member rather than
  // producing something that reads like a range but is not one.
  if (!from || !to) return group.label;
  if (!/^\d+$/.test(from) || !/^\d+$/.test(to)) return group.label;

  return `${head}[${from}…${to}]${tail}`;
}

/**
 * Whether a run is better opened collapsed.
 *
 * Node count alone is the wrong test. A 12-wide fan-out is only 17 nodes and
 * still unreadable, because the problem is twelve lanes in a 300px pane rather
 * than the total. So each dimension gets its own budget, and a run that busts
 * any of them — and that collapsing would actually help — starts aggregated.
 */
export function shouldAggregate(graph: CanvasGraph, plan: CollapsePlan): boolean {
  if (!plan.groups.length) return false;
  if (plan.nodesAtRest >= plan.nodesExpanded) return false;

  const widestLevel = graph.levels.reduce((n, ids) => Math.max(n, ids.length), 0);
  return plan.nodesExpanded > 40 || graph.levels.length > 12 || widestLevel > 6;
}

/**
 * Which edge kind survives when collapsing merges two edges into one.
 *
 * Higher wins. An asserted dependency must not be downgraded to a broken line
 * just because some other member of the group had a weaker relationship — but
 * equally, a group's edge must not be *promoted* to a dependency that no member
 * actually had. Taking the strongest present is the honest reading: at least one
 * member really did wait.
 */
const EDGE_RANK: Record<CanvasEdge['kind'], number> = {
  sequence: 4,
  fanIn: 3,
  fanOut: 3,
  alternate: 2,
  unconfirmed: 1,
};

export type CollapsedGraph = {
  graph: CanvasGraph;
  /** The group behind each `kind: 'group'` node. */
  groupByNodeID: ReadonlyMap<string, CanvasGroup>;
};

/**
 * Rewrite a graph with the given groups collapsed.
 *
 * The result is an ordinary `CanvasGraph`, so the renderer needs no idea that
 * collapsing exists: it still reads (level, lane) and draws. Levels emptied by
 * collapsing are removed and the rest renumbered, which is what turns 502
 * columns into three.
 *
 * `expanded` names groups the user has opened; those are left alone.
 */
export function applyCollapse(
  graph: CanvasGraph,
  plan: CollapsePlan,
  expanded: ReadonlySet<string> = new Set()
): CollapsedGraph {
  const active = plan.groups.filter((g) => !expanded.has(g.id));
  if (!active.length) {
    return { graph, groupByNodeID: new Map() };
  }

  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  const groupByNodeID = new Map<string, CanvasGroup>();
  /** member node id -> the group node that replaces it */
  const replacement = new Map<string, string>();

  const groupNodes = new Map<number, CanvasNode[]>();
  for (const group of active) {
    const members = group.memberNodeIDs.map((id) => byID.get(id)!).filter(Boolean);
    if (!members.length) continue;

    const first = members[0]!;
    const node: CanvasNode = {
      ...first,
      id: group.id,
      kind: 'group',
      label: group.label,
      status: group.status,
      level: group.level,
      lane: group.lane,
      // Selecting a group selects its first member, so the existing step panel
      // opens on something real rather than on nothing.
      spanID: first.spanID,
      queuedAt: Math.min(...members.map((m) => m.queuedAt)),
      startedAt: group.envelope.firstStartedAt,
      endedAt: group.envelope.lastEndedAt,
      attempts: Math.max(...members.map((m) => m.attempts)),
      parentNodeID: null,
      parentNodeIDs: null,
      discovery: null,
      junction: null,
    };

    groupByNodeID.set(group.id, group);
    pushTo(groupNodes, group.level, node);
    for (const id of group.memberNodeIDs) replacement.set(id, group.id);
  }

  // --- levels -------------------------------------------------------------
  // `graph.levels` holds only the nodes that sit *on* a level. Junctions do not:
  // `insertJunctions` adds them after levelling, on half-levels between two
  // columns. Rebuilding from `levels` alone would drop them, which severs the
  // graph — a fan-out and its collect step would end up with nothing between
  // them. So they are carried separately and repositioned from their edges.
  const onLevel = new Set(graph.levels.flat());

  const rebuilt: string[][] = [];
  const levelOfNode = new Map<string, number>();
  const kept: CanvasNode[] = [];

  for (const [oldLevel, ids] of graph.levels.entries()) {
    const survivors: CanvasNode[] = [];
    for (const id of ids) {
      if (replacement.has(id)) continue;
      const node = byID.get(id);
      if (node) survivors.push(node);
    }
    survivors.push(...(groupNodes.get(oldLevel) ?? []));
    if (!survivors.length) continue;

    const newLevel = rebuilt.length;
    rebuilt.push(survivors.map((n) => n.id));
    for (const node of survivors) {
      levelOfNode.set(node.id, newLevel);
      kept.push({ ...node, level: newLevel });
    }
  }

  const offLevel = graph.nodes.filter((n) => !onLevel.has(n.id) && !replacement.has(n.id));
  const surviving = new Set([...levelOfNode.keys(), ...offLevel.map((n) => n.id)]);

  // --- edges --------------------------------------------------------------
  const merged = new Map<string, CanvasEdge>();
  for (const edge of graph.edges) {
    const from = replacement.get(edge.from) ?? edge.from;
    const to = replacement.get(edge.to) ?? edge.to;

    // An edge between two members of the same group described a hop that is now
    // inside the collapsed node; there is nothing left to draw it between.
    if (from === to) continue;
    if (!surviving.has(from) || !surviving.has(to)) continue;

    const key = `${from}->${to}`;
    const existing = merged.get(key);
    if (!existing || EDGE_RANK[edge.kind] > EDGE_RANK[existing.kind]) {
      merged.set(key, { id: key, from, to, kind: edge.kind });
    }
  }

  const edges = [...merged.values()];

  // Place each junction between the columns it actually joins, now that those
  // columns have been renumbered. Half-levels are what `toFlowElements` expects.
  for (const node of offLevel) {
    const before = edges
      .filter((e) => e.to === node.id)
      .map((e) => levelOfNode.get(e.from))
      .filter((l): l is number => l !== undefined);
    const after = edges
      .filter((e) => e.from === node.id)
      .map((e) => levelOfNode.get(e.to))
      .filter((l): l is number => l !== undefined);

    const left = before.length ? Math.max(...before) : null;
    const right = after.length ? Math.min(...after) : null;
    const level =
      left !== null && right !== null ? (left + right) / 2 : left ?? right ?? node.level;

    kept.push({ ...node, level });
  }

  return {
    graph: {
      ...graph,
      nodes: kept,
      edges,
      levels: rebuilt,
      warnings: [...graph.warnings, ...plan.warnings],
    },
    groupByNodeID,
  };
}

/**
 * The same collapsing, applied to timeline rows.
 *
 * The detection is shared with the canvas — the whole point of doing it once in
 * the model — and this only decides how to draw the result. A group becomes one
 * row whose `children` are its members, which means the timeline's existing
 * expansion affordance opens it: no new interaction, and the row keeps its
 * expander, tooltip, selection and keyboard behaviour for free.
 *
 * The row spans the envelope, first start to last end, so a fan-out's stagger is
 * still legible as a bar rather than being averaged into a point.
 */
/**
 * One segment per member, positioned within the group's envelope.
 *
 * Widths are real, so the picture is honest about where the time went, but each
 * mark is floored at a visible width — 500 steps of 3ms across a 1.9s envelope
 * would otherwise be invisible, and an invisible mark answers nothing. A small
 * gap is left between marks so a run of them reads as *many* rather than as one
 * continuous block.
 */
function memberSegments(
  kind: CollapseKind,
  groupID: string,
  rows: TimelineBarData[],
  envelope: CollapseEnvelope,
  maxMarks: number
): BarSegment[] {
  const span = envelope.lastEndedAt - envelope.firstStartedAt;
  if (span <= 0) return [];

  // Past this, marks are closer together than they can be drawn apart: at 500
  // members in a few hundred pixels the pitch is about a pixel, so every mark
  // floors to the same minimum width and the row renders as one solid block
  // with ragged edges. The cap comes from the measured plot width, so a narrow
  // pane gets fewer, larger marks rather than mush. Bucketed members take their
  // worst member's status — the same rule the group node uses, so a bucket
  // holding a failure is red.
  // A FAN-OUT IS NOT A SEQUENCE.
  //
  // Evenly-pitched marks say "one, then the next, then the next", which is
  // right for a loop body and a lie for a fan-out: `wide`'s twelve steps all
  // started within 14ms of each other and were drawn as twelve equal ticks
  // spread evenly across the row with white gaps between them, reading as
  // twelve sequential 8ms steps. The minimap directly above drew the same
  // twelve overlapping across twelve lanes — the two strips disagreed about
  // the shape of the run, touching each other on screen.
  //
  // So a siblings group draws its members where they actually were. They
  // overlap, they merge into a block, and that block IS the answer: these
  // happened at once.
  if (kind === 'siblings') return concurrentSegments(groupID, rows, envelope);

  const marks = Math.max(1, Math.min(rows.length, maxMarks));

  // Every mark the same size with an even gap. Widths proportional to duration
  // were tried and read as ragged: the marks say *where in the sequence*, and
  // the row's own label already says how long the whole thing took.
  const pitch = 100 / marks;
  const width = pitch * 0.62;

  const buckets: Array<{ status?: string; index: number; names: string[] }> = [];
  for (let i = 0; i < marks; i++) buckets.push({ index: i, names: [] });

  rows.forEach((row, i) => {
    const bucket = buckets[Math.min(marks - 1, Math.floor((i / rows.length) * marks))]!;
    const severity = (s?: string) => SEVERITY[(s ?? 'UNKNOWN') as CanvasStatus] ?? 0;
    if (severity(row.status) > severity(bucket.status)) bucket.status = row.status;
    bucket.names.push(row.name);
  });

  return buckets.map((bucket) => ({
    id: `${groupID}-member-${bucket.index}`,
    startPercent: bucket.index * pitch,
    widthPercent: width,
    style: 'step.run' as const,
    status: bucket.status,
    // Which members a mark stands for. When several are bucketed into one the
    // mark takes the worst status of them, so saying only the status would hide
    // that the red covers four steps of which one failed.
    tooltip:
      bucket.names.length === 1
        ? `${bucket.names[0]}${bucket.status ? ` — ${bucket.status.toLowerCase()}` : ''}`
        : `${bucket.names.length} steps: ${bucket.names.slice(0, 4).join(', ')}${
            bucket.names.length > 4 ? ', …' : ''
          }${bucket.status ? ` — worst: ${bucket.status.toLowerCase()}` : ''}`,
  }));
}

/**
 * Members of a fan-out, each at its own real interval within the envelope.
 *
 * Overlap is the point rather than a problem to lay out around: steps that ran
 * at the same time are drawn on top of each other, so the row reads as one
 * solid stretch of concurrent work instead of a tidy sequence that never
 * happened. Widths are floored so a 1ms member is still findable.
 */
function concurrentSegments(
  groupID: string,
  rows: TimelineBarData[],
  envelope: CollapseEnvelope
): BarSegment[] {
  const span = envelope.lastEndedAt - envelope.firstStartedAt;
  if (span <= 0) return [];

  return rows.map((row, i) => {
    // When it RAN, not when it was queued — the envelope is built from the same
    // instant. Measuring from `startTime` put every member of `wide` at exactly
    // 0%, because a fan-out queues all twelve together, and threw away the
    // stagger between their starts. That stagger is a picture of the
    // concurrency limit and is the one thing a collapsed fan-out has to keep.
    const startMs = row.startTime.getTime() + (row.delayMs ?? 0);
    const endMs = Math.max(startMs, (row.endTime ?? row.startTime).getTime());
    const startPercent = ((startMs - envelope.firstStartedAt) / span) * 100;

    return {
      id: `${groupID}-member-${i}`,
      startPercent: Math.max(0, Math.min(100, startPercent)),
      widthPercent: Math.max(1.2, ((endMs - startMs) / span) * 100),
      style: 'step.run' as const,
      status: row.status,
      tooltip: `${row.name}${row.status ? ` — ${row.status.toLowerCase()}` : ''}`,
    };
  });
}

/** Roughly the narrowest a mark plus its gap can be and still read as two. */
const MARK_PITCH_PX = 7;

/**
 * How many marks a group row can usefully draw.
 *
 * Derived from the measured plot width rather than fixed, so a narrow pane gets
 * fewer, larger marks instead of a smear. Falls back to a middling count when
 * the width is not known yet (first paint, or a test with no layout).
 */
export function markBudget(plotWidthPx: number | undefined): number {
  if (!plotWidthPx || plotWidthPx <= 0) return 40;
  return Math.max(6, Math.floor(plotWidthPx / MARK_PITCH_PX));
}

export function applyCollapseToBars(
  bars: TimelineBarData[],
  plan: CollapsePlan,
  expanded: ReadonlySet<string> = new Set(),
  maxMarks = 40
): TimelineBarData[] {
  const active = plan.groups.filter((g) => !expanded.has(g.id));
  if (!active.length) return bars;

  const groupByID = new Map(active.map((g) => [g.id, g]));
  const groupOfBar = new Map<string, string>();
  for (const group of active) {
    for (const id of group.memberNodeIDs) groupOfBar.set(id, group.id);
  }

  const walk = (list: TimelineBarData[]): TimelineBarData[] => {
    const out: TimelineBarData[] = [];
    const members = new Map<string, TimelineBarData[]>();
    /** Where each group's row goes: the position of its first member. */
    const slots = new Map<string, number>();

    for (const bar of list) {
      const groupID = groupOfBar.get(bar.id);
      if (!groupID) {
        out.push({ ...bar, children: bar.children ? walk(bar.children) : undefined });
        continue;
      }
      if (!members.has(groupID)) {
        slots.set(groupID, out.length);
        out.push(bar); // placeholder, replaced below
      }
      pushTo(members, groupID, bar);
    }

    for (const [groupID, slot] of slots) {
      const group = groupByID.get(groupID)!;
      const rows = members.get(groupID)!;
      const first = rows[0]!;

      out[slot] = {
        ...first,
        id: group.id,
        name: `${groupTitle(group)} × ${group.count}`,
        // From the earliest member's queuedAt, not the envelope's first START.
        // Every other row in the timeline begins at queuedAt, so an envelope
        // start made the group row begin AFTER its own children — an 88ms
        // overhang on `wide` — and silently dropped the queued phase that
        // sibling rows all include.
        startTime: new Date(Math.min(...rows.map((r) => r.startTime.getTime()))),
        endTime: new Date(group.envelope.lastEndedAt),
        status: group.status,
        children: rows,
        // Draw the members themselves rather than one continuous block, each
        // coloured by its own status. A solid bar would say only "something
        // happened here for 1.9s"; a run of small marks says *where in the
        // sequence* the failures were, which is the question a collapsed row
        // otherwise forces you to expand to answer.
        segments: memberSegments(group.kind, group.id, rows, group.envelope, maxMarks),
        // A group is a summary, not a span: the per-step phase breakdowns belong
        // to the members and are shown when it is expanded. Claiming one step's
        // discovery or HTTP timing as the group's would be a fabrication.
        timingBreakdown: undefined,
        httpTimingBreakdown: undefined,
        inngestBreakdown: undefined,
        runInngestBreakdown: undefined,
      };
    }

    return out;
  };

  return walk(bars);
}
