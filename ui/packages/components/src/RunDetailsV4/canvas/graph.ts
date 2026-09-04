/**
 * Turns a run's trace into the Run Canvas graph model.
 *
 * Pure: no React, no rendering, no fetching. Input is the *rolled up* trace
 * (`traceRollup`'s output), because rollup is what collapses retry attempts,
 * names the finalization span, and — critically — de-duplicates the two spans
 * the executor emits per step. See PARALLELISM below.
 */
import {
  isStepInfoInvoke,
  isStepInfoSignal,
  isStepInfoWait,
  type RunDiscovery,
  type Trace,
} from '../types';
import type {
  CanvasEdge,
  CanvasGraph,
  CanvasNode,
  CanvasNodeKind,
  CanvasStatus,
  CanvasTrigger,
  ParallelismSource,
  WaitKind,
} from './graph.types';

export const RESULT_ID = 'canvas:result';
export const eventID = (i: number) => `canvas:event:${i}`;
const joinID = (level: number) => `canvas:join:${level}`;

const WAIT_OPS = new Set(['SLEEP', 'WAIT_FOR_EVENT', 'WAIT_FOR_SIGNAL']);
const STATUSES = new Set<CanvasStatus>([
  'QUEUED',
  'RUNNING',
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'WAITING',
  'SKIPPED',
]);

function toStatus(raw: string | null | undefined): CanvasStatus {
  const s = (raw ?? '').toUpperCase() as CanvasStatus;
  return STATUSES.has(s) ? s : 'UNKNOWN';
}

const ms = (t: string | null | undefined): number | null => {
  if (!t) return null;
  const n = new Date(t).getTime();
  return Number.isNaN(n) ? null : n;
};

/** A rolled-up child of the run span that we care about. */
type Entry = {
  trace: Trace;
  op: string;
  queuedAt: number;
  startedAt: number | null;
  endedAt: number | null;
};

/**
 * A run's own output spans ("Finalization", "executor.nonstep") are folded into
 * the synthetic result node rather than drawn as work.
 */
function isRunOutputSpan(t: Trace): boolean {
  return !t.stepID && !t.stepOp;
}

function toEntry(t: Trace): Entry {
  return {
    trace: t,
    op: (t.stepOp ?? t.stepType ?? '').toUpperCase(),
    queuedAt: ms(t.queuedAt) ?? 0,
    startedAt: ms(t.startedAt),
    endedAt: ms(t.endedAt),
  };
}

/**
 * PARALLELISM — read this before changing the levelling.
 *
 * There are two sources, and the difference matters.
 *
 * 1. AUTHORITATIVE (`toLevelsFromPlans`). `RunTraceSpan.plannedSteps` carries
 *    every opcode one SDK response returned — the SDK's own statement of what
 *    it planned together. Steps sharing a plan were genuinely fanned out.
 *    Only SDKs on execution version 2+ populate this: inngest-js v4 prefers
 *    `ExecutionVersion.V2`, while v3 prefers V1 and reports one opcode per
 *    response, so a v3 run has no plans to read.
 *
 * 2. INFERRED (`toLevels`). When there are no plans we fall back to observed
 *    interval overlap on the rolled-up spans: steps whose [start, end)
 *    intervals overlap ran at the same time, so they share a level.
 *
 * TODO(canvas): the inferred path is an observation, not the structure. Steps
 * that were genuinely fanned out but happened not to overlap — a concurrency
 * limit of 1, or one branch finishing before the other is dequeued — render as
 * sequential, and nothing in a v1 payload can tell us otherwise. `graph.parallelismSource`
 * reports which path was taken so the UI can say so rather than overstating.
 *
 * Three other candidate signals were measured and rejected:
 *
 *  - `groupID` looks like a batch id and is not. The executor deliberately
 *    gives each parallel opcode its own group (executor.go, `if
 *    group.ShouldStartHistoryGroup`). On a real fan-out the three parallel
 *    steps shared a groupID on their *planned* spans and diverged on their
 *    *execution* spans — and rollup keeps the latter.
 *  - Shared `queuedAt` is real (the executor stamps one `handledAt` across a
 *    whole SDK response, on purpose) but survives only on the planned spans,
 *    which rollup discards. Worse, pre-rollup it false-positives: on a strictly
 *    sequential `step.run -> step.sleep("2s") -> step.run`, the sleep and the
 *    step after it shared both groupID and queuedAt.
 *  - Reading pre-rollup spans instead means reimplementing retry collapsing,
 *    finalization naming and the EXE-1992 duplicate fix. Not worth it.
 *
 * Branch membership — which of `left-2`/`right-2` continues `left-1` — is a
 * separate question, answered by the executor and validated here. See
 * resolveBranches below.
 */
