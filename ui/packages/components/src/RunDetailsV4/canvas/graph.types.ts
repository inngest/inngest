/**
 * The Run Canvas graph model.
 *
 * Deliberately free of React and of rendering concerns: this is the output of a
 * pure function over a run's trace, and it is where the correctness lives.
 * Layout is expressed as (level, lane) — level is the x column, lane is the y
 * row within that column — so the renderer needs no DAG layout pass.
 */

export type CanvasNodeKind =
  /** Synthetic start node: one triggering event. A batch produces several. */
  | 'event'
  /** A step.run / AI gateway step. */
  | 'step'
  /**
   * A sleep / waitForEvent / waitForSignal. These are steps the user writes, so
   * they are nodes, not edge decoration — the canvas should read like the code.
   */
  | 'wait'
  /** A step.invoke, which links out to a child run. */
  | 'invoke'
  /** Synthetic fan-out / fan-in point between two levels. */
  | 'join'
  /**
   * A stand-in for several repeated nodes — a loop body or a wide fan-out drawn
   * once with a count. Never produced by `toCanvasGraph`: it only appears in the
   * rewritten graph `applyCollapse` returns, and the group behind it travels
   * separately. See `collapse.ts`.
   */
  | 'group'
  /** Synthetic end node: the run's result. Selects the run root. */
  | 'result';

export type CanvasStatus =
  | 'QUEUED'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'WAITING'
  | 'SKIPPED'
  | 'UNKNOWN';

export type WaitKind = 'sleep' | 'waitForEvent' | 'waitForSignal';

export type CanvasNode = {
  /** spanID for real nodes; a `canvas:` prefixed synthetic id otherwise. */
  id: string;
  kind: CanvasNodeKind;
  label: string;
  /** Secondary line: event id, wait target, iteration count. */
  sublabel?: string | null;
  status: CanvasStatus;
  /** x position. 0 is the trigger column. */
  level: number;
  /** y position within the level. */
  lane: number;
  /**
   * The span this node selects when clicked, resolved against the same
   * spanID -> Trace map the timeline builds. Synthetic nodes point at the run
   * root so they open the existing run-level panel.
   */
  spanID: string;
  stepID: string | null;
  stepOp: string | null;
  /**
   * The SDK call the user actually wrote, when the platform reports one.
   *
   * Needed because `stepOp` does not always name it: a `step.sendEvent` is
   * reported as an ordinary `RUN`, so a node labelled from `stepOp` alone calls
   * it `step.run` — which is the one thing this view promises not to do.
   */
  stepType?: string | null;
  /** Highest attempt number observed; 0 means it succeeded first try. */
  attempts: number;
  queuedAt: number;
  startedAt: number | null;
  endedAt: number | null;
  /** For invoke nodes: the child run this step started, when one exists. */
  childRunID?: string | null;
  /** For wait nodes. */
  waitKind?: WaitKind;
  /** True when a waitForEvent/waitForSignal expired rather than matched. */
  waitTimedOut?: boolean | null;
  /** For event nodes: how many events this trigger stacked (batch size). */
  batchSize?: number;
  /** The step ID as the user wrote it, before hashing. */
  userlandStepID?: string | null;
  /** The SDK's 0-based collision index. Present even on unique steps. */
  userlandStepIndex?: number | null;
  /**
   * 1-based position among steps sharing a user-written ID, set only when this
   * run really does reuse an ID. Rendered as a suffix so two identically-named
   * steps can be told apart.
   */
  duplicateIndex?: number | null;
  /**
   * The node this one continues, when branch membership resolved. Present only
   * on levels where the pairing was unambiguous.
   *
   * A step that joins several branches has several parents; this is the one it
   * is laid out beneath, and `parentNodeIDs` holds all of them.
   */
  parentNodeID?: string | null;
  /**
   * Every node this one waited for. One entry for a chain, several for a join.
   */
  parentNodeIDs?: string[] | null;
  /**
   * For a junction: the SDK request this convergence or divergence actually
   * was. Present when a discovery could be matched to it.
   */
  discovery?: JunctionDiscovery | null;
  /** For a junction: which way the run changed width, and by how much. */
  junction?: JunctionShape | null;
};

/**
 * Only 'sequence', 'fanOut' and 'fanIn' assert a dependency. The other two are
 * drawn as broken lines, because the target did not necessarily wait for the
 * source:
 *
 *  - 'alternate'   the losing side of a `Promise.race`/`any`. A known
 *                  relationship: it could have unblocked the target, and did
 *                  not. The target would have started regardless.
 *  - 'unconfirmed' the source finished in time to have gated the target, and
 *                  nothing reported that it did. Either a step whose branch
 *                  simply ended, or a join the SDK could not observe — those
 *                  are indistinguishable downstream, so the edge says only that
 *                  the ordering allows it.
 */
/**
 * Which way a junction changed the width of the run.
 *
 *  - 'diverge'  one request planned several steps, which then ran in parallel
 *  - 'converge' several parallel steps finished and the run continued as one.
 *               The executor skips the discovery for every branch but the last
 *               (`shouldEnqueueDiscovery`) and dedupes concurrent finishes on a
 *               shared coalesce key, so N completions become one request.
 *  - 'both'     a set of steps that both gathered and re-fanned out
 */
export type JunctionShape = {
  direction: 'diverge' | 'converge' | 'both';
  /** How many steps fed in. */
  from: number;
  /**
   * How many of those are unproven — they finished in time to have taken part,
   * and nothing reported that they did. Above zero, the junction is still true
   * as an ordering but cannot claim what the engine waited for.
   */
  unproven: number;
  /** How many steps came out. */
  to: number;
};

/** What a junction was, once matched to the request that produced it. */
export type JunctionDiscovery = {
  spanID: string;
  startedAt: number | null;
  endedAt: number | null;
  /** What ran next as a result, by the label the canvas shows them under. */
  next: string[];
};

export type CanvasEdgeKind = 'sequence' | 'fanOut' | 'fanIn' | 'alternate' | 'unconfirmed';

export type CanvasEdge = {
  id: string;
  from: string;
  to: string;
  kind: CanvasEdgeKind;
};

/** The triggering event(s), which the trace payload does not itself carry. */
export type CanvasTrigger = {
  eventName: string | null;
  IDs: string[];
  isBatch: boolean;
  batchID: string | null;
  cron: string | null;
};

/**
 * Where the parallel grouping came from.
 *  - 'sdk'      the SDK told us which steps it planned together (execution v2+)
 *  - 'inferred' guessed from observed execution overlap
 *  - 'none'     the run has no parallelism, so nothing had to be decided
 */
export type ParallelismSource = 'sdk' | 'inferred' | 'none';

export type CanvasGraph = {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  /** Node ids per level, in lane order. This is the layout. */
  levels: string[][];
  minTime: number;
  /** null while the run is still in flight. */
  maxTime: number | null;
  runStatus: CanvasStatus;
  parallelismSource: ParallelismSource;
  /** How many level-to-level hops had their branch membership resolved. */
  branchesResolved: number;
  /**
   * Places where the model is inferred rather than known. Surfaced in the UI
   * so the graph never silently overstates what the trace actually says.
   */
  warnings: string[];
};
