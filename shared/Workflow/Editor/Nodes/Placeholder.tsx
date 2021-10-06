import React from "react";
import styled from "@emotion/styled";
import { Handle, Position, NodeProps } from "react-flow-renderer";
import { nodeW, nodeH } from "../consts";
import { GraphMutation, WorkflowAction, useWorkflowContext } from "../../state";

type Props = NodeProps & {
  data: {
    mutation: GraphMutation;
    action?: WorkflowAction;
  };
};

// Placeholder is the react component that's added to the workflow when
// a user hits "Add child" from the workflow.
//
// It allows users to select which action they want to add, and how.
const Placeholder: React.FC<Props> = (props) => {
  const [, dispatch] = useWorkflowContext();
  const { mutation } = props.data;

  const outgoingID = mutation.edge.outgoing;
  const edgeMetadata = mutation.edge.metadata;

  return (
    <>
      <Handle
        type="target"
        position={"top" as Position}
        style={{ opacity: 0, top: 1 }}
      />
      <Wrapper
        onClick={() => {
          dispatch({
            type: "toggleAddNode",
            node: {
              id: props.id,
              outgoingID,
              edgeMetadata,
            },
          });
        }}
      >
        <label>Select an action...</label>
      </Wrapper>
    </>
  );
};

export default Placeholder;

const Wrapper = styled.div`
  cursor: pointer;
  padding: 20px;
  border: 1px dotted #eee;
  background: #fdfbf6;
  width: ${nodeW}px;
  height: ${nodeH}px;
  text-align: center;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.03);
  box-shadow: rgb(0 0 0 / 15%) 0px 8px 35px;

  display: flex;
  flex-direction: column;
  justify-content: center;
`;
