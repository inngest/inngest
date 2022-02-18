import React from "react";
import styled from "@emotion/styled";
import { Handle, Position, NodeProps } from "react-flow-renderer";
import { conditionalHeight, nodeW } from "../consts";
import { EdgeMetadata, EdgeMetadataIf, useWorkflowContext } from "../../state";

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
  if (metadata && metadata.type === "async") {
    return (
      <>
        <code>{metadata.async.event}</code>
        <span>When this event is received within {metadata.async.ttl}</span>
      </>
    );
  }

  if ((metadata as EdgeMetadataIf).if) {
    const len = metadata?.if?.length || 0;

  if (metadata.name) {
      return <strong>{metadata.name}</strong>;
    }

    return (
      <>
        {len < 30 ? (
            <code>{(metadata as EdgeMetadataIf).if}</code>
          ) : (
            <strong>Conditional</strong>
          )
        }
        <span>Only ran when the condition is true</span>
      </>
    );
  }

  return null;
};

const ConditionalAction = (props: NodeProps) => {
  const [, dispatch] = useWorkflowContext();
  const edge = props.data.edge;

  const isBlank =
    !edge.metadata ||
    !edge.metadata.type ||
    (edge.metadata.type === "edge" && edge.metadata.if === "");

  return (
    <>
      <Handle
        type="target"
        position={"top" as Position}
        style={{ opacity: 0, top: 30, pointerEvents: "none" }}
      />
      {isBlank ? (
        <Blank></Blank>
      ) : (
        <Condition
          onClick={() => {
            dispatch({
              type: "configure",
              tab: "callers",
              clientID: edge.incoming,
            });
          }}
        >
          {edge && getInnerContent(edge.metadata)}
        </Condition>
      )}
      <Handle
        type="source"
        position={"bottom" as Position}
        style={{ opacity: 0, bottom: 30, pointerEvents: "none" }}
      />
    </>
  );
};

export default ConditionalAction;

const Blank = styled.div`
  height: ${conditionalHeight}px;
  background: transparent;
  cursor: default;
  pointer-events: none;
  width: ${nodeW}px;
`;

const Condition = styled.div`
  cursor: pointer;
  background: var(--gray);
  border-top-left-radius: 4px;
  border-top-right-radius: 4px;
  border-radius: 4px;
  height: ${conditionalHeight}px;
  width: ${nodeW}px;
  padding: 3px 0 4px;

  border: 1px solid var(--gray);

  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;

  text-align: center;
  font-size: 11px;
  line-height: 1.2;

  code,
  span {
    display: block;
  }

  strong {
    opacity: 0.85;
  }

  span {
    opacity: 0.75;
  }
`;
