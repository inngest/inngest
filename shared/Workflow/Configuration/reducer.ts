import { useReducer } from "react";
import { WorkflowEdge } from "../state";
import { AvailableData } from "../data";

type Tab = "configuration" | "callers" | "data";

export type State = {
  name?: string;
  dirty: boolean;
  metadata: { [key: string]: any };
  incomingEdges: WorkflowEdge[];
  tab: Tab;

  availableData?: AvailableData;

  showSelectEvent?: boolean;
  previewTemplates?: boolean;
};

export type Action =
  | { type: "dirty"; dirty: boolean }
  | { type: "metadata"; metadata: Object }
  | { type: "name"; name: string }
  | { type: "metadataKey"; key: string; value: string | number }
  | { type: "tab"; tab: Tab }
  | { type: "edges"; incomingEdges: WorkflowEdge[] }
  | { type: "selectEvent"; to: boolean }
  | { type: "previewTemplates"; to: boolean }
  | { type: "availableData"; data?: AvailableData }
  | {
      type: "reset";
      metadata?: Object;
      incomingEdges: WorkflowEdge[];
      tab: Tab;
    };

export type Dispatch = (a: Action) => void;

const defaultState = {
  dirty: false,
  metadata: {},
  incomingEdges: [],
  tab: "configuration" as Tab,
};

const reducer = (s: State, a: Action): State => {
  switch (a.type) {
    case "dirty":
      return { ...s, dirty: a.dirty };
    case "name":
      return { ...s, name: a.name, dirty: true };
    case "metadata":
      const dirty = JSON.stringify(a.metadata) !== JSON.stringify(s.metadata);
      return { ...s, metadata: a.metadata, dirty };
    case "metadataKey":
      const metadata = Object.assign({}, s.metadata, {
        [a.key]: a.value,
      });
      return {
        ...s,
        metadata: metadata,
        dirty: s.dirty || s.metadata[a.key] !== a.value,
      };
    case "edges":
      return { ...s, incomingEdges: a.incomingEdges, dirty: true };
    case "tab":
      return { ...s, tab: a.tab };
    case "selectEvent":
      return { ...s, showSelectEvent: a.to };
    case "previewTemplates":
      return { ...s, previewTemplates: a.to };
    case "availableData":
      return { ...s, availableData: a.data };
    case "reset":
      return {
        ...defaultState,
        tab: a.tab,
        incomingEdges: a.incomingEdges,
        metadata: a.metadata || {},
      };
  }
};

// useConfigReducer returns a new reducer for use when configuring a single
// action.  It stores the metadata and sidepanel configuration for the action.
export const useConfigReducer = (defaults: { tab: Tab }) => {
  return useReducer(reducer, { ...defaultState, ...defaults });
};