/**
 * Level using the SDK's own statement of what it planned together.
 *
 * `plannedSteps` comes from the `response.step.ops` span attribute: every
 * opcode one SDK response returned. Two steps sharing a plan were genuinely
 * fanned out, with no inference involved. Returns null when no step in the run
 * carries a plan, which is the case for every SDK on execution version 1 —
 * those report one opcode per response, so there is never a batch to record.
 */
function toLevelsFromPlans(entries: Entry[]): Entry[][] | null {
  const planKey = (e: Entry) => {
    const plan = e.trace.plannedSteps;
    if (!plan || plan.length < 2) return null;
    // Order within a response is not meaningful, so sort for a stable key.
    return plan
      .map((s) => s.stepID)
      .slice()
      .sort()
      .join('|');
  };

  if (!entries.some((e) => planKey(e) !== null)) return null;

  const ordered = [...entries].sort(
    (a, b) => (a.startedAt ?? a.queuedAt) - (b.startedAt ?? b.queuedAt)
  );

  const groups: Entry[][] = [];
  const byKey = new Map<string, Entry[]>();

  for (const entry of ordered) {
    const key = planKey(entry);
    if (key === null) {
      groups.push([entry]);
      continue;
    }
    const existing = byKey.get(key);
    if (existing) {
      existing.push(entry);
      continue;
    }
    const group = [entry];
    byKey.set(key, group);
    groups.push(group);
  }

  return groups;
}

function toLevels(entries: Entry[], fallbackEnd: number): Entry[][] {
  const ordered = [...entries].sort(
    (a, b) => (a.startedAt ?? a.queuedAt) - (b.startedAt ?? b.queuedAt)
  );

  const levels: Entry[][] = [];
  let current: Entry[] = [];
  let currentEnd = -Infinity;

  for (const entry of ordered) {
    const start = entry.startedAt ?? entry.queuedAt;
    // An unfinished step is treated as running to the end of the known window,
    // so in-flight siblings cluster together rather than each starting a level.
    const end = entry.endedAt ?? fallbackEnd;

    if (current.length === 0 || start < currentEnd) {
      current.push(entry);
      currentEnd = Math.max(currentEnd, end);
    } else {
      levels.push(current);
      current = [entry];
      currentEnd = end;
    }
  }
  if (current.length > 0) levels.push(current);

  return levels;
}

/**
 * A circle wherever the run changes width: one step becoming several, or
 * several becoming one.
 *
 * This is where the SDK actually made a discovery — it ran the function body
 * until it fanned out, or waited for a set of steps before continuing — and it
 * is the moment the shape of the run changed. Drawing the convergence once, as
 * a point, also stops a wide fan-in reading as a bipartite mesh of lines.
 *
 * A race's loser is kept out: it is *known* not to have contributed, so routing
 * it through the convergence would say the opposite. An unconfirmed edge is
 * routed through, because it only exists where the source had already finished
 * before the target started — so the convergence is true in time either way,
 * and the line stays broken to carry the doubt about dependency. Leaving those
 * out drew a circle on the fan-out of a run and nothing at all where the same
 * run came back together.
 */
