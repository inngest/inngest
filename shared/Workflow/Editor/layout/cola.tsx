import { useMemo } from "react";
import { FlowElement } from "react-flow-renderer";
import { Graph } from "setcola";
import { nodeW, getNodeHeight } from "../consts";
import { State, WorkflowEdge, GraphMutation } from "../../state";

const useLayout = (state: State, m: GraphMutation | null): FlowElement[] => {
  const w = state.workflow;

  return useMemo(() => {
    if (!w) {
      return [];
    }

    // Iterate through each action and build a mapping of action IDs to outgoing edges, which allows
    // us to iterate through eachc child of an action.
    const outgoingedges: { [clientID: string]: Array<WorkflowEdge> } = {};
    w.edges.concat(m ? [m.edge] : []).forEach((e) => {
      if (!Array.isArray(outgoingedges[e.outgoing])) {
        outgoingedges[e.outgoing] = [];
      }
      outgoingedges[e.outgoing].push(e);
    });

    // itereate through things and add nodes, links, then figure out some other shit.
    const graph: Graph = {
      nodes: [],
      links: [],
    };

    // Each action is placed in a group with its child.
    const queue = w.triggers.map((item) => ({
      id: "trigger",
      type: "trigger",
      item,
      depth: 0,
    }));

    while (queue.length > 0) {
      const item = queue.shift();
      if (!item) {
        break;
      }

      graph.nodes.push({
        id: item.id,
        width: nodeW,
        height: getNodeHeight(item.type),
      });

      // TODO: Group nodes, add parent conditionals, add constraints.
    }

    return [];
  }, [w, m]);
};

export default useLayout;
