import React from "react";
import styled from "@emotion/styled";
import cronstrue from "cronstrue";
import { WorkflowTrigger } from "./state";
import Tag from "src/shared/Tag";
import Modal from "src/shared/Modal";
import Box from "src/shared/Box";
import Button, { ButtonGroup } from "src/shared/Button";
import { CRONTrigger } from "./NewWorkflowPage";
import { Workflow, Action, useWorkflowContext } from "./state";

type Props = {
  triggers: WorkflowTrigger[];
};

const TriggerPanel = ({ triggers }: Props) => {
  return (
    <>
      <h4>Triggers</h4>
      <List>
        {triggers.map((t) => (
          <div key={JSON.stringify(t)}>
            {t.cron && <Cron cron={t.cron} />}
            {t.event && <Event event={t.event} />}
          </div>
        ))}
      </List>
    </>
  );
};

export default TriggerPanel;

const Event = ({ event }: { event: string }) => {
  return (
    <Wrapper>
      <div>
        <Tag>Event</Tag>
      </div>
      <div>
        <p>{event}</p>
      </div>
    </Wrapper>
  );
};

const Cron = ({ cron }: { cron: string }) => {
  const [state, dispatch] = useWorkflowContext();
  const [modal, setModal] = React.useState(false);
  const [value, setValue] = React.useState(cron);

  const readable = React.useMemo(() => {
    try {
      return cronstrue.toString(value);
    } catch (e) {
      return "Invalid cron schedule";
    }
  }, [value]);

  // Find all other triggers except for this, so we can update the trigger
  // on change.
  const other = React.useMemo(() => {
    return state.workflow
      ? state.workflow.triggers.filter((t) => t.cron !== cron)
      : [];
  }, [cron]);

  const onClick = React.useCallback(() => {
    dispatch({ type: "triggers", triggers: other.concat([{ cron: value }]) });
    setModal(false);
  }, [value]);

  return (
    <Wrapper onClick={() => setModal(true)}>
      <div>
        <Tag kind="grey">Cron</Tag>
      </div>
      <div>
        <p>{readable}</p>
        {modal && (
          <Modal
            onClose={() => {
              setModal(false);
              // Also reset the value back to the original.
              setValue(cron);
            }}
          >
            <Box>
              <CRONTrigger
                value={value}
                onChange={(cron: string) => {
                  setValue(cron);
                }}
              />
              <ButtonGroup right style={{ marginTop: 20 }}>
                <Button kind="primary" onClick={onClick}>
                  Save
                </Button>
              </ButtonGroup>
            </Box>
          </Modal>
        )}
      </div>
    </Wrapper>
  );
};

const List = styled.div``;

const Wrapper = styled.div`
  display: grid;
  grid-template-columns: 60px auto;
  grid-gap: 20px;
  padding: 20px;
  box-shadow: 0;
  transition: all 0.2s;

  border-top: 1px solid #f5f5f5;

  > div:first-of-type {
    display: flex;
    justify-content: center;
  }

  &:last-of-type {
    border-bottom: 1px solid #f5f5f5;
  }

  svg {
    opacity: 0.6;
    margin: 0 10px 0 0;
  }

  p {
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  &:hover {
    background: #fff;
    cursor: pointer;
    box-shadow: 0 5px 40px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.05);
  }
`;
