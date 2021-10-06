import React from "react";
import styled from "@emotion/styled";
import Button, { ButtonGroup } from "src/shared/Button";
import { InputEditor } from "src/shared/InputEditor/InputEditor";
import {
  WorkflowAction,
  useWorkflowContext,
  State,
  WorkflowEdge,
  isEdgeMetadataIf,
  isEdgeMetadataAsync,
} from "../state";
import { State as ConfigState, Action as ConfigAction } from "./reducer";

type Props = {
  action: WorkflowAction;
  configState: ConfigState;
  configDispatch: (a: ConfigAction) => void;
};

// Callers represents how this action is going to be called via its incoming edges
const Callers: React.FC<Props> = ({ action, configState, configDispatch }) => {
  const { incomingEdges } = configState;
  const [state] = useWorkflowContext();
  const gqlAction = state.actions.find((a) => a.dsn === action.dsn);

  if (!gqlAction) {
    return null;
  }

  const conditions =
    incomingEdges.filter(
      (e) => e.metadata && Object.keys(e.metadata).length > 0
    ).length > 0;

  if (incomingEdges.length === 0) {
    // XXX: This will likely never be hit as we dont render the action in the workflow.
    return <h2>This action has no incoming edges and will not be called</h2>;
  }

  return (
    <div>
      <Intro>
        This action is called in {incomingEdges.length} place
        {incomingEdges.length > 1 && "s"}{" "}
        {conditions ? (
          <b>conditionally</b>
        ) : (
          <>
            <b>unconditionally</b> - it will always happen each time the parents
            finish
          </>
        )}
        .
      </Intro>

      {incomingEdges.map((e, n) => (
        <EdgeUI
          edge={e}
          action={action}
          state={state}
          key={e.incoming}
          n={n}
          onChange={(e) => {
            const copy = incomingEdges.slice();
            copy[n] = e;
            configDispatch({ type: "edges", incomingEdges: copy });
          }}
        />
      ))}
    </div>
  );
};

