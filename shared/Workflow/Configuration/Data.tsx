import React from "react";
import styled from "@emotion/styled";
import ReactJSON from "react-json-view";
import { WorkflowAction, useWorkflowContext } from "../state";
import { useToast } from "src/shared/Toast";
import Tag from "src/shared/Tag";
import Button from "src/shared/Button";
import { Dispatch } from "./reducer";
import { showAvailableActionData } from "../data";

type Props = {
  action: WorkflowAction;
  cd: Dispatch;
};

// Data shows the data that is available to the given action from its parents.
const Data: React.FC<Props> = ({ action, cd }) => {
  const [state, dispatch] = useWorkflowContext();
  const { push } = useToast();

  const isEventTrigger = React.useMemo(() => {
    if (!state.workflow) {
      return false;
    }
    return !!state.workflow.triggers.find((t) => !!t.event);
  }, [state.workflow]);

  const data = React.useMemo(
    () => showAvailableActionData(state, action.clientID, state.exampleEvent),
    [action.clientID, state.exampleEvent]
  );

  const clipboard = (prefix?: string | undefined) => async (copy: any) => {
    // copy.namespace contains an array with our path.  for some reason the first
    // element is false, so we remove that.
    const data = copy.namespace.filter(Boolean).join(".");
    try {
      await navigator.clipboard.writeText((prefix || "") + data);
      push({ message: "Template path copied to clipboard", type: "default" });
    } catch (e) {}
  };

  return (
    <div>
      <Intro>
        <div>This action can use the following data:</div>

        {isEventTrigger && !state.exampleEvent && (
          <Button
            kind="link"
            onClick={() => cd({ type: "selectEvent", to: true })}
          >
            Select an example event to show real data
          </Button>
        )}

        {isEventTrigger && state.exampleEvent && (
          <EvtExample>
            Using example event{" "}
            <Tag kinds={["identifier", "grey"]} style={{ margin: "0 0 0 5px" }}>
              {state.exampleEvent.id}
            </Tag>
            .
            <Button
              kind="link"
              onClick={() => cd({ type: "selectEvent", to: true })}
              style={{ marginLeft: 5 }}
            >
              Change
            </Button>
            <span>or</span>
            <Button
              kind="link"
              onClick={() => dispatch({ type: "exampleEvent" })}
            >
              Clear
            </Button>
            .
          </EvtExample>
        )}
      </Intro>

      <ReactJSON
        quotesOnKeys={false}
        style={{
          fontSize: 12,
          fontFamily:
            'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
          marginBottom: 40,
        }}
        displayObjectSize={false}
        src={data.displayJSON}
        name={false}
        enableClipboard={clipboard()}
        collapsed
      />

      <Section>
        <h6>Event</h6>
        <small>
          Accessible in templating via{" "}
          <Tag
            keepCase
            kinds={["identifier", "grey"]}
          >{`{{ event.$field }}`}</Tag>
        </small>
        {data.event.hasMultiple && (
          <p>Note: this workflow is triggered by multiple events</p>
        )}
        <ReactJSON
          quotesOnKeys={false}
          style={{
            fontSize: 12,
            fontFamily:
              'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
          }}
          displayObjectSize={false}
          src={data.event.displayJSON}
          name={false}
          enableClipboard={clipboard("event.")}
        />
      </Section>

      {data.actions.length > 0 && (
        <Section>
          <h6>Parent actions</h6>
          {data.actions.map((a) => {
            return (
              <ActionWrapper key={a.clientID}>
                <p>{a.name}</p>
                <small>
                  Accessible in templating via{" "}
                  <Tag
                    keepCase
                    kinds={["identifier", "grey"]}
                  >{`{{ action.${a.clientID}.$field }}`}</Tag>
                </small>
                <ReactJSON
                  quotesOnKeys={false}
                  style={{
                    fontSize: 12,
                    fontFamily:
                      'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
                  }}
                  displayObjectSize={false}
                  src={a.data}
                  name={false}
                  enableClipboard={clipboard(`action.${a.clientID}.`)}
                />
              </ActionWrapper>
            );
          })}
        </Section>
      )}
    </div>
  );
};

/** Data styles **/

const Intro = styled.div`
  margin-bottom: 2rem;
  display: flex;
  justify-content: space-between;
`;

const EvtExample = styled.div`
  display: flex;
  flex: 1;
  justify-content: flex-end;

  button {
    margin: 0;
  }

  span {
    display: block;
    margin: 0 5px;
  }
`;

const Section = styled.div`
  margin: 0;
  padding: 2rem 0 2.5rem;
  border-top: 1px solid #eee;

  h6 {
    margin: 0 0 1.5rem;
    font-weight: 500;
    opacity: 0.4;
  }

  h6 + small {
    display: block;
    margin: -1.25rem 0 1.5rem;
  }
`;

const ActionWrapper = styled.div`
  & + & {
    margin-top: 30px;
  }

  p {
    display: block;
  }

  p + small {
    display: block;
    margin: 0 0 1.5rem;
  }
`;

export default Data;
