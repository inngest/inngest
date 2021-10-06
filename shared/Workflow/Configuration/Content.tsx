import React from "react";
import { getContent } from "./consts";
import { WorkflowAction, useWorkflowContext } from "../state";
import { State as ConfigurationState } from "./reducer";
import { useCurrentWorkspace } from "src/state/workspaces";
import { Action } from "src/types";

type Props = {
  action: WorkflowAction;
  state: ConfigurationState;
  abstractAction: Action;
  setDirty: () => void;
  setName: (s: string) => void;
  onMetadataKeyChange: (key: string, value: string | number) => void;
  onMetadataChange: (m: Object) => void;

  previewTemplates: boolean;
};

const Content: React.FC<Props> = ({
  state,
  abstractAction,
  action,
  setDirty,
  setName,
  onMetadataChange,
  onMetadataKeyChange,
}) => {
  const w = useCurrentWorkspace();
  const [workflowState] = useWorkflowContext();
  const gqlAction = workflowState.actions.find((a) => a.dsn === action.dsn);

  if (!gqlAction) {
    return null;
  }

  const C = getContent(action.dsn);

  return (
    <>
      <label>Name</label>
      <input
        style={{ marginBottom: "30px" }}
        defaultValue={action.name}
        onChange={(e) => {
          setName(e.target.value);
        }}
      />
      <C
        action={action}
        abstractAction={abstractAction}
        state={state}
        workspaceID={w.id}
        setDirty={setDirty}
        onMetadataKeyChange={onMetadataKeyChange}
        previewTemplates={!!state.previewTemplates}
        onChange={(metadata: Object) => {
          onMetadataChange(metadata);
        }}
      />
    </>
  );
};

export default Content;
