import React from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import { baseCSS } from "./css";
import { Handle, Position, NodeProps } from "react-flow-renderer";
import { addW, addH, newClientID } from "../consts";
import Plus from "src/shared/Icons/Add";
import { useWorkflowContext } from "../../state";

const paddingH = (addH - 40) / 2;
const paddingW = (addW - 40) / 2;

const Add = (props: NodeProps) => {
  const { outgoingID } = props.data;
  const [state, dispatch] = useWorkflowContext();

  const onClick = () => {
    dispatch({
      type: "addMutations",
      mutations: state.addMutations.concat([
        {
          edge: {
            outgoing:
              outgoingID === "trigger" ? outgoingID : parseInt(outgoingID, 10),
            incoming: state.workflow ? newClientID(state) : 1,
          },
        },
      ]),
    });
  };

  const selected =
    state.selectedAddNode && state.selectedAddNode.id === props.id;

  return (
    <div style={{ padding: `${paddingH}px ${paddingW}px` }}>
      <Handle
        type="target"
        position={"top" as Position}
        style={{ opacity: 0, top: 20 }}
      />
      <Wrapper onClick={onClick} css={selected && selectedCSS}>
        <Plus width="14" height="14" fill="#888" />
      </Wrapper>
      <Handle
        type="source"
        position={"bottom" as Position}
        style={{ opacity: 0, bottom: 20 }}
      />
    </div>
  );
};

export default Add;

const Wrapper = styled.div`
  ${baseCSS};

  padding: 0;
  width: 40px;
  height: 40px;

  padding: 0;
  border: 1px solid #eee;
  color: #103f10;
  line-height: 1.5;
  position: relative;
  text-align: center;
  display: flex;
  justify-content: center;
  align-items: center;
  border-radius: 40px;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

  .tag {
    position: absolute;
    top: -10px;
    left: calc(50% - 56px);
  }

  > div {
    padding-top: 2px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  small {
    font-size: 12px;
    opacity: 0.6;
  }
`;

const selectedCSS = css`
  border: 2px solid #ceecce;
`;
