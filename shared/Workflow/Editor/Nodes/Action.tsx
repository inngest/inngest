// @ts-nocheck
import React, { useState, useEffect } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import Button, { ButtonGroup } from "src/shared/Button";
import {
  Handle,
  Position,
  NodeProps,
  useStoreActions,
} from "react-flow-renderer";
import { WorkflowAction, useWorkflowContext, EdgeMetadata } from "../../state";
import { Action as BaseAction } from "src/types";
import { categoryIcon } from "src/shared/Actions/icons";
import { nodeW, nodeH, newClientID } from "../consts";
import { getHoverContent } from "./cards";
import ActionNode from "./ActionNode";
import Placeholder from "./Placeholder";

const Action = (props: NodeProps) => {
  const [draggable, setDraggable] = useState(false);
  const [state, dispatch] = useWorkflowContext();
  const { selected } = props;

  const paused = state.pausedOn[props.id];

  // Allow us to dyanmically choose which el is selected in react flow.
  // This is needed to manipulate the flow chart's z-indexes to show hover cards.
  //
  // By default, react-flow sets a style of `z-index: 3` to all nodes.
  // For our hover card to show over any other sibling nodes, we need to actually
  // selec this node in react-flow's state.
  const setSelectedElements = useStoreActions(
    (actions) => actions.setSelectedElements
  );

  useEffect(() => {
    if (!state.moveAction || state.moveAction !== workflowAction) {
      // We've no longer got an action in moveAction, or we're coping another action.
      // Set this draggable to false.
      setDraggable(false);
    }
  }, [state.moveAction]);

  const mutation = !!props.data.mutation;
  const workflowAction = props.data.action as WorkflowAction;

  // If this is a mutation _without_ an action, show the placeholder node
  // so that the mutation can be configured.
  if (mutation && !workflowAction) {
    return <Placeholder {...props} />;
  }

  const action = state.actions.find((a) => a.dsn === workflowAction.dsn);
  const Icon = categoryIcon(action?.category?.name);
  const C = getHoverContent(workflowAction.dsn);

  return (
    <>
      <Handle
        type="target"
        position={"top" as Position}
        style={{ opacity: 0, top: 1 }}
      />

      <Wrapper css={[selected && hoverCardVisibleCSS, paused && pausedCSS]}>
        <ActionNode
          action={workflowAction}
          mutation={mutation}
          draggable={draggable}
          onClick={() => {
            dispatch({
              type: "configure",
              clientID: workflowAction.clientID,
            });
          }}
          onMouseOver={() => {
            setSelectedElements([
              {
                id: props.id,
                type: props.type,
              } as any,
            ]);
          }}
          Icon={Icon}
        />
        <HoverCard className="inngest--hover-card">
          <HoverHeader
            onClick={() => {
              dispatch({
                type: "configure",
                clientID: workflowAction.clientID,
              });
            }}
          >
            {action ? (
              <p>{action.tagline}</p>
            ) : (
              <p>This is an unknown or internal action.</p>
            )}
          </HoverHeader>
          {action && (
            <HoverBody>
              {C && <C action={workflowAction} />}
              <Actions
                action={action}
                workflowAction={workflowAction}
                visible={selected}
              />
            </HoverBody>
          )}
        </HoverCard>
        <BG
          className="inngest-hover-bg"
          onMouseOver={() => {
            setSelectedElements([] as any);
          }}
        />
      </Wrapper>

      <Handle
        type="source"
        position={"bottom" as Position}
        style={{ opacity: 0, bottom: 1 }}
      />
    </>
  );
};

export default Action;

type ActionProps = {
  action: BaseAction;
  workflowAction: WorkflowAction;
  visible: boolean;
};

