import React, { createContext, useContext, useReducer, useEffect } from "react";
// import { Action as BaseAction, IntegrationEvent } from "src/types";
// import { useActionsWithCategory } from "src/shared/Actions/query";
import { init } from "./parse";
// import { Event, useEventDetails, RecentEvent } from "./queries";
// import { apiURL } from "src/utils";

// DefaultStateProps are props passed in to the context wrapper that set
// default state in our reducer, used when editing an existing workflow.
export type DefaultStateProps = {
  workflowID?: string;
  isLatest?: boolean;
  version?: {
    version: number;
    config: string;
    description?: string | null;
    validFrom: string | null;
    validTo: string | null;
  };
};

export type StateProps = {
  state: State;
  dispatch: Dispatcher;
};

// Workflow is the shape of Workflow returned from parsing via webassembly.
export type Workflow = {
  name: string;
  triggers: WorkflowTrigger[];
  actions: WorkflowAction[];
  edges: WorkflowEdge[];
};

export type WorkflowTrigger = {
  event?: string;
  cron?: string;
};

export type WorkflowAction = {
  clientID: number;
  name: string;
  dsn: string;
  metadata: { [key: string]: any };
  version: null | number;
};

export type WorkflowEdge = {
  outgoing: number | "trigger";
  incoming: number;
  metadata?: EdgeMetadata;
};

// EdgeMetadata structures types for known edge metadata
export type EdgeMetadata = undefined | EdgeType;

export type EdgeType = {
  name?: string;
  if?: string;
} & (EdgeTypeEdge | EdgeTypeAsync);

export type EdgeTypeEdge = {
  type: "edge";
};

export type EdgeTypeAsync = {
  type: "async";
  async: {
    ttl: string;
    event: string;
    match?: string;
  };
};

export type EdgeMetadataIf = EdgeType;

export function isEdgeMetadataIf(edge: EdgeMetadata): edge is EdgeMetadataIf {
  return edge !== undefined && (edge as EdgeMetadataIf).if !== undefined;
}

export function isEdgeMetadataAsync(edge: EdgeMetadata): edge is EdgeTypeAsync {
  return edge !== undefined && (edge as EdgeMetadataIf).type === "async";
}

export type EdgeMetadataElse = {
  isElse: true;
};

export type Tool = "debug" | "logs" | "run";

export type State = {
  // id represents the uuid of the workflow we're editing, if we're editing a
  // workflow (as opposed to creating a workflow)
  id?: string;

  mode: "graph" | "code";
  tool?: Tool;

  // The final cue configuration for the workflow.  This is our source of truth.
  config: string;
  // Our parsed workflow into data structures that the UI can visualize.
  workflow: Workflow | null;
  parseError: string | null;
  saving: boolean;
  dirty?: boolean;

  // exampleEvent is an event from the history store that's chosen to show example
  // data that may be present within the workflow.
  exampleEvent?: RecentEvent;

  // integrationEvent is the event information fetched from our integrations endpoint.
  // The integrations API can be searched by event name, and each integration can provide
  // detailed information about integration events.
  //
  // This data can be used when making edge expressions, within autocompletes, etc.
  integrationEvent?: IntegrationEvent;

  // dragMutation represents the temporary action that we are placing.  If this is null,
  // there is no action currently being placed. This is added via drag-and-drop.
  dragMutation: GraphMutation | null;

  // addMutations represents temporary actions we are placing after hitting "Add node"
  // to an existing node.
  addMutations: GraphMutation[];

  // triggerEventDetails stores deatils about each of the workflow's triggers, such as
  // the shape/type definition of the event fields.  This is useful when configuring events
  // and is used only within the UI.  It should not be relied upon for making workflows.
  //
  // TODO Ensure that this only has the versions specified by a WorkflowTrigger.
  triggerEventDetails: Event[];

  // dragAction represents the abstract action we're dragging from the sidebar,
  // or moving in the graph.
  //
  // The drag/drop spec doesn't allow onDragOver to look into dataTransfer, therefore we can't
  // use this to preview the action youre placing in the flowchart.  State allows us to
  // get this.
  dragAction?: BaseAction | null;

  // moveAction represents the concrete WorkflowAction with metadata that's being moved.
  // If this exists on drop, this WorkflowAction should be added to the graph vs a new one
  // being created.
  moveAction?: WorkflowAction | null;

  // available actions
  actions: BaseAction[];

  // selectedAddNode stores which "add" node has been selected after
  // clicking.
  selectedAddNode?: SelectedAddNode | null;

  // The client ID of the workflow that we're configuring
  configuring: number | null;
  configurationTab?: "configuration" | "callers";

  // Whether we're showing a warning to remove a node. This exists as the
  // "hover card" must be able to render a top-level confirm modal.
  removeConfirmClientID?: number | null;

  // DEBUG UI
  // ========

  pausedOn: {
    // A list of client IDs that the debugger is paused on, mapped to the async token
    // UUID used to resume the workflow.
    [clientID: string]: string;
  };

  // Denormalization of workflow items below
  // ===============

  // incomingActionEdges represents a mapping of action client IDs to its incoming edge
  // (edges in the future when supported).
  //
  // We need this when constructing the graph editor because the visuals are different
  // if the edge metadata has conditions, etc.
  incomingActionEdges: {
    [clientID: string]: WorkflowEdge[];
  };
  // workflowActions stores a map of client IDs to actions within the workflow for fast lookup
  workflowActions: { [clientID: string]: WorkflowAction };
};

