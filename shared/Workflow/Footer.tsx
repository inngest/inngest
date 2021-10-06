import React, { useRef, useEffect, useState, useMemo } from "react";
import ReactJSON from "react-json-view";
import { useQuery } from "urql";
import { DateTime } from "luxon";
import { Link } from "react-router-dom";
import styled from "@emotion/styled";
import Split from "react-split-pane";
import PlayIcon from "src/shared/Icons/Play";
import Button, { ButtonGroup } from "src/shared/Button";
import Cross from "src/shared/Icons/Cross";
import Tag from "src/shared/Tag";
import { titleCase } from "src/utils";
import useToast from "src/shared/Toast";
import { useCurrentWorkspace } from "src/state/workspaces";
import { EMPTY_EVENT } from "src/utils";
import { useWorkflowContext, State } from "./state";
import { useStartWorkflow } from "./queries";
import {
  runGQL,
  WorkflowRun,
  WorkflowEvent,
  useRunEvents,
  useActionEvent,
} from "src/scenes/Workflows/Run/queries";

import CodeEditor from "src/shared/CodeEditor";

type Props = {
  save: (publish: boolean, tool?: string) => Promise<any>;
  workflowID: string;
  version: number;
};

const Footer: React.FC<Props> = (props) => {
  const [state] = useWorkflowContext();

  switch (state.tool) {
    case "run":
      return <Run {...props} />;
    default:
      return null;
  }
};

const useTriggerState = (state: State) => {
  // If we have an example event within our state use that as the default value
  // for the trigger.  Else, use an empty event.
  const defValue = useMemo(() => {
    const evt = state.exampleEvent
      ? state.exampleEvent.event
      : JSON.stringify(EMPTY_EVENT);
    return JSON.stringify(JSON.parse(evt), undefined, "  ");
  }, []);
  return useState(defValue);
};

const usePollingLogs = (
  workspaceID: string,
  workflowID: string,
  runID: string
) => {
  const queries = useRef(0);
  const interval = useRef<any>(null);

  const [{ data }, execute] = useQuery<{
    workspace: { workflow: WorkflowRun };
  }>({
    query: runGQL,
    variables: { workspaceID, workflowID, runID },
    requestPolicy: "cache-and-network",
    pause: true,
  });

  const events = useRunEvents(data && data.workspace.workflow);
  const sorted = useMemo(() => events.reverse(), [events]);

  const poll = () => {
    execute();

    // Have two queries in the first second, then poll every 2.5 seconds,
    // then every 5 seconds.
    if (queries.current === 1) {
      window.clearInterval(interval.current);
      interval.current = window.setInterval(poll, 2500);
    }

    if (queries.current === 10) {
      window.clearInterval(interval.current);
      interval.current = window.setInterval(poll, 5000);
    }

    queries.current += 1;
  };

  useEffect(() => {
    if (interval.current) {
      window.clearInterval(interval.current);
    }
    if (!runID) return;
    interval.current = window.setInterval(poll, 500);
    return () => window.clearInterval(interval.current);
  }, [runID]);

  return sorted;
};

