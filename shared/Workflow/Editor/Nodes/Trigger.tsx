// @ts-nocheck
import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";

import { baseCSS } from "./css";
import Button, { ButtonGroup } from "src/shared/Button";
import Tag from "src/shared/Tag";
import cronstrue from "cronstrue";
import {
  Handle,
  Position,
  NodeProps,
  useStoreActions,
} from "react-flow-renderer";
import { WorkflowTrigger } from "../../state";
import { nodeW, nodeH, newClientID } from "../consts";
import { WorkflowAction, useWorkflowContext, EdgeMetadata } from "../../state";

const Trigger = (props: NodeProps) => {
  const { data, selected } = props;
  const trigger = data.trigger as WorkflowTrigger;

  // Allow us to dyanmically choose which el is selected in react flow.
  // This is needed to manipulate the flow chart's z-indexes to show hover cards.
  //
  // By default, react-flow sets a style of `z-index: 3` to all nodes.
  // For our hover card to show over any other sibling nodes, we need to actually
  // selec this node in react-flow's state.
  const setSelectedElements = useStoreActions(
    (actions) => actions.setSelectedElements
  );

  const human = React.useMemo(() => {
    try {
      return trigger.cron && cronstrue.toString(trigger.cron);
    } catch (e) {
      return "Invalid cron schedule";
    }
  }, []);

  return (
    <>
      <Handle
        type="target"
        position={"top" as Position}
        style={{ opacity: 0, top: 20 }}
      />
      <Wrapper css={selected && hoverCardVisibleCSS}>
        {trigger.event && <Tag>Event trigger</Tag>}
        {trigger.cron && <ScheduledTag>Scheduled trigger</ScheduledTag>}
        <TriggerCard
          style={{ padding: 0 }}
          className=".inngest--node-card"
          onMouseOver={() => {
            setSelectedElements([
              {
                id: props.id,
                type: props.type,
              } as any,
            ]);
          }}
        >
          {trigger.event ? (
            <div>
              <p>{trigger.event}</p>
              <small>
                <b>Whenever</b> this event is sent
              </small>
            </div>
          ) : (
            <div>
              {" "}
              <pre>{trigger.cron}</pre>
              <small>
                <b>{human}</b> (UTC)
              </small>
            </div>
          )}
        </TriggerCard>
        <HoverCard className="inngest--hover-card">
          <HoverHeader onClick={() => {}}>
            <p>Workflow trigger</p>
          </HoverHeader>
          <HoverBody>
            <Actions visible={selected} />
          </HoverBody>
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
        style={{ opacity: 0, bottom: 20 }}
      />
    </>
  );
};

const Actions = ({ visible }: { visible: boolean }) => {
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
              outgoing: "trigger",
              incoming: state.workflow ? newClientID(state) : 1,
            },
          },
        ]),
      });
    },
    [state, dispatch]
  );

  return (
    <ActionWrapper css={[showEdges && showEdgesCSS]}>
      <div>
        <DefaultActions>
          <div right>
            <Button
              size="small"
              onClick={(e: React.SyntheticEvent) => {
                e.stopPropagation();
                addPlaceholder();
              }}
            >
              Add child
            </Button>
          </div>
        </DefaultActions>
      </div>
    </ActionWrapper>
  );
};

export default Trigger;

const ActionWrapper = styled.div`
  /* This shows only the default actions initially */
  height: 28px;

  &,
  > div {
    transition: all 0.3s;
  }
`;

const hoverCardVisibleCSS = css`
  .inngest--node-card {
    cursor: pointer;
    box-shadow: 0 3px 8px rgba(0, 0, 0, 0.03);
  }

  .inngest--hover-card {
    cursor: default;
    z-index: 2;
    opacity: 1;
    transform: scale(1);
    pointer-events: auto;
  }

  .inngest-hover-bg {
    pointer-events: auto;
  }
`;

const DefaultActions = styled.div`
  display: flex;
  justify-content: space-between;
  transition: all 0.3s;
`;

const TriggerCard = styled.div`
  ${baseCSS};
  padding: 12px 20px 8px;
  color: #fff;
  line-height: 1.5;
  position: relative;
  text-align: center;
  display: flex;

  flex-diection: column;
  justify-content: center;

  > div {
    padding-top: 2px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  pre {
    font-size: 12px;
  }

  small {
    font-size: 12px;
    opacity: 0.6;
    line-height: 1.1;
  }

  z-index: 3;
`;

const Wrapper = styled.div`
  position: relative;

  .tag {
    position: absolute;
    top: -10px;
    left: calc(50% - 56px);
    z-index: 6;
  }
`;

const ScheduledTag = styled(Tag)`
  position: absolute;
  top: -10px;
  left: calc(50% - 67px) !important;
  z-index: 6;
`;

const HoverCard = styled.div`
  display: none;
  width: 560px;
  max-height: 340px;
  position: absolute;
  left: -20px;
  top: -20px;
  border: 1px solid #eee;
  background: #fff;
  opacity: 0;
  transform: scale(0.97);
  // z-index: 1;
  pointer-events: none;
  box-shadow: rgb(0 0 0 / 30%) 0px 16px 70px;
  border-radius: 4px;

  transition: all 0.3s;
`;

const HoverHeader = styled.div`
  cursor: default;
  background: #fdfbf6;
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
  border-top: 1px solid #eee;
  padding: 14px 20px;
  background: #fdfbf666;
  height: auto;
  overflow: hidden;

  font-size: 14px;
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
