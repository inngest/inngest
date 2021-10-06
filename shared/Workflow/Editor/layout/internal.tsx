import { useMemo } from "react";
import { FlowElement } from "react-flow-renderer";
import {
  nodeW,
  nodeH,
  nodeMarginY,
  nodeMarginX,
  newNode,
  newEdge,
  conditionalHeight,
  conditionalMargin,
  getNodeHeight,
} from "../consts";
import {
  State,
  WorkflowEdge,
  WorkflowAction,
  GraphMutation,
} from "../../state";

// useLayout returns a set of graph nodes and edges given a workflow and a mutation.
//
// This constructs all of the elements that must be placed within the react-flow graph.
const useLayout = (state: State, width: number): FlowElement[] => {
  const w = state.workflow;

  // Merge all mutations together
  const m: GraphMutation[] = (state.dragMutation
    ? [state.dragMutation]
    : []
  ).concat(state.addMutations);

  return useMemo(() => {
    const center = width / 2;

    if (!w) {
      return [];
    }

    // nodes represents all of the nodes rendered into the flow UI.
    // This starts with only the trigger - we'll add to this later.
    //
    // This is the _final_ return type of the algorithm.
    const nodes: Array<FlowElement> = [
      newNode({
        id: "trigger",
        type: "trigger",
        data: { trigger: w.triggers[0] },
        position: { y: nodeMarginY, x: center - nodeW / 2 },
      }),
    ];

    // The general strategy is:
    //
    // 1. Create a list of all edges, from parent -> child
    // 2. BFS through the graph from the trigger down.
    // 3. Add the child edge to `unprocessedActions` as we iterate through edges.
    // 4. Record the number of "rows" we have in our graph, for constraint-based layouts.
    // 5. Iterate through the rows and add nodes with the correct positioning.

    // Add mutation edge to workflow edges so that we iterate once through
    // the outgoingedges map.
    const outgoingedges = new EdgeList(w.edges.concat(m.map((m) => m.edge)));

    // TODO: Iterate through all sinks at the end and place add nodes if the node
    // is not a placeholder, instead of checking mutation length.
    if (w.actions.length === 0 && m.length === 0) {
      // No nodes yet, which means we only have the trigger and add node.
      placeAddNode(nodes, "trigger", center - nodeW / 2, nodeMarginY);
    }

    // Process each edge going from one node to the next, starting with the trigger.
    //
    // We need to iterate through each edge and add the edge's child actions to the map
    // for processing.
    const unprocessedActions: Map<
      string,
      {
        // action may be undefined if this action has yet to be picked, in the case
        // of mutations.
        action: WorkflowAction | undefined;
        depth: number;
        // edgemetadata retains a unique set of edge metadata, ensuring that if there are > 1 edges
        // that lead towards this edge with the same metadata we can group them.
        edgeMetadata: Set<string>;
        incomingedges: WorkflowEdge[];
        mutation: GraphMutation | undefined;
        isSink: boolean;
      }
    > = new Map();

    // rows stores whether each row has an expression, which allows us to generate the y offset
    // for each action in the row at the given depth correctly.
    const rows: Array<{
      expression: boolean;
      count: number;
      actionIDs: string[];
    }> = [];

    const queue: Array<{ clientID: string; depth: number }> = [
      { clientID: "trigger", depth: 0 },
    ];

    // Record whether we've seen this edge before, so that we don't re-render things.
    const traversededges = new Map();

    while (queue.length > 0) {
      const item = queue.shift();
      if (!item) {
        break;
      }

      const childedges = outgoingedges.get(item.clientID);

      // For each element in the queue we're going to add the children's actions to the
      // scene, and calculate the children's edges.
      //
      // Iterate through all edges from the parent to the child to begin processing.
      childedges.forEach((edge) => {
        // If we've already traversed this edge we are going to skip rendering, else we
        // will end up with an infinite queue.
        const edgeKey = `${edge.outgoing}-${edge.incoming}`;
        if (traversededges.has(edgeKey)) {
          console.warn("cycle detected");
          return;
        }
        traversededges.set(edgeKey, true);

        const clientID = edge.incoming.toString();

        // If we have a dragging mutation or we have a `selectedAddNode`, this is
        // true.
        const mutation = m.find((m) => m.edge.incoming.toString() === clientID);
        const action = mutation
          ? mutation.action
          : state.workflowActions[clientID];

        if (!mutation && !action) {
          console.warn("action not found for client ID: ", clientID);
          return;
        }

        // Enqueue this actions' children to process, if there are any.
        const row = item.depth + 1;
        queue.push({ clientID, depth: row });

        if (!rows[row]) {
          // We don't push to the actionIDs yet because we may update an action's depth
          // if it has incoming edges from a later node.
          rows[row] = { count: 0, expression: false, actionIDs: [] };
        }
        rows[row].count++;

        // Now we need to render this "row's" edges.  A row's edge may have an icon if
        // any one of the actions from this node has an expression or metadata.
        const isBlank =
          !edge.metadata ||
          !edge.metadata.type ||
          (edge.metadata.type === "edge" && edge.metadata.if === "");
        if (!isBlank) {
          // indicate that this row has expressions, so that we render the actions in
          // this row at the same Y coordinate.
          rows[row].expression = true;
        }

        // An action can have > 1 incoming edge, and we want to ensure it's rendered as
        // deep in the graph as possible so that we can have orderly rows in the UI.
        const actionItem = unprocessedActions.get(edge.incoming.toString()) || {
          action: action,
          depth: row,
          edgeMetadata: new Set(),
          incomingedges: [] as WorkflowEdge[],
          mutation,
          // isSink is true if there are no outgoing edges from this action.  This lets us
          // determine whether to show the "Add" icon.
          isSink:
            !action ||
            outgoingedges.get(action.clientID.toString()).length === 0,
        };

        if (actionItem.depth <= row) {
          actionItem.depth = row;
        }

        // Store a unique set of edge metadata for each action.
        !isBlank && actionItem.edgeMetadata.add(JSON.stringify(edge.metadata));
        actionItem.incomingedges.push(edge);

        unprocessedActions.set(clientID, actionItem);
      });
    }

    unprocessedActions.forEach((item, clientID) => {
      rows[item.depth].actionIDs.push(clientID);
    });

    // Iterate through and render each row.  We start a at given offset after the target,
    // and we increase the offset after rendering each row.  An offset isn't increased by
    // a given number each row:  we may render conditionals and actions in a single taller
    // row.
    let yOffset = nodeH + nodeMarginY * 2;

    rows.forEach((row) => {
      const siblings = row.actionIDs.length - 1;
      const actionYOffset = row.expression
        ? yOffset + conditionalHeight + conditionalMargin
        : yOffset;

      row.actionIDs.forEach((id, n) => {
        const block = unprocessedActions.get(id);

        if (!block) {
          // This is a placeholder action.
          console.warn("unknown action for id: ", id);
          return;
        }

        // TODO: Subgroups of these - if we have a dag where an action that occurs after
        // this action loops back, we want to render the incoming edge to the left or right of
        // the action block.

        const rowWidth = (siblings + 1) * nodeW + siblings * nodeMarginX;
        const xOffset = n * nodeW + n * nodeMarginX + (center - rowWidth / 2);

        // What type of node should we render?  We render different nodes for
        // saved actions vs placeholders.
        const type = block.action === undefined ? "placeholder" : "action";

        block.incomingedges.forEach((e) => {
          if (row.expression) {
            // TODO:  Render icon above.

            const isBlank =
              !e.metadata ||
              (e.metadata.type === "edge" && e.metadata.if === "");

            nodes.push(
              newNode({
                id: e.outgoing + "-expression-" + e.incoming,
                // blankExpression is used for a custom css rule in the global CSS to avoid pointer
                // events when nothing is rendered.
                //
                // We have to render this element ot get react-flow's edge visuals to look correct.
                type: !isBlank ? "expression" : "blankExpression",
                data: {
                  edge: e,
                  action: block,
                },
                position: {
                  y: yOffset,
                  x: xOffset,
                },
              })
            );

            // Add the edge from the parent node to the condition
            nodes.push(
              newEdge({
                source: e.outgoing.toString(),
                target: e.outgoing + "-expression-" + e.incoming,
              })
            );
            nodes.push(
              newEdge({
                source: e.outgoing + "-expression-" + e.incoming,
                target: e.incoming.toString(),
              })
            );
            return;
          }

          block.incomingedges.forEach((e) => {
            nodes.push(
              newEdge({
                source: e.outgoing.toString(),
                target: e.incoming.toString(),
              })
            );
          });
        });

        nodes.push(
          newNode({
            id,
            type,
            data: {
              action: block.action,
              mutation: block.mutation,
            },
            position: {
              x: xOffset,
              y: actionYOffset,
            },
          })
        );

        // If this is a sink, push a new node and edge.
        if (block.isSink && !block.mutation) {
          placeAddNode(nodes, id, xOffset, actionYOffset);
        }
      });

      yOffset = actionYOffset + nodeH + nodeMarginY;
    });

    return nodes;
    // eslint-disable-next-line
  }, [w, m, width]);
};