export type GraphMutation = {
  // Action represents the action we are adding to a specific node.
  action?: WorkflowAction;
  // if we're adding a new source action we need no edge;  it's automatically handled for us by
  // creating an edge from the trigger to sources by default.
  edge: WorkflowEdge;
};

type SelectedAddNode = {
  id: string;
  outgoingID: number | "trigger"; // The incomingID for the new action (the parent action or trigger).
  edgeMetadata: Object | undefined;
};

// Dispatcher is the type of the dispatch fn passed down to components
export type Dispatcher = (a: Action) => void;

const defaultState: State = {
  mode: "graph" as "graph",
  // the configuration either derived from our code editor, or serialized from the workflow
  // from the graph editor. The last edit mode determines how this was created.
  config: "",
  // the Workflow type derived from the cue configuration in our code editor, or from the
  // graph editor.  The last edit mode determines how this was created.
  workflow: null,
  parseError: null,
  dragMutation: null,
  addMutations: [],
  saving: false,
  configuring: null,
  actions: [],
  triggerEventDetails: [],
  pausedOn: {},
  incomingActionEdges: {},
  workflowActions: {},
};

const emptyWorkflow = {
  name: "",
  actions: [],
  triggers: [],
  edges: [],
};

type WorkflowKeys = keyof Workflow;

export type Action =
  | { type: "edit-workflow"; property: WorkflowKeys; value: any }
  | { type: "new-workflow"; name: string; triggers: WorkflowTrigger[] }
  | { type: "triggers"; triggers: WorkflowTrigger[] }
  | { type: "events"; events: Event[] }
  | { type: "exampleEvent"; event?: RecentEvent }
  | { type: "integrationEvent"; event?: IntegrationEvent }
  | { type: "mode"; mode: "graph" | "code" }
  | { type: "dragMutation"; mutation: GraphMutation | null }
  | { type: "addMutations"; mutations: GraphMutation[] }
  | { type: "config"; config: string }
  | { type: "setTool"; tool?: "debug" | "logs" | "run" }
  | { type: "addAction"; mutation: GraphMutation }
  | { type: "removeConfirm"; clientID?: number | null } // show confirm modal before removing
  | { type: "removeAction"; clientID: number }
  | { type: "workflowID"; workflowID: string | undefined }
  | { type: "saving"; saving: boolean; dirty: boolean }
  | { type: "actions"; actions: BaseAction[] }
  | {
      type: "configure";
      clientID: number | null;
      tab?: "configuration" | "callers";
    }
  | { type: "toggleAddNode"; node: SelectedAddNode | null }
  | { type: "addPausedOn"; clientID: string; uuid: string }
  | { type: "removePausedOn"; clientID: string }
  | { type: "clearPausedOn" }
  | {
      type: "updateAction";
      metadata: Object;
      incomingEdges: WorkflowEdge[];
      name?: string;
    }
  | {
      type: "setDragAction";
      dragAction: BaseAction | null;
      moveAction: WorkflowAction | null;
    };

