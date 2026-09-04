/**
 * Maps the pure graph model onto React Flow's node/edge shape.
 *
 * Layout is a straight read of (level, lane) — no DAG layout pass, because the
 * derivation already levelled the graph.
 */
import type { Edge, Node } from '@xyflow/react';

import type { CanvasGraph, CanvasNode } from './graph.types';

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
  joinWidth: 22,
} as const;

export type CanvasNodeData = CanvasNode & {
  /** Set while the scrubber is behind this node's start (Phase 3). */
  future?: boolean;
  [key: string]: unknown;
};

function position(node: CanvasNode, laneMid: number, xAt: (level: number) => number) {
  // Centre each level on the middle of its own lane range. Lanes are fractional
  // once nodes have been pulled onto their children's average lane, so this
  // cannot be derived from a simple count.
  const offset = laneMid * LAYOUT.laneGap;
  const width = node.kind === 'join' ? LAYOUT.joinWidth : LAYOUT.nodeWidth;
  const height = node.kind === 'join' ? LAYOUT.joinWidth : LAYOUT.nodeHeight;
  return {
    // Centre nodes of differing widths/heights on the level axis, so a join
    // circle sits on the same centre line as the wide nodes either side of it.
    x: xAt(node.level) + (LAYOUT.nodeWidth - width) / 2,
    y: node.lane * LAYOUT.laneGap - offset + (LAYOUT.nodeHeight - height) / 2,
  };
}

export function toFlowElements(
  graph: CanvasGraph,
  selectedSpanID?: string
): { nodes: Node<CanvasNodeData>[]; edges: Edge[] } {
  const byID = new Map(graph.nodes.map((n) => [n.id, n]));
  const laneMids = graph.levels.map((ids) => {
    const lanes = ids.map((id) => byID.get(id)?.lane ?? 0);
    return lanes.length ? (Math.min(...lanes) + Math.max(...lanes)) / 2 : 0;
  });
  const placed = new Map<string, { x: number; y: number }>();

  // Levels are evenly pitched except where a junction sits between two of them.
  const junctionLevels = graph.nodes.filter((n) => n.kind === 'join').map((n) => n.level);
  const levelX: number[] = [];
  let cursor = 0;
  for (let level = 0; level < graph.levels.length; level++) {
    levelX[level] = cursor;
    const holdsJunction = junctionLevels.some((l) => l > level && l < level + 1);
    cursor += LAYOUT.columnGap + (holdsJunction ? LAYOUT.junctionGap : 0);
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
    const pos = position(node, laneMids[node.level] ?? 0, xAt);
    placed.set(node.id, pos);
    return {
      id: node.id,
      type: 'canvasStep',
      position: pos,
      data: { ...node },
      draggable: false,
      connectable: false,
      selectable: true,
      // Selection is driven by the shared step-selection emitter, not by React
      // Flow's own click state, so that selecting a row in the trace highlights
      // the matching node here too.
      selected: selectedSpanID !== undefined && node.spanID === selectedSpanID,
      // Declared up front so the initial fitView has real dimensions to work
      // with; without these it fits against zero-sized nodes and overflows.
      width: LAYOUT.nodeWidth,
      height: LAYOUT.nodeHeight,
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

    return {
      id: edge.id,
      source: edge.from,
      target: edge.to,
      type: sameLane ? 'straight' : 'smoothstep',
      pathOptions: sameLane ? undefined : { borderRadius: 24 },
      // Both broken kinds are drawn a little heavier than a solid edge, not
      // lighter: they carry the least obvious claim on the canvas and are the
      // easiest to lose against the dot grid. The dash lengths, not the weight,
      // are what separate them — long dashes for a race, short for unconfirmed.
      style: isAlternate
        ? { strokeDasharray: '7 4', strokeWidth: 1.5, opacity: 0.85 }
        : isUnconfirmed
        ? { strokeDasharray: '2 3', strokeWidth: 1.5, opacity: 0.8 }
        : undefined,
      data: { canvas: edge },
    };
  });

  return { nodes, edges };
}
