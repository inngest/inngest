// @ts-nocheck
import React from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import { baseCSS } from "./css";
import { WorkflowAction } from "../../state";

type Props = {
  Icon: React.FC<{ size: number }>;
  action: WorkflowAction;
  draggable?: boolean;
  mutation?: boolean;
  onMouseOver?: () => void;
  onClick?: () => void;
};

const ActionNode: React.FC<Props> = (props) => {
  const { draggable, mutation, action, onMouseOver, onClick, Icon } = props;
  return (
    <NodeCard
      className="inngest--node-card"
      css={[mutation && mutationCSS, draggable && draggableCSS]}
      onClick={() => onClick && onClick()}
      onMouseOver={() => {
        onMouseOver && onMouseOver();
      }}
    >
      <Icon size={20} color="#fff" />
      <div>
        <p>{action.name}</p>
        {draggable && (
          <small>
            <b>Drag to copy</b>
          </small>
        )}

        {!draggable &&
          (Object.keys(action.metadata || {}).length > 0 ? (
            <small>Configured</small>
          ) : (
            <small />
          ))}
      </div>
    </NodeCard>
  );
};

export default ActionNode;

// NodeCard is the actual node box that shows in the editor.
const NodeCard = styled.div`
  ${baseCSS}
  cursor: default;
  z-index: 5;
  padding-left: 1px;

  display: grid;
  grid-template-columns: 55px auto;

  align-items: center;

  p {
    line-height: 1.2;
    margin-bottom: 1px;
  }

  > svg {
    align-self: center;
    justify-self: center;
    opacity: 0.5;
  }

  small {
    opacity: 0.6;
  }
`;

const mutationCSS = css`
  opacity: 0.5;
`;

const draggableCSS = css`
  opacity: 1;
  box-shadow: 0 5px 30px rgba(0, 0, 0, 0.2), 0 1px 4px rgba(0, 0, 0, 0.15);
  cursor: move;
`;