const reducer = (s: State, a: Action): State => {
  switch (a.type) {
    case "edit-workflow": {
      // edit an entire workflow property wholesale, eg. its name
      const workflow: Workflow = Object.assign({}, s.workflow || emptyWorkflow);
      workflow[a.property] = a.value;

      const config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      return {
        ...s,
        workflow,
        config,
        parseError: null,
        dirty: true,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
      };
    }

    case "new-workflow": {
      // set triggers and workflow name at once
      const workflow: Workflow = Object.assign({}, s.workflow || emptyWorkflow);
      workflow.name = a.name;
      workflow.triggers = a.triggers;

      const config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      return {
        ...s,
        workflow,
        config,
        parseError: null,
        dirty: true,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
      };
    }

    case "triggers": {
      const workflow: Workflow = Object.assign({}, s.workflow || emptyWorkflow);
      workflow.triggers = a.triggers;

      const config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      return {
        ...s,
        workflow,
        config,
        parseError: null,
        dirty: true,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
      };
    }

    case "dragMutation": {
      return Object.assign({}, s, { dragMutation: a.mutation });
    }

    case "addMutations": {
      return Object.assign({}, s, {
        addMutations: a.mutations.slice(),
        dirty: true,
      });
    }

    // addAction adds the action from dragging/dropping on the dag, or from confirming
    // the mutation should exist.
    case "addAction": {
      const workflow: Workflow = Object.assign({}, s.workflow || emptyWorkflow);

      if (!a.mutation.action || !a.mutation.edge) {
        return s;
      }

      // We're always dragging the incoming action, so don't allow this to be different.
      if (a.mutation.edge.incoming !== a.mutation.action.clientID) {
        console.warn("mismatched action client id and incoming edge");
        a.mutation.edge.incoming = a.mutation.action.clientID;
      }

      workflow.actions = workflow.actions.concat([a.mutation.action]);
      if (a.mutation.edge) {
        workflow.edges = workflow.edges.concat([a.mutation.edge]);
      } else {
        // Add an edge to the trigger.
        workflow.edges = workflow.edges.concat([
          {
            outgoing: "trigger",
            incoming: a.mutation.action.clientID,
            metadata: undefined,
          },
        ]);
      }

      let config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      if (!config) {
        // With no config do nothing, as this is an invalid op.
        return s;
      }

      const cycles = detectCycles(workflow.edges);

      // Remove the action ID from the mutation nodes, if necessary

      return {
        ...s,
        parseError: cycles
          ? "This workflow has cycles which are currently unsupported."
          : null,
        dragAction: null,
        selectedAddNode: null,
        dragMutation: null,
        dirty: true,
        addMutations: s.addMutations.filter(
          (m) => m.edge.outgoing !== a.mutation.edge.outgoing
        ),
        workflow,
        config,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
      };
    }

    case "config":
      try {
        const parsed = window.parseCue
          ? (window.parseCue(a.config) as Workflow)
          : "Initializing config parser";
        if (typeof parsed === "string") {
          return { ...s, config: a.config, parseError: parsed };
        }

        const cycles = detectCycles(parsed.edges);

        return {
          ...s,
          config: a.config,
          dirty: a.config !== s.config,
          workflow: parsed,
          incomingActionEdges: incomingActionEdges(parsed as Workflow),
          workflowActions: workflowActions(parsed as Workflow),
          parseError: cycles
            ? "This workflow has cycles which are currently unsupported."
            : null,
        };
      } catch (e) {
        console.warn("parse error: ", e);
        return {
          ...s,
          config: a.config,
          parseError: "Error parsing your config file",
        };
      }

    case "mode":
      return { ...s, mode: a.mode };

    case "removeConfirm": {
      return { ...s, removeConfirmClientID: a.clientID };
    }
    case "addPausedOn": {
      return {
        ...s,
        pausedOn: { ...(s.pausedOn || {}), [a.clientID]: a.uuid },
      };
    }
    case "removePausedOn": {
      const pausedOn = Object.assign({}, s.pausedOn || {});
      delete pausedOn[a.clientID];
      return { ...s, pausedOn };
    }
    case "clearPausedOn": {
      return { ...s, pausedOn: {} };
    }
    case "removeAction": {
      if (!s.workflow) {
        return s;
      }
      const workflow = removeAction(a.clientID, s.workflow);
      const config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      return {
        ...s,
        workflow,
        config,
        dirty: true,
        selectedAddNode: null,
        removeConfirmClientID: null,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
      };
    }
    case "setDragAction":
      return { ...s, dragAction: a.dragAction, moveAction: a.moveAction };
    case "workflowID":
      return { ...s, id: a.workflowID };
    case "saving":
      return { ...s, saving: a.saving, dirty: a.dirty };
    case "actions":
      return { ...s, actions: a.actions };
    case "events":
      return { ...s, triggerEventDetails: a.events };
    case "setTool":
      return { ...s, tool: a.tool };
    case "configure":
      return {
        ...s,
        configuring: a.clientID,
        selectedAddNode: null,
        configurationTab: a.tab,
      };
    case "exampleEvent":
      return { ...s, exampleEvent: a.event };
    case "integrationEvent":
      return { ...s, integrationEvent: a.event };
    case "updateAction": {
      // setActionMetadata updates the currently configuring action's
      // metadata.
      const workflow: Workflow = Object.assign({}, s.workflow || emptyWorkflow);
      const index = workflow.actions.findIndex(
        (a) => a.clientID === s.configuring
      );

      if (index < 0) {
        return s;
      }

      workflow.actions = workflow.actions.slice();
      workflow.actions[index].metadata = a.metadata;
      if (a.name) {
        workflow.actions[index].name = a.name;
      }

      // Remove the previous action edges and add the edges to the workflow.
      // Note that we want to do this in-place to keep the edge ordering.
      const outgoingIDs: { [id: string]: WorkflowEdge } = {};
      a.incomingEdges.forEach((e) => (outgoingIDs[e.outgoing] = e));

      workflow.edges = workflow.edges
        .slice()
        .map((e) => {
          if (e.incoming !== s.configuring) {
            return e;
          }
          if (!outgoingIDs[e.outgoing]) {
            return e;
          }
          const copy = outgoingIDs[e.outgoing];
          // Remove this from the list, leaving only new edges.
          delete outgoingIDs[e.outgoing];
          return copy;
        })
        .filter(Boolean)
        .concat(Object.values(outgoingIDs)) as WorkflowEdge[];

      const config = window.serializeCue
        ? window.serializeCue(JSON.stringify(workflow))
        : s.config;

      return {
        ...s,
        workflow: { ...workflow },
        dirty: true,
        incomingActionEdges: incomingActionEdges(workflow),
        workflowActions: workflowActions(workflow),
        config,
      };
    }
    case "toggleAddNode":
      return { ...s, selectedAddNode: a.node };
    default:
      return s;
  }
};

