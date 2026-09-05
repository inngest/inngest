/**
 * Maps the pure graph model onto React Flow's node/edge shape.
 *
 * Layout is a straight read of (level, lane) — no DAG layout pass, because the
 * derivation already levelled the graph.
 */
import { MarkerType, type Edge, type Node } from '@xyflow/react';

import { formatDuration } from '../runDetailsUtils';
import type { CanvasGroup } from './collapse';
import type { CanvasEdge, CanvasGraph, CanvasNode } from './graph.types';

export const LAYOUT = {
  /** Horizontal distance between level centres. */
  columnGap: 252,
  /**
   * Extra width for a gap that holds a junction.
   *
   * The circle sits in the middle of the gap between two node columns and
   * halves it, leaving ~22px either side. The stepped router runs straight out
   * of each end before turning and needs that run to be under half the span, so
   * in 22px it can only turn after 11px — a ~5px corner, against 10px on every
   * other hop, which is why a fan-out read as hard right angles. Widening only
   * the gaps that contain a junction buys the room to route them exactly like
   * everything else, and leaves the rest of the graph at its usual pitch.
   */
  junctionGap: 60,
  /** Vertical distance between lanes within a level. */
  laneGap: 80,
  nodeWidth: 186,
  nodeHeight: 52,
  /** A group node carries a count, a variant line and a sparkline. */
  groupHeight: 72,
  joinWidth: 22,
  /**
   * An invoke node with the run it started opened inside it.
   *
   * Sized to the child's depth between these bounds, so a two-step child does
   * not sit in a mostly-empty box and a deep one scrolls inside its own region
   * rather than pushing the graph around without limit. The child is a
   * different run and stays inside a border that says so.
   */
  invokeOpenMinHeight: 150,
  invokeOpenMaxHeight: 260,
  /**
   * How wide an open invoke gets.
   *
   * A summary fitted in 186px; the child's actual graph does not. The level
   * holding an open invoke is widened by the difference so the child has room
   * to be a graph rather than a list, and nothing downstream is drawn over.
   */
  invokeOpenWidth: 460,
  /** Header, border and padding around the child's levels. */
  invokeChrome: 44,
  /** One level of the child's graph. */
  invokeLevelHeight: 17,
} as const;

export type CanvasNodeData = CanvasNode & {
  /** Set while the scrubber is behind this node's start (Phase 3). */
  future?: boolean;
  /** True while the pointer is over this step in any view. */
  hovered?: boolean;
  /** Present on `kind: 'group'` nodes: the repetition this one stands in for. */
  group?: CanvasGroup;
  /** Opens this group, so its members are drawn individually. */
  onExpandGroup?: (groupID: string) => void;
  /** True when this invoke is showing the run it started, inside itself. */
  childOpen?: boolean;
  /** The child run's own graph, once fetched. `null` means it could not load. */
  childGraph?: CanvasGraph | null;
  /** True between the disclosure being opened and the child arriving. */
  childLoading?: boolean;
  /** Opens or closes the child run inside this node. */
  onToggleChild?: (nodeID: string) => void;
  /** Leaves this run for the child's own page — the escape, not the only path. */
  onOpenChildRun?: (childRunID: string) => void;
  [key: string]: unknown;
};

/**
 * How tall an invoke grows to hold the run it started.
 *
 * Exported because the node renders itself at this height and the layout
 * reserves the room for it before that happens. Two independent guesses would
 * drift and the child would either be clipped or float in an empty box.
 */
export function invokeOpenHeight(childLevels: number | undefined): number {
  const levels = childLevels ?? 1;
  return Math.max(
    LAYOUT.invokeOpenMinHeight,
    Math.min(
      LAYOUT.invokeOpenMaxHeight,
      LAYOUT.nodeHeight + LAYOUT.invokeChrome + levels * LAYOUT.invokeLevelHeight
    )
  );
}

/**
 * Levels of the child that actually draw a row.
 *
 * The preview shows only the child's work — its Trigger and terminal node are
 * structure, and drawing them put a second `Trigger` and a second `Completed`
 * into a picture that already had one of each. Sizing on `levels.length` then
 * reserved room for rows that are never drawn, leaving the box mostly empty.
 */
export function childPreviewRows(child: CanvasGraph): number {
  const byID = new Map(child.nodes.map((n) => [n.id, n]));
  return child.levels.filter((ids) =>
    ids.some((id) => {
      const n = byID.get(id);
      return n?.kind === 'step' || n?.kind === 'wait' || n?.kind === 'invoke';
    })
  ).length;
}

function heightOf(
  node: CanvasNode,
  openChildren?: ReadonlySet<string>,
  childGraphs?: ReadonlyMap<string, CanvasGraph | null>
): number {
  if (node.kind === 'join') return LAYOUT.joinWidth;
  if (node.kind === 'group') return LAYOUT.groupHeight;

  // An open invoke holds another run inside it, so it needs the room BEFORE
  // layout runs — otherwise the child draws over whatever is in the lane below.
  // The child's depth is already known here, so the box is sized to it rather
  // than to a guess, between bounds that keep the reflow predictable.
  if (openChildren?.has(node.id)) {
    const child = node.childRunID ? childGraphs?.get(node.childRunID) : undefined;
    return invokeOpenHeight(child ? childPreviewRows(child) : undefined);
  }

  return LAYOUT.nodeHeight;
}