function insertJunctions(nodes: CanvasNode[], edges: CanvasEdge[]): void {
  const byID = new Map(nodes.map((n) => [n.id, n]));

  // A junction between two junctions would be meaningless, and the fallback
  // path already places one where membership could not be resolved.
  const spannable = (e: CanvasEdge) =>
    e.kind !== 'alternate' && byID.get(e.from)?.kind !== 'join' && byID.get(e.to)?.kind !== 'join';

  // One rule, both directions: group the proven edges by the exact set of
  // sources feeding a step. Every step sharing a set is fed by the same
  // discovery, so the set and its dependants are one bundle.
  //
  //   {a}       -> {b}        1:1, no junction
  //   {a}       -> {b, c, d}  single -> parallel
  //   {a, b, c} -> {d}        parallel -> single
  //   {a, b}    -> {c, d}     both at once, and still one circle
  const bundles = new Map<
    string,
    { sources: CanvasNode[]; targets: CanvasNode[]; incoming: CanvasEdge[] }
  >();

  for (const edge of edges) {
    if (!spannable(edge)) continue;
    const target = byID.get(edge.to);
    if (!target) continue;

    const incoming = edges.filter((e) => e.to === edge.to && spannable(e));
    const sources = incoming
      .map((e) => byID.get(e.from))
      .filter((n): n is CanvasNode => Boolean(n));

    const key = sources
      .map((n) => n.id)
      .sort()
      .join('|');

    const existing = bundles.get(key);
    if (existing) {
      if (!existing.targets.includes(target)) existing.targets.push(target);
    } else {
      bundles.set(key, { sources, targets: [target], incoming });
    }
  }

  const mean = (values: number[]) => values.reduce((a, b) => a + b, 0) / values.length;
  let created = 0;

  for (const { sources, targets, incoming } of bundles.values()) {
    // A one-to-one hop is just a line. Anything else changes the width of the
    // run, which is exactly where a discovery happened.
    if (sources.length < 2 && targets.length < 2) continue;

    const junction: CanvasNode = {
      id: `junction-${created++}`,
      kind: 'join',
      label: '',
      status: [...sources, ...targets].some((n) => n.status === 'FAILED') ? 'FAILED' : 'COMPLETED',
      // Halfway between what feeds it and what follows, so it reads as the
      // point the lines meet rather than as a step of its own.
      level: (mean(sources.map((n) => n.level)) + mean(targets.map((n) => n.level))) / 2,
      lane: mean([...sources, ...targets].map((n) => n.lane)),
      spanID: '',
      stepID: null,
      stepOp: null,
      attempts: 0,
      queuedAt: Math.min(...targets.map((n) => n.queuedAt)),
      startedAt: null,
      endedAt: null,
      junction: {
        direction:
          sources.length > 1 && targets.length > 1
            ? 'both'
            : sources.length > 1
            ? 'converge'
            : 'diverge',
        from: sources.length,
        unproven: incoming.filter((e) => e.kind === 'unconfirmed').length,
        to: targets.length,
      },
    };
    nodes.push(junction);

    const targetIDs = new Set(targets.map((n) => n.id));
    for (let i = edges.length - 1; i >= 0; i--) {
      const edge = edges[i]!;
      if (spannable(edge) && targetIDs.has(edge.to)) edges.splice(i, 1);
    }

    for (const source of sources) {
      const original = incoming.find((e) => e.from === source.id);
      edges.push({
        id: `${source.id}->${junction.id}`,
        from: source.id,
        to: junction.id,
        // An unproven contributor keeps its broken line all the way in.
        kind:
          original?.kind === 'unconfirmed'
            ? 'unconfirmed'
            : sources.length > 1
            ? 'fanIn'
            : 'sequence',
      });
    }
    for (const target of targets) {
      edges.push({
        id: `${junction.id}->${target.id}`,
        from: junction.id,
        to: target.id,
        kind: targets.length > 1 ? 'fanOut' : 'sequence',
      });
    }
  }
}

/**
 * Ties each junction to the SDK request that produced it, so the circle is a
 * real thing that happened rather than a drawing device.
 *
 * Two ways to match, because a discovery reports what it *planned* and not
 * everything it did:
 *
 *  - A fan-out planned its targets outright, so the plan names them exactly.
 *  - A step discovered on its own is executed inline and checkpointed rather
 *    than planned, so it appears in no plan at all. There the request is the one
 *    whose window the step ran inside.
 *
 * Anything unmatched is left as a plain circle. A junction is worth drawing
 * either way; the discovery only adds detail to it.
 */