export const useWorkflowState = (p: DefaultStateProps): [State, Dispatcher] => {
  // Set the default state, using any provided config if possible.
  //
  // Note that this only sets the config string in state;  it doesn't yet parse the
  // config into types that we need to display things within the UI.
  let defaults = Object.assign({}, defaultState);
  if (p.version) {
    defaults = Object.assign({}, defaultState, { config: p.version.config });
  }

  const [state, dispatch] = useReducer(reducer, defaults);

  // Initialize webassembly cue/config parsing and generation.
  const initializeParsing = async () => {
    await init();

    // And if we have default state, this is where we need to set it so that
    // it's properly parsed and we generate our data structures for types, etc.
    if (p.version) {
      dispatch({
        type: "config",
        config: p.version.config,
      });
    }
  };
  useEffect(() => {
    initializeParsing();
  }, []);

  return [state, dispatch];
};

type ContextState = [State, (a: Action) => void];

export const WorkflowContext = createContext<ContextState>([
  {} as State,
  (_a: Action) => {},
] as ContextState);

export const useWorkflowContext = () => {
  return useContext(WorkflowContext);
};

export const WorkflowStateProvider: React.FC<DefaultStateProps> = (props) => {
  const [state, dispatch] = useWorkflowState(props);
  // const [{ data }] = useActionsWithCategory(false);
  const names: string[] = state.workflow
    ? (state.workflow.triggers.map((t) => t.event).filter(Boolean) as string[])
    : [];

  // When the triggers change we want to fetch the event information from our server.
  // The event information is used when configuring every action from the UI; we need the
  // types/fields in the event.
  useEffect(() => {
    if (!state.workflow || state.workflow.triggers.length === 0) {
      return;
    }
  }, [state.workflow && state.workflow.triggers]);

  useEffect(() => {
    // We may have saved a draft, redirected, and attempted to debug/run immediately.  If so
    // load this specific tool.
    if (window.location.hash !== "") {
      const tool = window.location.hash.replace("#", "");
      if (["debug", "logs", "run"].indexOf(tool) > -1) {
        dispatch({ type: "setTool", tool: tool as Tool });
      }
    }
  }, []);

  // Whenever the trigger changes, find the integration event if possible.
  // useIntegrationEvent(state?.workflow?.triggers || [], dispatch);

  return (
    <WorkflowContext.Provider value={[state, dispatch]}>
      {props.children}
    </WorkflowContext.Provider>
  );
};