// Run allows you to run a new workflow and monitors output.
const Run: React.FC<Props> = ({ save, workflowID, version }) => {
  const w = useCurrentWorkspace();
  const [state, dispatch] = useWorkflowContext();
  const [, start] = useStartWorkflow();
  const { push } = useToast();

  // State
  const [runID, setRunID] = useState<string | null>(null);
  const [evt, setEvt] = useTriggerState(state);

  const events = usePollingLogs(w.id, workflowID, runID || "");

  // First we must save the draft.
  const onRun = async (debug: boolean) => {
    if (state.dirty) {
      await save(false, "run");
      return;
    }

    let event;
    try {
      event = JSON.parse(evt);
    } catch (e) {}

    setRunID("");

    // TODO: Create a new start of this workflow using the current evt
    // as a trigger.
    const result = await start({
      input: {
        workspaceID: w.id,
        workflowID,
        workflowVersion: version,
        baggage: {
          event,
        },
      },
      debug,
    });

    if (result.error || !result.data) {
      push({
        type: "error",
        message: `Error running workflow: ${result?.error?.message}`,
      });
      return;
    }

    push({
      type: "success",
      message: `Workflow started`,
    });
    setRunID(result.data.startWorkflowRun.id);
  };

  useEffect(() => {
    // If we have "run" as a hash prefix, run this immediately.  This came from
    // a previous save
    if (window.location.hash === "#run") {
      onRun(false);
    }
  }, []);

  if (!state.tool) return null;

  return (
    <FooterWrapper>
      <ToolbarWrapper>
        <div>
          <span>Run configuration</span>
        </div>
        <div>
          <button onClick={() => dispatch({ type: "setTool" })}>
            <Cross size={10} />
            Close
          </button>
          <button onClick={() => onRun(true)}>
            <PlayIcon outline={false} size={14} />
            Debug workflow
          </button>
          <button onClick={() => onRun(false)}>
            <PlayIcon outline={false} size={14} />
            Run workflow
          </button>
        </div>
      </ToolbarWrapper>
      <Split split="vertical" minSize={200} defaultSize="40%">
        <div>
          <Tag kind="grey">Trigger</Tag>
          <CodeEditor value={evt} onChange={setEvt} />
        </div>
        {!runID && (
          <None>
            <Tag kind="grey">Activity (newest first)</Tag>
            <p>
              No run yet.
              <br />
              Fill your event data then run the workflow to show logs.
            </p>
          </None>
        )}

        {runID && (
          <Logs>
            <Tag kind="grey">Activity (newest first)</Tag>

            {events.map((evt) => {
              if (evt.type === "action") {
                return (
                  <LogItem key={evt.data.id}>
                    <div>
                      <span>
                        {DateTime.fromISO(evt.data.createdAt).toLocal().toISO()}
                      </span>
                    </div>
                    <div>
                      <span>
                        {titleCase(evt.data.name)} action {evt.data.clientID} (
                        {evt.data.dsn})
                      </span>
                      <ActionLogs
                        workflowID={workflowID}
                        dsn={evt.data.dsn}
                        id={evt.data.id}
                      />
                    </div>
                  </LogItem>
                );
              }
              return (
                <LogItem key={evt.data.id}>
                  <div>
                    <span>
                      {DateTime.fromISO(evt.data.createdAt).toLocal().toISO()}
                    </span>
                  </div>
                  <div>
                    <span>{workflowEventName(evt.data.name)}</span>
                    {evt.data.name == "debugger_paused" ? (
                      <DebugActions evt={evt.data}></DebugActions>
                    ) : (
                      <>
                        {(evt.data.data || "").length > 0 && (
                          <div>
                            <ReactJSON
                              style={{
                                fontSize: 11,
                                fontFamily:
                                  'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
                                margin: "10px 0 0",
                              }}
                              displayObjectSize={false}
                              quotesOnKeys={false}
                              enableClipboard={false}
                              collapsed
                              src={JSON.parse(evt.data.data || "{}")}
                              name={"data"}
                            />
                          </div>
                        )}
                      </>
                    )}
                  </div>
                </LogItem>
              );
            })}

            {runID && (
              <LogItem>
                <div />
                <div>
                  Workflow initialized with run ID{" "}
                  <Link
                    to={`/workflows/${workflowID}/run/${runID}`}
                    target="_blank"
                  >
                    {runID}
                  </Link>
                </div>
              </LogItem>
            )}
          </Logs>
        )}
      </Split>
    </FooterWrapper>
  );
};