function attachDiscoveries(
  nodes: CanvasNode[],
  edges: CanvasEdge[],
  discoveries: RunDiscovery[]
): void {
  const junctions = nodes.filter((n) => n.kind === 'join');
  if (!junctions.length || !discoveries.length) return;

  const byID = new Map(nodes.map((n) => [n.id, n]));

  const labelByStepID = new Map<string, string>();
  for (const node of nodes) {
    if (node.stepID) labelByStepID.set(node.stepID, node.label);
  }

  const byPlan = new Map<string, RunDiscovery>();
  for (const discovery of discoveries) {
    if (discovery.plannedStepIDs.length < 2) continue;
    byPlan.set([...discovery.plannedStepIDs].sort().join('|'), discovery);
  }

  const taken = new Set<RunDiscovery>();

  for (const junction of junctions) {
    // Read the junction's own outgoing edges. `parentNodeID` still names the
    // steps a node waited for, which is the lineage, not the layout.
    const targets = edges
      .filter((e) => e.from === junction.id)
      .map((e) => byID.get(e.to))
      .filter((n): n is CanvasNode => Boolean(n));

    if (!targets.length) continue;

    const key = targets
      .map((n) => n.stepID)
      .filter((id): id is string => Boolean(id))
      .sort()
      .join('|');

    let match = byPlan.get(key);

    if (!match) {
      // The inline case: one step, run inside the request that found it.
      match = discoveries.find((discovery) => {
        if (taken.has(discovery)) return false;
        const from = ms(discovery.startedAt);
        const to = ms(discovery.endedAt);
        if (from === null || to === null) return false;
        return targets.every(
          (target) =>
            target.startedAt !== null && target.startedAt >= from && target.startedAt <= to
        );
      });
    }

    if (!match) continue;
    taken.add(match);

    junction.spanID = match.spanID;
    junction.startedAt = ms(match.startedAt);
    junction.endedAt = ms(match.endedAt);
    junction.discovery = {
      spanID: match.spanID,
      startedAt: ms(match.startedAt),
      endedAt: ms(match.endedAt),
      // What the run did next, rather than the raw plan. A fan-out planned
      // exactly these; a step discovered on its own was run by this request
      // without ever being planned, and its id appears in no plan at all.
      next: targets.map((n) => n.label),
    };
  }
}

function waitKindOf(op: string): WaitKind {
  if (op === 'SLEEP') return 'sleep';
  if (op === 'WAIT_FOR_SIGNAL') return 'waitForSignal';
  return 'waitForEvent';
}

function nodeKind(entry: Entry): CanvasNodeKind {
  if (WAIT_OPS.has(entry.op)) return 'wait';
  return entry.op === 'INVOKE' ? 'invoke' : 'step';
}

/**
 * Whether this step ended by expiry rather than by succeeding or failing.
 * Invokes can time out too, not just waits, and a timeout is neither a success
 * nor a step failure — the canvas greys it rather than colouring it.
 */
function didTimeOut(entry: Entry): boolean | null {
  const info = entry.trace.stepInfo;
  if (isStepInfoWait(info) || isStepInfoSignal(info) || isStepInfoInvoke(info)) {
    return info.timedOut;
  }
  return null;
}

/**
 * BRANCH MEMBERSHIP — which steps each step waited for.
 *
 * `parentStepIDs` comes from the SDK, not from inference here or in the
 * executor, and it is a *set*. The SDK resumes memoised steps one at a time,
 * draining microtasks between each, so a step discovered while a given step is
 * being resumed was reached by a continuation waiting on it. Where that
 * continuation was waiting on a promise combinator, the SDK sees the combinator
 * being built and names every member of it. Both are observations of the
 * promise graph, made in the only place it is visible.
 *
 * Three earlier attempts inferred it downstream and all were unsound, because
 * ordering cannot distinguish independent chains from a join that re-fans out:
 *   - pairing on span end timestamps, client-side: ~6% of pairings wrong
 *   - pairing the k-th new step with the k-th recent completion, in the
 *     executor: correct on balanced shapes, wrong on ragged ones and on joins
 *   - a single parent per step: true but partial on a join, which left one
 *     branch of every `Promise.all` looking like a dead end
 *
 * With the set in hand a join is drawable, so this no longer declines one. It
 * still declines in two cases, both of which mean our account is incomplete
 * rather than that the run was shaped that way.
 *
 * The first is disagreement: a named parent that is not in the previous level.
 *
 * The second is a previous-level step that nothing names. That is either a real
 * dead end (`const pa = step.run(a), pb = step.run(b); await pa; step.run(c);
 * await pb` — c starts while b is still running) or a join the SDK could not
 * see (`Promise.all([chain(), chain()])`, where the combinator is over branch
 * *functions* and there is no step promise to observe). Those need opposite
 * drawings, and no reported field separates them.
 *
 * The trace narrows it, though, by rejecting rather than inferring: a step that
 * had not finished when the next level started cannot have gated it. So an
 * unnamed step that was still running is a genuine dead end and the drawing is
 * complete. An unnamed step that *had* finished in time is the ambiguous case,
 * and it is drawn as an `unconfirmed` edge — a broken line that claims only
 * that the ordering allows it. That is true under both readings, and it is why
 * the hop no longer has to be abandoned to a junction to stay honest.
 *
 * Timing is used only to rule an edge *out*. Earlier rounds tried to infer
 * edges from timestamps and were ~6% wrong; ruling out is sound where ruling in
 * is not.
 *
 * A step legitimately maps to several parents (a join) and a parent
 * legitimately maps to several children (a branch that fans out), so this is
 * neither a bijection nor a function — it is the DAG's edge set.
 */