/*
const useIntegrationEvent = (
  triggers: WorkflowTrigger[],
  dispatch: Dispatcher
) => {
  const fetchAPI = async (event: string) => {
    try {
      const result = await fetch(
        apiURL(`/v1/public/integrations?event=${event}`)
      );
      const data = await result.json();
      const evt = (data?.events || {})[event];
      if (!evt) {
        throw new Error("event not found");
      }
      dispatch({ type: "integrationEvent", event: evt as IntegrationEvent });
    } catch (e) {
      dispatch({ type: "integrationEvent" });
    }
  };

  useEffect(() => {
    triggers.forEach((t) => t.event && fetchAPI(t.event));
  }, [triggers]);
};
  */

// removeAction removes an action from a workflow and reparents all of the children
// of the removed action to its parent.
const removeAction = (clientID: number, w: Workflow): Workflow => {
  const parents = w.edges.filter((e) => e.incoming === clientID);
  const children = w.edges.filter((e) => e.outgoing === clientID);

  // For each parent, make a new edge pointing to each child.  Take the metadata
  // from the parent, as the parent action dicates metadata (eg. the parent is an "if",
  // and the parent edge itself has metadta regarding true/false).
  const newEdges = parents
    .map((pe) => {
      return children.map((ce) => ({
        outgoing: pe.outgoing,
        incoming: ce.incoming,
        metadata: pe.metadata,
      }));
    })
    .flat();

  const other = w.edges.filter(
    (e) => e.incoming !== clientID && e.outgoing !== clientID
  );

  return {
    ...w,
    actions: w.actions.filter((a) => a.clientID !== clientID),
    edges: other.concat(newEdges),
  };
};

// reparentAction moves a single action to a new parent.  All children of the moved client
// remain children of the moved node.
//
// If the parentclientID is null the given action will be reparented as a source.
//
// This allows us to move an action without copying.
//
// TODO: Implement drag/drop of current actions.
// TODO: Unit test
const reparentAction = (
  clientID: number,
  parentclientID: number | null,
  w: Workflow
): Workflow => {
  const edges = w.edges.filter((e) => e.incoming !== clientID);

  if (parentclientID) {
    return {
      ...w,
      edges: edges.concat([
        {
          outgoing: parentclientID,
          incoming: clientID,
          metadata: undefined,
        },
      ]),
    };
  }

  return {
    ...w,
    edges: edges,
  };
};

// detectCycles returns true if there are cycles detected in the workflow DAG.
//
// Detecting cycles in a DAG is slightly different to cycle detection in a tree.
// We can either use Tarjan's SCC algorithm and see if there are SCCs with a size
// > 1, or we can perform a BFS through the workflow and see if we encounter the same
// edge > 1 time.
//
// We opt for the latter, here.
export const detectCycles = (edges: Array<WorkflowEdge>) => {
  const outgoingEdges: { [clientID: string]: Array<WorkflowEdge> } = {};
  edges.forEach((e) => {
    if (!Array.isArray(outgoingEdges[e.outgoing])) {
      outgoingEdges[e.outgoing] = [];
    }
    outgoingEdges[e.outgoing].push(e);
  });

  // Start at the trigger and work our way down.
  const queue: Array<{ node: "trigger" | number; depth: number }> = [
    { node: "trigger", depth: 0 },
  ];
  let parents = new Set();

  while (queue.length > 0) {
    const item = queue.shift();
    if (!item) {
      break;
    }

    // Iterate through each child of the current item
    const children = outgoingEdges[item.node] || [];
    if (item.depth < parents.size) {
      parents = new Set(Array.from(parents).slice(0, item.depth));
    }

    parents.add(item.node);

    const seen = new Set();
    for (let c of children) {
      const key = c.incoming;
      if (parents.has(key)) {
        // We've seen this particular edge before;  there are cycles.
        return true;
      }
      if (seen.has(c.incoming)) {
        // duplicative edges
        return true;
      }
      seen.add(c.incoming);
      // Add the child of the edge to be processed next.
      queue.unshift({ node: c.incoming, depth: item.depth + 1 });
    }
  }

  return false;
};

const incomingActionEdges = (
  w: Workflow
): { [actionID: number]: WorkflowEdge[] } => {
  // TODO: Support for multiple incoming edges.
  const actionEdgeMap: { [clientID: number]: WorkflowEdge[] } = {};
  w.edges.map((e) => {
    if (!Array.isArray(actionEdgeMap[e.incoming])) {
      actionEdgeMap[e.incoming] = [] as WorkflowEdge[];
    }
    actionEdgeMap[e.incoming].push(e);
  });
  return actionEdgeMap;
};

const workflowActions = (
  w: Workflow
): { [actionID: number]: WorkflowAction } => {
  const map: { [clientID: number]: WorkflowAction } = {};
  w.actions.map((a) => {
    map[a.clientID] = a;
  });
  return map;
};