const DebugActions: React.FC<{ evt: WorkflowEvent }> = ({ evt }) => {
  const [show, setShow] = useState(true);
  const [state, dispatch] = useWorkflowContext();

  const data = JSON.parse(evt.data || "{}");
  const childAction = state.workflowActions[data.child_action_id.toString()];

  useEffect(() => {
    // On first render of this component, set the "paused at" workflow state
    // to the data's child ID.  This allows the workflow graph layout to show
    // debug nodes.
    dispatch({
      type: "addPausedOn",
      clientID: data.child_action_id.toString(),
      uuid: data.pause_id,
    });

    return () => {
      // remove all paused on UI as this is being hidden.
      dispatch({ type: "clearPausedOn" });
    };
  }, []);

  const onContinue = async () => {
    setShow(false);
    await fetch(
      `//${process.env.REACT_APP_INGEST_API_HOST}/continue/${data.pause_id}`,
      {
        method: "POST",
      }
    );
    dispatch({
      type: "removePausedOn",
      clientID: data.child_action_id.toString(),
    });
  };

  if (!show) {
    return null;
  }

  return (
    <DebugActionWrapper>
      <ButtonGroup style={{ margin: "12px 0 8px" }}>
        <Button onClick={onContinue} kind="primary">
          Continue from {childAction.name} (action #{data.child_action_id})
        </Button>
      </ButtonGroup>
    </DebugActionWrapper>
  );
};

const ActionLogs = ({
  workflowID,
  dsn,
  id,
}: {
  workflowID: string;
  dsn: string;
  id: string;
}) => {
  const w = useCurrentWorkspace();
  const [{ data }] = useActionEvent(w.id, workflowID, dsn, id);

  const actionEvent = data?.workspace?.workflow?.actionEvent;
  if (!actionEvent) {
    return null;
  }

  let evt = null;
  try {
    evt = JSON.parse(data?.workspace?.workflow?.actionEvent?.data || "");
  } catch (e) {}

  if (!evt) {
    return null;
  }

  return (
    <div>
      <ReactJSON
        style={{
          fontSize: 11,
          fontFamily:
            'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
          margin: "10px 0 0",
        }}
        displayObjectSize={false}
        quotesOnKeys={false}
        enableClipboard={false}
        collapsed
        src={evt}
        name={actionEvent.name === "started" ? "data" : "output"}
      />
    </div>
  );
};

const workflowEventName = (s: string) => {
  switch (s) {
    case "expression_evaluated":
      return titleCase(s);
    default:
      return `Workflow ${titleCase(s).toLowerCase()}`;
  }
};

export default Footer;

const FooterWrapper = styled.div`
  border-top: 1px solid #92928122;

  .Pane {
    display: flex;
    align-items: stretch;
  }

  .Pane > div {
    flex: 1;
    position: relative;
    display: flex;
    align-items: stretch;
  }

  .tag {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 2;
  }
`;

const Logs = styled.div`
  flex-direction: column;
  padding: 20px;
  font-family: monospace;
  font-size: 12px;
  overflow-y: auto;

  .node-ellipsis {
    font-size: 11px !important;
  }
`;

const LogItem = styled.div`
  display: grid;
  grid-template-columns: 220px 1fr;
  grid-gap: 10px;
  padding: 6px 0;
  border-bottom: 1px dotted #aaa;

  > div:first-of-type {
    opacity: 0.75;
  }

  > div > span {
    display: block;
  }
`;

const None = styled.div`
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 0 20px;

  > p {
    align-self: center;
    opacity: 0.4;
  }
`;

const ToolbarWrapper = styled.div`
  background: #fdfbf666;
  background: #fff;
  position: relative;
  border-bottom: 1px solid #92928122;
  height: 40px;
  padding: 0 20px;
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.05);

  font-size: 12px;

  > div {
    display: flex;
    align-items: stretch;
  }

  span {
    display: block;
    font-weight: 600;
    line-height: 40px;
  }

  button {
    background: #fdfbf666;
    border: 0;
    padding: 3px 24px 0;
    color: #666;
    border-left: 1px solid #92928122;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s;

    svg {
      margin: 0 6px 0 -2px;
      opacity: 0.7;
    }

    &:hover {
      background: #f7f3e8;
      color: #222;
      svg {
        opacity: 1;
      }
    }

    &:last-of-type {
      border-right: 1px solid #92928122;
    }
  }
`;

const DebugActionWrapper = styled.div`
  display: flex;
  flex-direction: column;
`;