type Branches = {
  /** Reported dependencies: solid edges. */
  pairs: Map<Entry, Entry[]>;
  /** Could have gated this level; nothing says it did. Broken edges. */
  unconfirmed: Entry[];
};

function resolveBranches(prev: Entry[], cur: Entry[]): Branches | null {
  if (prev.length < 2 || cur.length < 1) return null;

  const prevByStepID = new Map<string, Entry>();
  for (const entry of prev) {
    if (entry.trace.stepID) prevByStepID.set(entry.trace.stepID, entry);
  }
  if (prevByStepID.size !== prev.length) return null;

  const pairs = new Map<Entry, Entry[]>();
  const named = new Set<Entry>();

  for (const entry of cur) {
    const parentIDs = entry.trace.parentStepIDs;
    if (!parentIDs?.length) return null;

    const parents: Entry[] = [];
    for (const parentID of parentIDs) {
      const parent = prevByStepID.get(parentID);
      // A parent outside the previous level means the levelling and the SDK
      // disagree about what ran together; do not guess.
      if (!parent) return null;
      parents.push(parent);
      named.add(parent);
    }

    // A race's losers are accounted for too: they are named, just not as
    // dependencies.
    for (const id of entry.trace.parentAlternateStepIDs ?? []) {
      const alternate = prevByStepID.get(id);
      if (alternate) named.add(alternate);
    }

    pairs.set(entry, parents);
  }

  const levelStart = Math.min(...cur.map((e) => e.startedAt ?? e.queuedAt));
  const unconfirmed: Entry[] = [];

  for (const entry of prev) {
    if (named.has(entry)) continue;

    // Still running when the next level began, so it gated nothing: a real dead
    // end, and the edges we have are the whole picture.
    if (entry.endedAt === null || entry.endedAt > levelStart) continue;

    // Finished in time, and nothing named it. Drawn, but broken.
    unconfirmed.push(entry);
  }

  // Only where this level converges: one step, or several sharing parents. When
  // each step continues a branch of its own, an unnamed step is a branch that
  // ended, and drawing it into every sibling would suggest it gated four
  // unrelated chains. A convergence is the shape where the missing edge would
  // actually be a join member, which is the case worth showing.
  const distinctNamed = new Set<Entry>();
  for (const parents of pairs.values()) {
    for (const parent of parents) distinctNamed.add(parent);
  }

  const converges = cur.length === 1 || distinctNamed.size < cur.length;

  return { pairs, unconfirmed: converges ? unconfirmed : [] };
}

/**
 * The losing side of a race, resolved against the previous level. These are
 * drawn dashed and never affect layout, so an unresolvable one is simply
 * dropped rather than declining the whole hop.
 */
function resolveAlternates(prev: Entry[], entry: Entry): Entry[] {
  const ids = entry.trace.parentAlternateStepIDs;
  if (!ids?.length) return [];

  const out: Entry[] = [];
  for (const id of ids) {
    const match = prev.find((candidate) => candidate.trace.stepID === id);
    if (match) out.push(match);
  }
  return out;
}