const EdgeUI: React.FC<{
  edge: WorkflowEdge;
  action: WorkflowAction;
  state: State;
  onChange: (e: WorkflowEdge) => void;
  n: number;
}> = (props) => {
  const { edge, state, onChange, action } = props;
  const kind = (() => {
    if (edge.metadata && edge.metadata.type === "async") {
      return "async";
    }
    if (isEdgeMetadataIf(edge.metadata)) {
      return "if";
    }
    return "always";
  })();

  return (
    <EdgeWrapper>
      <div>
        <label>
          After
          <select
            defaultValue={edge.outgoing === "trigger" ? "trigger" : undefined}
          >
            <option value="trigger">The trigger</option>
            {Object.values(state.workflowActions)
              .filter(
                (a) => a.clientID.toString() !== action.clientID.toString()
              )
              .map((a) => (
                <option
                  value={a.clientID}
                  key={a.clientID}
                  selected={edge.outgoing.toString() === a.clientID.toString()}
                >
                  {a.name}
                </option>
              ))}
          </select>
        </label>
        <label style={{ margin: 0 }}>
          Called
          <ButtonGroup packed>
            <Button
              kind={kind === "always" ? "primary" : "default"}
              onClick={() => {
                onChange({ ...edge, metadata: undefined });
              }}
            >
              Always
            </Button>
            <Button
              kind={kind === "if" ? "primary" : "default"}
              onClick={() => {
                onChange({ ...edge, metadata: { type: "edge", if: "" } });
              }}
            >
              Conditionally
            </Button>
            <Button
              kind={kind === "async" ? "primary" : "default"}
              onClick={() => {
                onChange({
                  ...edge,
                  metadata: {
                    type: "async",
                    if: "",
                    async: { event: "", ttl: "", match: "" },
                  },
                });
              }}
            >
              Asynchronously
            </Button>
          </ButtonGroup>
        </label>
      </div>

      {kind === "async" && isEdgeMetadataAsync(edge.metadata) && (
        <Details>
          <Conditions {...props} />

          <label>
            Event name
            <small>The event name we must wait for.</small>
            <input
              defaultValue={
                isEdgeMetadataAsync(edge.metadata)
                  ? edge.metadata.async.event
                  : ""
              }
              onChange={(e) => {
                onChange({
                  ...edge,
                  metadata: {
                    ...edge.metadata,
                    type: "async",
                    async: Object.assign(
                      { event: "", ttl: "", match: "" },
                      isEdgeMetadataAsync(edge.metadata)
                        ? edge.metadata.async
                        : {},
                      { event: e.target.value }
                    ),
                  },
                });
              }}
            />
          </label>
          <label>
            Event matching
            <small>
              An{" "}
              <a
                href="https://docs.inngest.com/docs/workflows/expressions"
                target="_blank"
              >
                expression
              </a>{" "}
              which the new event must match to be called which allows you to
              filter the incoming event. The new event has the key{" "}
              <code>async</code>.
            </small>
            <input
              defaultValue={
                isEdgeMetadataAsync(edge.metadata)
                  ? edge.metadata.async.match
                  : ""
              }
              onChange={(e) => {
                onChange({
                  ...edge,
                  metadata: {
                    ...edge.metadata,
                    type: "async",
                    async: Object.assign(
                      { event: "", ttl: "", match: "" },
                      isEdgeMetadataAsync(edge.metadata)
                        ? edge.metadata.async
                        : {},
                      { match: e.target.value }
                    ),
                  },
                });
              }}
            />
          </label>
          <label>
            Timeout
            <small>
              How long we can wait for the event, as a{" "}
              <a
                href="https://docs.inngest.com/docs/workflows/durations"
                target="_blank"
              >
                duration
              </a>
              .
            </small>
            <input
              defaultValue={
                isEdgeMetadataAsync(edge.metadata)
                  ? edge.metadata.async.ttl
                  : ""
              }
              onChange={(e) => {
                onChange({
                  ...edge,
                  metadata: {
                    ...edge.metadata,
                    type: "async",
                    async: Object.assign(
                      { event: "", ttl: "", match: "" },
                      isEdgeMetadataAsync(edge.metadata)
                        ? edge.metadata.async
                        : {},
                      { ttl: e.target.value }
                    ),
                  },
                });
              }}
            />
          </label>
        </Details>
      )}

      {kind === "if" && (
        <Details>
          <Conditions {...props} />
        </Details>
      )}
    </EdgeWrapper>
  );
};

const Conditions: React.FC<{
  edge: WorkflowEdge;
  action: WorkflowAction;
  state: State;
  onChange: (e: WorkflowEdge) => void;
  n: number;
}> = ({ edge, state, n, onChange, action }) => {
  return (
    <div>
      <label>
        Conditions
        <small>
          Write{" "}
          <a
            href="https://docs.inngest.com/docs/workflows/expressions"
            target="_blank"
          >
            expressions
          </a>{" "}
          which must evaluate to true for this to be called.
        </small>
        <InputEditor
          kind="textarea"
          value={isEdgeMetadataIf(edge.metadata) ? edge.metadata.if : ""}
          onChange={(value) => {
            onChange({
              ...edge,
              metadata: {
                ...edge.metadata,
                type: "edge",
                if: value,
              },
            });
          }}
        />
      </label>
    </div>
  );
};

const Intro = styled.p`
  margin-bottom: 2rem;
`;

const EdgeWrapper = styled.div`
  margin: 0;
  padding: 2rem 0 2.5rem;
  border-top: 1px solid #eee;

  div {
    display: flex;
    flex-direction: row;

    > label:first-of-type {
      flex: 1;
      margin-right: 20px;
    }

    > label:last-of-type {
      > div {
        padding: 10px 0 0;
      }
    }
  }

  h6 {
    margin: 0 0 1.5rem;
    font-weight: 500;
    opacity: 0.4;
  }
`;

const Details = styled.div`
  margin: 2rem 0 0;
  display: block !important;
`;

export default Callers;
