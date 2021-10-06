import React from "react";
import styled from "@emotion/styled";
import { NodeProps } from "react-flow-renderer";
import { conditionalHeight } from "../consts";
import Action from "./Action";
import { useWorkflowContext, EdgeMetadata, EdgeMetadataIf } from "../../state";

const getInnerContent = (metadata: EdgeMetadata) => {
  /*
  if ((metadata as EdgeMetadataElse).isElse) {
    return (
      <>
        <span>All others</span>
      </>
    );
  }

  if ((metadata as EdgeMetadataRandom).ratio) {
    return (
      <>
        <code>{(metadata as EdgeMetadataRandom).ratio}%</code>
        <span>of the time</span>
      </>
    );
  }
  */

  if ((metadata as EdgeMetadataIf).if) {
    return (
      <>
        <code>{(metadata as EdgeMetadataIf).if}</code>
        <span>If this is true</span>
      </>
    );
  }
  return null;
};

const ConditionalAction = (props: NodeProps) => {
  const [state] = useWorkflowContext();

  // XXX: Put edge map in workflow state - this is used in layout and in each action.
  const edge = state.incomingActionEdges[props.id as any];
  // TODO: Get correct edge from parent

  return (
    <div>
      <Condition>{edge && getInnerContent(edge[0].metadata)}</Condition>
      <Action {...props} />
    </div>
  );
};

export default ConditionalAction;

const Condition = styled.div`
  background: #f6f6f6;
  border-top-left-radius: 4px;
  border-top-right-radius: 4px;
  height: ${conditionalHeight + 4}px;
  padding: 3px 0 4px;
  margin-bottom: -4px;

  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;

  color: #303e2faa;
  text-align: center;
  font-size: 11px;
  line-height: 1.2;

  code,
  span {
    display: block;
  }
  code {
    font-weight: 600;
  }
`;
