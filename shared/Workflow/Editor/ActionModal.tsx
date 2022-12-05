import React, { useMemo } from "react";
import styled from "@emotion/styled";
import Box from "../../legacy/Box";
import ActionGrid from "../../Actions/Grid";
import Navigator from "../../Actions/Navigator";
import { useWorkflowContext, EdgeMetadata } from "../state";
import { newClientID } from "./consts";
import { Action } from "src/types";

const ActionModal = () => {
  const [state, dispatch] = useWorkflowContext();
  const node = state.selectedAddNode;

  const actions = useMemo(() => {
    return state.actions.map((a) => {
      return { ...a, category: { name: a.category.name } };
    });
  }, [state.actions.length]);

  // In order to show the edges available for the current node we must find the
  // action for the parent.
  const wa = state.workflowActions[node ? node.outgoingID : ""];
  const parentAction = actions.find((a) => wa && a.dsn === wa.dsn);
  if (node && node.outgoingID !== "trigger" && (!wa || !parentAction)) {
    return null;
  }

  const onClick = (a: Action) => {
    if (!node || !state.workflow) {
      // must always be visible to show this modal
      return;
    }

    const clientID = newClientID(state);
    dispatch({
      type: "addAction",
      mutation: {
        action: {
          clientID: clientID,
          name: a.latest.name,
          dsn: a.dsn,
          metadata: {},
          version: null,
        },
        edge: {
          outgoing: node.outgoingID,
          incoming: clientID,
          metadata: node.edgeMetadata as EdgeMetadata,
        },
      },
    });
  };

  return (
    <Wrapper nopadding>
      <Navigator actions={actions} onClick={onClick} />
    </Wrapper>
  );
};

export default ActionModal;

const Wrapper = styled(Box)`
  max-height: 80vh;
  width: 80vw;
  overflow: auto;

  h2 {
    margin-bottom: 20px;
  }
`;