// placeAddNodes pushes an add node and the edge from the parent -> add node
// to the nodes array.
const placeAddNode = (
  nodes: Array<FlowElement>,
  parentID: any,
  xOffset: number,
  yOffset: number
) => {
  // This is a no-op in the website.
  return;

  nodes.push(
    newNode({
      id: "add-" + parentID,
      type: "add",
      data: {
        outgoingID: parentID,
        edgeMetadata: undefined,
      },
      position: {
        x: xOffset, // TODO
        y: yOffset + nodeMarginY + getNodeHeight("add") / 2,
      },
    })
  );

  nodes.push(
    newEdge({
      source: parentID,
      target: "add-" + parentID,
    })
  );
};

// EdgeList stores all edges from parent -> child for quick lookup and traversal
// of the graph.
class EdgeList {
  edges: { [clientID: string]: Array<WorkflowEdge> } = {};

  constructor(list: Array<WorkflowEdge>) {
    this.edges = {};
    list.forEach((e) => this.push(e));
  }

  push(e: WorkflowEdge) {
    if (!Array.isArray(this.edges[e.outgoing])) {
      this.edges[e.outgoing] = [];
    }
    this.edges[e.outgoing].push(e);
  }

  get(clientID: string): Array<WorkflowEdge> {
    return this.edges[clientID] || [];
  }
}

export default useLayout;
