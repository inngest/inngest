import React, { useRef, useCallback } from "react";
import { useMouse } from "react-use";
import throttle from "lodash.throttle";
import ReactFlow, {
  ReactFlowProvider,
  useStoreState,
  FlowElement,
  Edge,
  Node,
  Background,
  ConnectionLineType,
} from "react-flow-renderer";
import { nodeW, getNodeHeight, gridSize, newClientID } from "./consts";
import { StateProps, GraphMutation } from "../state";
import NodeTrigger from "./Nodes/Trigger";
import NodeAction from "./Nodes/Action";
import NodePlaceholder from "./Nodes/Placeholder";
import NodeExpression from "./Nodes/Conditional";
import NodeConditionalAction from "./Nodes/ConditionalAction";
import NodeAdd from "./Nodes/Add";
import NodeEdgeIcon from "./Nodes/EdgeIcon";
import useElements from "./layout/internal";
import useLayout from "./layout/dagre";

const nodeTypes = {
  trigger: NodeTrigger,

  // action represents an action that has no conditions
  action: NodeAction,
  placeholder: NodePlaceholder,
  conditionalAction: NodeConditionalAction,
  expression: NodeExpression,
  blankExpression: NodeExpression,

  edgeIcon: NodeEdgeIcon,

  // add represents the "+" add node.
  add: NodeAdd,
};

// EditorWrapper wraps the editor with the flow provider, allowing us access to the state
// of the flow diagram.  This is needed for us to detect zoom levels.
const EditorWrapper: React.FC<StateProps> = (props) => (
  <ReactFlowProvider>
    <Editor {...props} />
  </ReactFlowProvider>
);

export default EditorWrapper;