function position(
  node: CanvasNode,
  laneMid: number,
  xAt: (level: number) => number,
  openChildren?: ReadonlySet<string>,
  childGraphs?: ReadonlyMap<string, CanvasGraph | null>
) {
  // Centre each level on the middle of its own lane range. Lanes are fractional
  // once nodes have been pulled onto their children's average lane, so this
  // cannot be derived from a simple count.
  const offset = laneMid * LAYOUT.laneGap;
  const width =
    node.kind === 'join'
      ? LAYOUT.joinWidth
      : openChildren?.has(node.id)
      ? LAYOUT.invokeOpenWidth
      : LAYOUT.nodeWidth;
  const height = heightOf(node, openChildren, childGraphs);
  return {
    // Centre nodes of differing widths/heights on the level axis, so a join
    // circle sits on the same centre line as the wide nodes either side of it.
    x: xAt(node.level) + (LAYOUT.nodeWidth - width) / 2,
    y: node.lane * LAYOUT.laneGap - offset + (LAYOUT.nodeHeight - height) / 2,
  };
}

export function toFlowElements(
  graph: CanvasGraph,
  selectedSpanID?: string,
  collapse?: {
    groupByNodeID: ReadonlyMap<string, CanvasGroup>;
    onExpandGroup: (groupID: string) => void;
  },
  /** Span hovered anywhere in this run — here, or in a row below. */
  hoveredSpanID?: string,
  /** Invoke nodes currently showing the run they started, inside themselves. */
  children?: {
    open: ReadonlySet<string>;
    graphs: ReadonlyMap<string, CanvasGraph | null>;
    loading: ReadonlySet<string>;
    onToggle: (nodeID: string) => void;
    onOpenRun?: (childRunID: string) => void;
  }
): { nodes: Node<CanvasNodeData>[]; edges: Edge[] } {
  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  const laneMids = graph.levels.map((ids) => {
    const lanes = ids.map((id) => byID.get(id)?.lane ?? 0);
    return lanes.length ? (Math.min(...lanes) + Math.max(...lanes)) / 2 : 0;
  });
  const placed = new Map<string, { x: number; y: number }>();

  // Levels are evenly pitched except where a junction sits between two of them.
  const junctionLevels = graph.nodes.filter((n) => n.kind === 'join').map((n) => n.level);
  // A level holding an open invoke needs the extra width that node takes.
  const openLevels = new Set(
    graph.nodes.filter((n) => children?.open.has(n.id)).map((n) => n.level)
  );
  const levelX: number[] = [];
  let cursor = 0;
  for (let level = 0; level < graph.levels.length; level++) {
    levelX[level] = cursor;
    const holdsJunction = junctionLevels.some((l) => l > level && l < level + 1);
    cursor +=
      LAYOUT.columnGap +
      (holdsJunction ? LAYOUT.junctionGap : 0) +
      (openLevels.has(level) ? LAYOUT.invokeOpenWidth - LAYOUT.nodeWidth : 0);
  }

  /** x of any level, including the half-levels junctions sit on. */
  const xAt = (level: number) => {
    const lo = Math.floor(level);
    const hi = Math.ceil(level);
    const a = levelX[lo] ?? lo * LAYOUT.columnGap;
    if (lo === hi) return a;
    const b = levelX[hi] ?? a + LAYOUT.columnGap;
    return a + (b - a) * (level - lo);
  };

  // Steps first. A junction is placed against the nodes it actually connects,
  // which cannot be worked out from lanes alone: its neighbours sit on two
  // different levels, and each level is centred on its own lane range, so the
  // same lane number means a different height on either side. Deriving it from
  // lanes put the circle off the line the edges converge on, which is what made
  // the routing wander.
  const steps = graph.nodes.filter((node) => node.kind !== 'join');
  const junctions = graph.nodes.filter((node) => node.kind === 'join');

  const nodes: Node<CanvasNodeData>[] = steps.map((node) => {
    const pos = position(node, laneMids[node.level] ?? 0, xAt, children?.open, children?.graphs);
    placed.set(node.id, pos);
    const group = collapse?.groupByNodeID.get(node.id);
    const childRunID = node.childRunID ?? undefined;
    return {
      id: node.id,
      type: node.kind === 'group' ? 'canvasGroup' : 'canvasStep',
      position: pos,
      data: {
        ...node,
        hovered: hoveredSpanID !== undefined && node.spanID === hoveredSpanID,
        ...(group ? { group, onExpandGroup: collapse?.onExpandGroup } : {}),
        ...(children && childRunID
          ? {
              childOpen: children.open.has(node.id),
              childGraph: children.graphs.get(childRunID),
              childLoading: children.loading.has(childRunID),
              onToggleChild: children.onToggle,
              onOpenChildRun: children.onOpenRun,
            }
          : {}),
      },
      draggable: false,
      connectable: false,
      selectable: true,
      // Selection is driven by the shared step-selection emitter, not by React
      // Flow's own click state, so that selecting a row in the trace highlights
      // the matching node here too.
      selected: selectedSpanID !== undefined && node.spanID === selectedSpanID,
      // Declared up front so the initial fitView has real dimensions to work
      // with; without these it fits against zero-sized nodes and overflows.
      width: children?.open.has(node.id) ? LAYOUT.invokeOpenWidth : LAYOUT.nodeWidth,
      height: heightOf(node, children?.open, children?.graphs),
    };
  });

  // Centre of a placed step, since the two shapes have different heights.
  const centreOf = (id: string) => {
    const pos = placed.get(id);
    return pos ? pos.y + LAYOUT.nodeHeight / 2 : null;
  };

  for (const node of junctions) {
    const neighbours = graph.edges
      .filter((edge) => edge.from === node.id || edge.to === node.id)
      .map((edge) => centreOf(edge.from === node.id ? edge.to : edge.from))
      .filter((y): y is number => y !== null);

    const centre = neighbours.length
      ? neighbours.reduce((a, b) => a + b, 0) / neighbours.length
      : LAYOUT.nodeHeight / 2;

    const pos = {
      x: xAt(node.level) + (LAYOUT.nodeWidth - LAYOUT.joinWidth) / 2,
      y: centre - LAYOUT.joinWidth / 2,
    };
    placed.set(node.id, pos);

    nodes.push({
      id: node.id,
      type: 'canvasJoin',
      position: pos,
      data: { ...node },
      draggable: false,
      connectable: false,
      selectable: true,
      selected:
        selectedSpanID !== undefined && node.spanID !== '' && node.spanID === selectedSpanID,
      width: LAYOUT.joinWidth,
      height: LAYOUT.joinWidth,
    });
  }

  /**
   * What happened between two steps, and how long it took.
   *
   * A node says how long its own step ran. The time BETWEEN steps — discovery,
   * queueing, system latency, a retry backoff — used to be nowhere on the
   * canvas at all, which left the graph unable to explain its own gaps while
   * the trace below showed them as blank space. It belongs on the edge: the
   * edge is literally the interval between one step ending and the next
   * starting.
   *
   * Only labelled when both ends are real work with real times, and when the
   * gap is big enough to be worth a reader's attention — every hop has a
   * millisecond or two of overhead and labelling all of them would be noise.
   */
  const MIN_LABELLED_GAP_MS = 20;
  const gapLabel = (edge: CanvasEdge): string | undefined => {
    const source = byID.get(edge.from);
    const target = byID.get(edge.to);
    if (!source || !target) return undefined;
    if (source.endedAt === null || target.startedAt === null) return undefined;

    // Synthetic nodes borrow the run's timings, so a gap measured against them
    // is not a gap between steps.
    const real = (n: CanvasNode) => n.kind === 'step' || n.kind === 'wait' || n.kind === 'invoke';
    if (!real(source) || !real(target)) return undefined;

    const gap = target.startedAt - source.endedAt;
    return gap >= MIN_LABELLED_GAP_MS ? formatDuration(gap) : undefined;
  };

  const edges: Edge[] = graph.edges.map((edge) => {
    const from = placed.get(edge.from);
    const to = placed.get(edge.to);
    // Straight lines whenever the two ends share a centre line; the stepped
    // router is only needed where the graph actually changes lane.
    const sameLane = from && to && Math.abs(from.y - to.y) < 1;

    // Neither of these asserts a dependency, so both are broken lines: the
    // race loser could have unblocked the target and did not, and the
    // unconfirmed one merely finished in time to have done so. The race is
    // dashed because the relationship is known; the unconfirmed one is dotted
    // and fainter because it is not.
    const isAlternate = edge.kind === 'alternate';
    const isUnconfirmed = edge.kind === 'unconfirmed';

    const label = gapLabel(edge);

    return {
      id: edge.id,
      source: edge.from,
      target: edge.to,
      // An arrow, so the direction of the run is readable without inferring it
      // from left-to-right convention.
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
      label,
      labelShowBg: Boolean(label),
      labelBgPadding: [3, 1] as [number, number],
      labelBgBorderRadius: 2,
      type: sameLane ? 'straight' : 'smoothstep',
      pathOptions: sameLane ? undefined : { borderRadius: 24 },
      // Both broken kinds are drawn a little heavier than a solid edge, not
      // lighter: they carry the least obvious claim on the canvas and are the
      // easiest to lose against the dot grid. The dash lengths, not the weight,
      // are what separate them — long dashes for a race, short for unconfirmed.
      //
      // Solid edges are drawn confidently rather than as hairlines. On a flow
      // chart the edges ARE the content — they are the only thing saying what
      // led to what — and at 1px against a dot grid they were the faintest
      // thing on a canvas whose whole purpose they carry.
      style: isAlternate
        ? { strokeDasharray: '7 4', strokeWidth: 1.5, opacity: 0.85 }
        : isUnconfirmed
        ? { strokeDasharray: '2 3', strokeWidth: 1.5, opacity: 0.8 }
        : { strokeWidth: 1.5 },
      data: { canvas: edge },
    };
  });

  return { nodes, edges };
}