const Actions: React.FC<ActionProps> = ({
  action,
  workflowAction,
  visible,
}) => {
  const [state, dispatch] = useWorkflowContext();
  const [showEdges, setShowEdges] = useState(false);

  useEffect(() => {
    if (!visible) {
      // Fade out then set show edges to false.
      window.setTimeout(() => setShowEdges(false), 300);
    }
  }, [visible]);

  const addPlaceholder = React.useCallback(
    (metadata?: EdgeMetadata) => {
      dispatch({
        type: "addMutations",
        mutations: state.addMutations.concat([
          {
            edge: {
              metadata: metadata,
              outgoing: workflowAction.clientID,
              incoming: state.workflow ? newClientID(state) : 1,
            },
          },
        ]),
      });
    },
    [state, dispatch, workflowAction]
  );

  // Calculate the edges.

  return (
    <ActionWrapper css={[showEdges && showEdgesCSS]}>
      <div>
        <DefaultActions>
          <Button
            kind="danger"
            size="small"
            onClick={(e: React.SyntheticEvent) => {
              e.stopPropagation(); // prevent HoverCard onClick from being called
              action &&
                dispatch({
                  type: "removeConfirm",
                  clientID: workflowAction.clientID,
                });
            }}
          >
            Remove action
          </Button>

          <ButtonGroup right>
            <Button
              size="small"
              onClick={(e: React.SyntheticEvent) => {
                e.stopPropagation();
                if (action.latest.Edges.length === 0) {
                  addPlaceholder();
                  return;
                }

                setShowEdges(true);
              }}
            >
              Add child
            </Button>
            <Button
              kind="primary"
              size="small"
              onClick={() => {
                dispatch({
                  type: "configure",
                  clientID: workflowAction.clientID,
                });
              }}
            >
              Configure
            </Button>
          </ButtonGroup>
        </DefaultActions>

        <EdgeOptions>
          <label>When should this action be triggered?</label>

          <ButtonGroup center style={{ marginTop: 15, flex: 0 }}>
            <Button
              size="small"
              onClick={(e: React.SyntheticEvent) => {
                e.stopPropagation();
                addPlaceholder();
              }}
            >
              Always
            </Button>
            {action?.latest.Edges.map((edge) => {
              return (
                <Button
                  size="small"
                  key={edge.name}
                  onClick={(e: React.SyntheticEvent) => {
                    e.stopPropagation();
                    addPlaceholder(edge);
                  }}
                >
                  On {edge.name}
                </Button>
              );
            })}
          </ButtonGroup>
        </EdgeOptions>
      </div>
    </ActionWrapper>
  );
};

const pausedCSS = css`
  z-index: 5;
  box-shadow: 0 0 0 4px #ceecce, 0 2px 0 4px #ceecce;

  &:after {
    display: block;
    width: 100%;
    content: "PAUSED";
    background: var(--bg-color);
    text-align: center;
    font-size: 10px;
    font-weight: bold;
    padding: 5px 0 0;
    color: #4e754d;
    cursor: default;
  }
`;

const hoverCardVisibleCSS = css`
  z-index: 5;
  .inngest--node-card {
    cursor: pointer;
    box-shadow: 0 3px 8px rgba(0, 0, 0, 0.03);
  }
  .inngest--hover-card {
    cursor: default;
    z-index: 4;
    opacity: 1;
    transform: scale(1);
    pointer-events: auto;
  }
  .inngest-hover-bg {
    pointer-events: auto;
  }
`;

const Wrapper = styled.div`
  position: relative;
  z-index: 1;
`;

const PausedCard = styled.div`
  max-height: 340px;
  position: absolute;
  background: #ceecce;
  background: #fff;
  opacity: 0;
  transform: scale(0.97);
  z-index: 1;
  pointer-events: none;
  box-shadow: rgb(0 0 0 / 30%) 0px 16px 70px;
  border-radius: 4px;

  transition: all 0.3s;
`;

const HoverCard = styled.div`
  display: none;
  width: 560px;
  max-height: 340px;
  position: absolute;
  left: -20px;
  top: -20px;
  border: 1px solid var(--black);
  background: var(--bg-color);
  opacity: 0;
  transform: scale(0.97);
  z-index: 1;
  pointer-events: none;
  box-shadow: rgb(0 0 0 / 30%) 0px 16px 70px;
  border-radius: 4px;

  transition: all 0.3s;
`;

const HoverHeader = styled.div`
  cursor: default;
  background: var(--bg-color);
  height: ${nodeH + 40}px;
  padding: 20px 20px 20px ${nodeW + 40}px;
  display: flex;
  align-items: center;

  border-top-right-radius: 4px;
  border-top-left-radius: 4px;

  p {
    font-size: 12px;
    opacity: 0.6;
  }
`;

const HoverBody = styled.div`
  border-top: 1px solid var(--black);
  padding: 14px 20px;
  background: var(--bg-color);
  height: auto;
  overflow: hidden;

  font-size: 14px;
`;

const DefaultActions = styled.div`
  display: flex;
  justify-content: space-between;
  transition: all 0.3s;
`;

const EdgeOptions = styled.div`
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 5px;

  label {
    padding-top: 20px;
  }
`;

const ActionWrapper = styled.div`
  /* This shows only the default actions initially */
  height: 28px;

  &,
  > div {
    transition: all 0.3s;
  }
`;

const showEdgesCSS = css`
  height: 65px;
  > div {
    transform: translateY(-52px);
  }
`;

// BG is a fully encapsulating BG that removes focus once
// the mouse moves out of the hover card.  Sometimes the mouse moves
// too quickly for onMouseOut to be called on HoverCard;  this guarantees
// we hide the card.
const BG = styled.div`
  position: fixed;
  left: -100vw;
  top: -100vh;
  height: 200vh;
  width: 200vw;
  z-index: 1;
  background: transparent;
  pointer-events: none;
`;