// Editor represents the visual editor used to edit workflows.  This handles complex state:
//
// 1: Rendering triggers and actions as a flow diagram
// 2: Handling dragging and dropping, editing the flow diagram with "unsaved" state
// 3: Hovering over triggers and actions to show context
// 4: Managing action-sapecific metadata.
export const Editor: React.FC<StateProps> = (props) => {
  const { state, dispatch } = props;

  const ref = useRef(null);
  const { posX, posY, elW } = useMouse(ref);

  // Get the X/Y offset (positive => things shifted to right/down) and zoom.

  const [flowX, flowY, flowScale] = useStoreState(
    (state: any) => state.transform
  );

  // mutations records a "preview" of a workflow mutation, caused by someone dragging
  // an action over the flow canvas.
  //
  // When dragging, we want to show a preview of where the action sits based off of where you're
  // dropping the element.
  const mutation = state.dragMutation;
  const setMutation = (m: GraphMutation | null) => {
    if (state.dragMutation === m) {
      return;
    }
    dispatch({ type: "dragMutation", mutation: m });
  };

  // setMutationThrottled only updates the mutation once every 100ms to prevent thrashing when moving
  // the dragged action over graphs.
  //
  // Dagre redraws the layout on potentially any change;  if the dragged action is in a position that
  // causes the graph to redraw this may result in a loop and the inability to place a node in any
  // particular position (as you drag over other nodes).
  const setMutationThrottled = useCallback(
    throttle((m: GraphMutation | null) => setMutation(m), 100),
    []
  );

  const elements = useElements(state, elW);

  // const layout = elements; // useLayout({ width: elW, elements });
  const layout = useLayout({ width: elW, elements });

  const onDragOver = (e: React.DragEvent) => {
    // stopPropagation and preventDefault are needed for onDrop to work within react.
    e.stopPropagation();
    e.preventDefault();

    const action = state.dragAction;
    if (!action) {
      return;
    }

    // We want to calculate which node we're dragging an action onto based off of the layout
    // of the current vertexes.
    //
    // To do that, we need the coordinates of the cursor in the local space of the canvas
    //  - including any scroll offsets, if visible, and the "zoom" factor of the flow diagram.
    let localX = e.clientX + window.scrollX - posX - flowX;
    let localY = e.clientY + window.scrollY - posY - flowY;

    // Now that we have the "local X/Y", we need to factor the offsets and zooms from panning around
    // the flow canvas.
    localX = localX / flowScale;
    localY = localY / flowScale;

    // We break the flow diagram up into "rows".  The targets are on row 0;  dropping a node onto
    // row 1 means "adding a child to the target" - ie. an edge from the target to the new node.
    //
    // We only want the "deepest" row, so record the max Y we see and filter out other els where
    // y < max.
    const nodes: Node[] = layout.filter(
      (e: FlowElement) => !(e as Edge).source // this removes edges.
    ) as Node[];

    let candidates: Node[] = nodes
      .filter((e) => e.data.mutation !== true) // you can't child the mutation
      .filter((e) => e.position.y + getNodeHeight(e.type || "") < localY)
      .sort((a, b) => b.position.y - a.position.y);

    // Find deepest row
    let max = candidates[0] ? candidates[0].position.y : 0;
    candidates = candidates.filter((a) => a.position.y === max);

    if (candidates.length === 0) {
      setMutationThrottled(null);
      return;
    }

    let candidate = candidates[0];
    if (candidates.length > 1) {
      // Find likely candidate based off of the X position in columns.
      // XXX: This can be improved.
      const mapped = candidates.map((a) => ({
        node: a,
        distance: Math.abs(a.position.x + nodeW / 2 - localX),
      }));
      candidate = mapped.sort((a, b) => a.distance - b.distance)[0].node;
    }

    // If the candidate is a trigger, then the action is going to be a "source" - it is one of
    // the first actions.  We represent mutations in the same format as our workflow Cue.
    const source =
      candidate.type === "trigger" || candidate.data.outgoingID === "trigger";
    const clientID = newClientID(state);

    // If the candidate is an "Add" box we want to copy its indegree edge metadata.  The
    // "Add" box is added if there are default branches (eg. true/false);  we want to ensure
    // this branch metadata is inherited.
    //
    // This can also be dropped as siblings to the add box.  If so, find the "add" box to the
    // left of the hoverable.
    let edgeMetadata;
    if (candidate.type === "add" && state.workflow) {
      edgeMetadata = candidate.data.edgeMetadata || undefined;
    }

    const edge = (() => {
      switch (true) {
        case source:
          return { outgoing: "trigger", incoming: clientID, metadata: {} };
        case candidate.type === "placeholder":
          return { ...candidate.data.mutation.edge, metadata: {} };
        default:
          return {
            // The candidate may be an action or an add node.  If it's an add ndoe,
            // the outgoing ID will be stored in data.outgoingID
            outgoing:
              candidate?.data?.outgoingID || candidate?.data?.action?.ClientID,
            incoming: clientID,
            metadata: edgeMetadata,
          };
      }
    })();

    if (!edge.outgoing || !edge.incoming) {
      console.warn("invalid edge created", edge, candidate);
      return;
    }

    setMutationThrottled({
      action: {
        clientID: clientID,
        name: action.latest.name,
        dsn: action.dsn,
        metadata: {},
        version: null,
      },
      edge,
    });
  };

  return (
    <div
      ref={ref}
      style={{ height: "100%" }}
      onDragOver={onDragOver}
      onDrop={() => {
        // Add the mutation to our workflow and remove the current mutation state.
        if (mutation) {
          dispatch({
            type: "addAction",
            mutation,
          });
          setMutation(null);
          setMutationThrottled(null);
        }
      }}
      onMouseMove={() => {
        setMutation(null);
        setMutationThrottled(null);
      }}
    >
      <ReactFlow
        minZoom={0.25}
        maxZoom={3}
        elements={layout}
        nodeTypes={nodeTypes}
        snapToGrid={true}
        snapGrid={[gridSize, gridSize]}
        nodesDraggable={false}
        connectionLineType={"step" as ConnectionLineType}
        onPaneContextMenu={(e: React.SyntheticEvent) => {
          e.preventDefault();
        }}
      >
        <Background color="#666" gap={gridSize} />
      </ReactFlow>
    </div>
  );
};
