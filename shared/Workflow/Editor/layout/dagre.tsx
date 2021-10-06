import { useMemo } from "react";
import dagre from "dagre";
import { FlowElement, Edge, Node } from "react-flow-renderer";
import { addW, nodeW, getNodeHeight, gridSize, nodeMarginX } from "../consts";

type LayoutArgs = {
  width: number;
  elements: FlowElement[];
};

const positioner = (g: any) => {
  const iterate = (fn: (key: string, obj: any) => void) => {
    Object.keys(g._nodes).forEach((key) => {
      fn(key, g._nodes[key]);
    });
  };

  const shift = (rank: number, y: number) => {
    iterate((_key, node) => {
      if (node.rank >= rank) {
        node.y += y;
      }
    });
  };

  iterate((key, node) => {
    // Move all of the expression nodes 15 pixels closer to the child action
    if (key.indexOf("expression") > -1) {
      // Everything in this row should be shifted up.
      shift(node.rank, -15);
      node.y += 35;
    }
    if (key.indexOf("add") > -1) {
      node.y -= 38;
    }
  });
};

// useLayout organizes nodes within our flow diagram to a tree.
const useLayout = (args: LayoutArgs): FlowElement[] => {
  // 652 is the width of menu and right panel.  It's used if the page renders with no
  // mouse, as we use the elW of useMouse for the width of the container.
  const width = args.width || (globalThis.window ? globalThis.window.innerWidth : 652) - 652;

  return useMemo(() => {
    const g = new dagre.graphlib.Graph();

    g.setGraph({
      nodesep: nodeMarginX,
      ranksep: 50,
      edgesep: 0,
      positioner: positioner,
      freezeOrder: true,
    } as any);

    g.setDefaultEdgeLabel(function () {
      return {};
    });

    // add all of our elements to the dagre graph so that dagre can compute an ideal layout
    args.elements.forEach((e: any) => {
      if (e.source) {
        g.setEdge(e.source, e.target);
        return;
      }

      // The "Add" box shown beneath each action has a different height and width - it's smaller
      // than the standard boxes.
      const width = e.type === "add" ? addW : nodeW;
      const height = getNodeHeight(e.type);

      g.setNode(e.id, {
        width,
        height,
        // store the original flow element for reconstruction
        element: e,
      });
    });

    dagre.layout(g);

    // dagre gives us "relative" positioning based off of the bounding box of the graph.
    // We want to position each node in the center of the canvas;  so, take the bounding
    // box and figure out the correct padding.
    const centerPadding = Math.max((width - (g.graph().width || 0)) / 2, 0);
    const triggerPadding = gridSize * 2;

    const nodes = g.nodes().map((id: string) => {
      let n = g.node(id);
      if (!n) {
        console.warn("unknown node");
        return undefined;
      }
      const element = (n as any).element as Node;

      return {
        ...element,
        width: n.width,
        height: n.height,
        position: {
          // the 'x' from dagre's node represents the center of each node.  we need to
          // enusre we account for the node's width and height when positioning.
          x: n.x - n.width / 2 + centerPadding,
          y: n.y - n.height / 2 + triggerPadding + 20, // shift 20 pixels down
        },
      };
    });

    // We don't actually need dagre edges;  we draw them using the flow dependency itself.
    const edges = args.elements.filter((e) => !!(e as Edge).source) as Edge[];
    return (nodes as FlowElement[]).concat(edges as Edge[]);
  }, [args.elements, width]);
};

export default useLayout;