export function toCanvasGraph(root: Trace, trigger?: CanvasTrigger | null): CanvasGraph {
  const warnings: string[] = [];

  const work: Entry[] = [];
  for (const child of root.childrenSpans ?? []) {
    if (isRunOutputSpan(child)) continue; // folded into the result node
    work.push(toEntry(child));
  }

  const rootQueued = ms(root.queuedAt) ?? 0;
  const rootEnded = ms(root.endedAt);
  const knownEnd =
    rootEnded ??
    work.reduce<number>(
      (acc, e) => Math.max(acc, e.endedAt ?? e.startedAt ?? e.queuedAt),
      rootQueued
    );

  // Prefer the SDK's own account of what it planned together; fall back to
  // observed execution overlap only when it cannot tell us.
  const planned = toLevelsFromPlans(work);
  const levels = planned ?? toLevels(work, knownEnd);

  const nodes: CanvasNode[] = [];
  const edges: CanvasEdge[] = [];
  const levelNodes: CanvasNode[][] = [];

  const pushNode = (n: CanvasNode) => {
    nodes.push(n);
    (levelNodes[n.level] ??= []).push(n);
  };

  // Level 0 is the trigger: one node, drawn as a stack when a batch produced
  // several events. Deliberately not status-coloured — an event is not a
  // success — and deliberately without event ids, which are chatty and add
  // nothing at this zoom.
  const triggerNode: CanvasNode = {
    id: eventID(0),
    kind: 'event',
    label: trigger?.cron ? `cron ${trigger.cron}` : trigger?.eventName ?? 'Trigger',
    status: 'UNKNOWN',
    level: 0,
    lane: 0,
    spanID: root.spanID,
    stepID: null,
    stepOp: null,
    attempts: 0,
    queuedAt: rootQueued,
    startedAt: ms(root.startedAt),
    endedAt: ms(root.startedAt),
    batchSize: trigger?.IDs?.length ?? 1,
  };
  pushNode(triggerNode);

  let boundaries: CanvasNode[] = [triggerNode];
  let levelIndex = 1;
  let branchesResolved = 0;
  const nodeByEntry = new Map<Entry, CanvasNode>();

  const connect = (from: CanvasNode[], targets: CanvasNode[]) => {
    const fanOut = targets.length > 1;
    const fanIn = from.length > 1;
    for (const source of from) {
      for (const target of targets) {
        edges.push({
          id: `${source.id}->${target.id}`,
          from: source.id,
          to: target.id,
          kind: fanOut ? 'fanOut' : fanIn ? 'fanIn' : 'sequence',
        });
      }
    }
  };

  // Resolve every hop up front. A level needs to know whether the NEXT hop will
  // draw direct edges: if it will, this level must not emit a junction, or the
  // junction is left with edges in and none out — a dead end that reads as if
  // every branch flowed into it.
  const branchesPerLevel: (Branches | null)[] = levels.map((lvl, i) =>
    i === 0 ? null : resolveBranches(levels[i - 1]!, lvl)
  );

  for (const [levelPos, level] of levels.entries()) {
    // When branch membership resolves, put each member in its parent's lane so
    // the graph reads as continuous branches rather than collapsing through a
    // junction on every hop.
    const branches = branchesPerLevel[levelPos] ?? null;

    const laneOfParent = new Map<Entry, number>();
    if (branches) {
      for (const entry of levels[levelPos - 1] ?? []) {
        const node = nodeByEntry.get(entry);
        if (node) laneOfParent.set(entry, node.lane);
      }
    }

    // A join has several parents; lay it out beneath the leftmost of them so
    // 1:1 branches still draw straight and joins sit between what they join.
    const primaryLane = (entry: Entry): number => {
      const parents = branches?.pairs.get(entry);
      if (!parents?.length) return 0;
      return Math.min(...parents.map((p) => laneOfParent.get(p) ?? 0));
    };

    const ordered: Entry[] = branches
      ? [...level].sort((a, b) => primaryLane(a) - primaryLane(b))
      : level;

    const created: CanvasNode[] = ordered.map((entry: Entry, lane: number) => {
      const { trace } = entry;
      const info = trace.stepInfo;
      const kind = nodeKind(entry);
      const node: CanvasNode = {
        id: trace.spanID,
        kind,
        label: trace.name,
        status: toStatus(trace.status),
        waitTimedOut: didTimeOut(entry),
        level: levelIndex,
        lane,
        spanID: trace.spanID,
        stepID: trace.stepID ?? null,
        stepOp: trace.stepOp ?? null,
        attempts: trace.attempts ?? 0,
        userlandStepID: trace.userlandStepID ?? null,
        userlandStepIndex: trace.userlandStepIndex ?? null,
        queuedAt: entry.queuedAt,
        startedAt: entry.startedAt,
        endedAt: entry.endedAt,
      };
      if (kind === 'wait') node.waitKind = waitKindOf(entry.op);
      if (isStepInfoInvoke(info)) node.childRunID = info.runID;
      nodeByEntry.set(entry, node);
      return node;
    });

    created.forEach(pushNode);

    if (branches) {
      // Direct branch-to-branch edges: no junction, because we know which
      // steps each one waited for.
      branchesResolved += 1;
      for (const node of created) {
        const entry = ordered[node.lane]!;
        const parentNodes = (branches.pairs.get(entry) ?? [])
          .map((parent) => nodeByEntry.get(parent))
          .filter((parent): parent is CanvasNode => Boolean(parent));

        if (parentNodes.length) {
          // Laid out beneath the leftmost parent; every parent still gets an
          // edge, so a join reads as a join.
          node.parentNodeID = parentNodes.reduce((a, b) => (a.lane <= b.lane ? a : b)).id;
          node.parentNodeIDs = parentNodes.map((parent) => parent.id);

          for (const parentNode of parentNodes) {
            edges.push({
              id: `${parentNode.id}->${node.id}`,
              from: parentNode.id,
              to: node.id,
              kind: 'sequence',
            });
          }
        }

        // Steps that could have unblocked this one but did not. Broken, and
        // deliberately not part of parentNodeIDs: they are not dependencies.
        for (const alternate of resolveAlternates(levels[levelPos - 1] ?? [], entry)) {
          const alternateNode = nodeByEntry.get(alternate);
          if (!alternateNode) continue;
          edges.push({
            id: `${alternateNode.id}~>${node.id}`,
            from: alternateNode.id,
            to: node.id,
            kind: 'alternate',
          });
        }
      }

      // Steps nothing claimed, which finished in time to have gated this level.
      // Drawn to every step here that has a lineage of its own, so a join the
      // SDK could not observe still reads as a convergence — just an unproven
      // one.
      for (const entry of branches.unconfirmed) {
        const from = nodeByEntry.get(entry);
        if (!from) continue;

        for (const node of created) {
          if (!node.parentNodeIDs?.length) continue;
          edges.push({
            id: `${from.id}?>${node.id}`,
            from: from.id,
            to: node.id,
            kind: 'unconfirmed',
          });
        }
      }
      boundaries = created;
    } else {
      connect(boundaries, created);

      // A level with more than one member gets an explicit join, so the fan-in
      // reads as a junction rather than as a bipartite mesh into the next level.
      // Skipped when the next hop draws direct edges from these nodes, since the
      // junction would then have no outgoing edge.
      const nextDrawsDirectEdges = Boolean(branchesPerLevel[levelPos + 1]);
      if (created.length > 1 && !nextDrawsDirectEdges) {
        levelIndex += 1;
        const join: CanvasNode = {
          id: joinID(levelIndex),
          kind: 'join',
          label: '',
          status: created.some((n) => n.status === 'FAILED') ? 'FAILED' : 'COMPLETED',
          level: levelIndex,
          lane: 0,
          spanID: '',
          stepID: null,
          stepOp: null,
          attempts: 0,
          queuedAt: Math.max(...created.map((n) => n.endedAt ?? n.queuedAt)),
          startedAt: null,
          endedAt: null,
        };
        pushNode(join);
        connect(created, [join]);
        boundaries = [join];
      } else {
        boundaries = created;
      }
    }

    levelIndex += 1;
  }

  // The last level fans back in to the result when it is still several wide.
  if (boundaries.length > 1) {
    const join: CanvasNode = {
      id: joinID(levelIndex),
      kind: 'join',
      label: '',
      status: boundaries.some((n) => n.status === 'FAILED') ? 'FAILED' : 'COMPLETED',
      level: levelIndex,
      lane: 0,
      spanID: '',
      stepID: null,
      stepOp: null,
      attempts: 0,
      queuedAt: Math.max(...boundaries.map((n) => n.endedAt ?? n.queuedAt)),
      startedAt: null,
      endedAt: null,
    };
    pushNode(join);
    connect(boundaries, [join]);
    boundaries = [join];
    levelIndex += 1;
  }

  const runStatus = toStatus(root.status);
  const resultNode: CanvasNode = {
    id: RESULT_ID,
    kind: 'result',
    label:
      runStatus === 'FAILED'
        ? 'Failed'
        : runStatus === 'CANCELLED'
        ? 'Cancelled'
        : runStatus === 'RUNNING' || runStatus === 'QUEUED'
        ? 'Running'
        : 'Completed',
    status: runStatus,
    level: levelIndex,
    lane: 0,
    spanID: root.spanID,
    stepID: null,
    stepOp: null,
    attempts: 0,
    queuedAt: rootQueued,
    startedAt: ms(root.startedAt),
    endedAt: rootEnded,
  };
  pushNode(resultNode);
  connect(boundaries, [resultNode]);

  // `userlandStepIndex` is 0 on unique steps as well as on the first of a
  // collision, so it cannot flag a duplicate by itself. Detect collisions by
  // grouping the run's steps on the ID the user wrote, then number the members.
  const byUserlandID = new Map<string, CanvasNode[]>();
  for (const node of nodes) {
    const key = node.userlandStepID;
    if (!key) continue;
    const bucket = byUserlandID.get(key);
    if (bucket) bucket.push(node);
    else byUserlandID.set(key, [node]);
  }
  for (const bucket of byUserlandID.values()) {
    if (bucket.length < 2) continue;
    for (const node of bucket) {
      node.duplicateIndex = (node.userlandStepIndex ?? 0) + 1;
    }
  }

  // Straighten the graph: pull each node onto the average lane of its children,
  // working right to left. A branch that continues 1:1 then draws as a straight
  // horizontal line, and a node that fans out sits centred on its children.
  // Lanes become fractional here, which the renderer centres per level.
  const childLanes = new Map<string, number[]>();
  for (const node of nodes) {
    if (!node.parentNodeID) continue;
    const bucket = childLanes.get(node.parentNodeID);
    if (bucket) bucket.push(node.lane);
    else childLanes.set(node.parentNodeID, [node.lane]);
  }
  for (let i = levelNodes.length - 1; i >= 0; i--) {
    const level = levelNodes[i] ?? [];

    // Pulling a node onto its children's average is only safe if it does not
    // land on a sibling. `Promise.race` is the case that breaks it: the winner
    // is pulled onto the lane of the step it unblocked, while the loser — which
    // has no children, because an alternate edge is not a parent — stays put,
    // and the two draw on top of each other.
    const desired = new Map<string, number>();
    for (const node of level) {
      const lanes = childLanes.get(node.id);
      desired.set(
        node.id,
        lanes?.length ? lanes.reduce((a, b) => a + b, 0) / lanes.length : node.lane
      );
    }

    // Keep the level's existing order, then walk left to right giving each node
    // its desired lane or one clear of its neighbour, whichever is further
    // right. Order is preserved, nothing overlaps, and a 1:1 chain still
    // straightens because there is no neighbour to be pushed by.
    const ordered = [...level].sort((a, b) => a.lane - b.lane);
    let previous = -Infinity;
    for (const node of ordered) {
      const lane = Math.max(desired.get(node.id) ?? node.lane, previous + 1);
      node.lane = lane;
      previous = lane;
    }
  }

  insertJunctions(nodes, edges);
  attachDiscoveries(nodes, edges, root.discoveries ?? []);

  const hasParallelLevel = levels.some((l) => l.length > 1);
  const parallelismSource: ParallelismSource = planned
    ? 'sdk'
    : hasParallelLevel
    ? 'inferred'
    : 'none';

  if (parallelismSource === 'inferred') {
    warnings.push(
      "Parallel groups are inferred from observed execution overlap. This run's SDK does not report which steps it planned together — inngest-js v4+ does."
    );
  }

  return {
    nodes,
    edges,
    levels: levelNodes.map((l) => (l ?? []).map((n) => n.id)),
    minTime: rootQueued,
    maxTime: rootEnded,
    runStatus,
    parallelismSource,
    branchesResolved,
    warnings,
  };
}
