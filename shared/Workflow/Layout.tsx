import React from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import { Link, useHistory } from "react-router-dom";
import { useCurrentWorkspace, useWorkspaces } from "src/state/workspaces";
import Split from "react-split-pane";
import Toggle from "src/shared/Toggle";
import useToast from "src/shared/Toast";
import Button, { ButtonGroup } from "src/shared/Button";
import Loading from "src/shared/Loading";
import NewWorkflowPage from "./NewWorkflowPage";
import ActionPanel from "src/shared/Actions/Panel";
import Editor from "./Editor/Editor";
import TriggerPanel from "./TriggerPanel";
import Tag from "src/shared/Tag";
import CueEditor from "./CueEditor";
import PlayIcon from "src/shared/Icons/Play";
import Configuration from "./Configuration/Configuration";
import { useUpsertWorkflow } from "./queries";
import Footer from "./Footer";
import { Workflow, Action, useWorkflowContext } from "./state";
import { Action as BaseAction } from "src/types";

type Props = {
  mode: "new" | "edit";
  version?: number;
  workflowID?: string;
};

const Layout: React.FC<Props> = (props: Props) => {
  const w = useCurrentWorkspace();
  const { test } = useWorkspaces();
  const [state, dispatch] = useWorkflowContext();
  const history = useHistory();
  const upsert = useUpsertWorkflow();
  const { push } = useToast();
  const disabled =
    !state.config ||
    state.saving ||
    (state.workflow ? state.workflow.actions.length === 0 : false);

  // Save upserts a new workflow as a draft or as a published version.  The tool is
  // used so that running & debugging can add state to the redirect, in case they need to
  // show the redirect tool after redirecting to the new version.
  const save = async (publish: boolean, tool?: string) => {
    const { workflow } = state;
    if (disabled || !workflow) {
      push({
        type: "error",
        message: "You must configure your workflow before saving.",
      });
      return;
    }

    dispatch({ type: "saving", saving: true, dirty: true });
    const result = await upsert({
      workspaceID: w.id,
      config: state.config,
      workflowID: state.id,
      draft: !publish,
      version: props.version,
      promote: publish ? new Date().toISOString() : undefined,
    });
    dispatch({ type: "saving", saving: false, dirty: !!result.error });

    const data = result.data;
    if (result.error || !data) {
      push({
        type: "error",
        message: `Error saving workflow: ${result?.error?.message}`,
      });
      return result;
    }

    const published =
      data.version.validFrom &&
      new Date(data.version.validFrom) < new Date() &&
      (!data.version.validTo || new Date(data.version.validTo) > new Date());

    if (props.version !== data.version.version) {
      push({
        type: "success",
        message: `New workflow version ${
          published ? "published" : "drafted"
        }: ${data.version.version}`,
      });
      history.push(
        `/workflows/${data.workflow.id}/versions/${data.version.version}${
          tool ? "#" + tool : ""
        }`
      );
      return;
    }

    push({
      type: "success",
      message: published ? "Workflow draft published" : "Workflow draft edited",
    });

    return result;
  };

  React.useEffect(() => {
    dispatch({ type: "workflowID", workflowID: props.workflowID });
  }, [props.workflowID]);

  React.useEffect(() => {
    if (state.dirty) {
      return history.block(
        "Are you sure you want to leave? You have unsaved changes"
      );
    }
  }, [state.dirty]);

  const showNewWorkflowPage =
    !props.version && (!state.workflow || state.workflow.triggers.length === 0);

  return (
    <Wrapper css={[test && adjustTestHeightCSS]}>
      {state.parseError && <ErrorBanner>{state.parseError}</ErrorBanner>}

      {showNewWorkflowPage && !props.version && (
        <TriggerWrapper>
          <CanvasWrapper>
            <NewWorkflowPage
              visible
              onSubmit={(
                workflowName: string,
                type: "cron" | "event",
                value: string
              ) => {
                dispatch({
                  type: "new-workflow",
                  name: workflowName,
                  triggers: [{ [type]: value }],
                });
              }}
            />
          </CanvasWrapper>
        </TriggerWrapper>
      )}
      {!showNewWorkflowPage && (
        <>
          <Header>
            <Left>
              <input
                type="text"
                placeholder="(No name)"
                disabled={!state.workflow}
                value={state.workflow?.name}
                onChange={(e) => {
                  dispatch({
                    type: "edit-workflow",
                    property: "name",
                    value: e.target.value,
                  });
                }}
              />
              <div>
                {props.version && <Tag>Editing version {props.version}</Tag>}
                {props.workflowID && (
                  <Link to={`/workflows/${props.workflowID}`}>
                    Back to workflow dashboard
                  </Link>
                )}
              </div>
            </Left>
            <ButtonGroup right>
              <Button
                disabled={disabled || !state.dirty}
                onClick={() => save(false)}
              >
                Save draft
              </Button>

              <Button
                disabled={disabled}
                kind="primary"
                onClick={() => save(true)}
              >
                Publish workflow
              </Button>
            </ButtonGroup>
          </Header>
          <Toolbar />

          <div style={{ flex: 1, position: "relative" }}>
            <Split
              split="horizontal"
              primary="first"
              defaultSize={state.tool !== undefined ? "60%" : "100%"}
              size={state.tool === undefined ? "100%" : undefined}
            >
              <Grid>
                {!state.workflow ? (
                  <Loading />
                ) : (
                  <>
                    {state.mode === "graph" && (
                      <CanvasWrapper>
                        <Editor state={state} dispatch={dispatch} />
                      </CanvasWrapper>
                    )}
                    {state.mode === "code" && (
                      <CueEditor
                        wrapperCss={cueEditorStyles}
                        dispatch={dispatch}
                        state={state}
                      />
                    )}
                  </>
                )}

                <Sidebar
                  dispatch={dispatch}
                  mode={state.mode}
                  workflow={state.workflow}
                  actions={state.actions}
                />
                <Configuration defaultTab={state.configurationTab} />
              </Grid>
              <Footer
                save={save}
                workflowID={state?.id || ""}
                version={props.version || 0}
              />
            </Split>
          </div>
        </>
      )}
    </Wrapper>
  );
};

const Toolbar = () => {
  const [state, dispatch] = useWorkflowContext();

  const toggle = (tool: any) => {
    dispatch({ type: "setTool", tool: state.tool === tool ? undefined : tool });
  };

  return (
    <>
      <ToolbarWrapper>
        <button onClick={() => toggle("run")}>
          <PlayIcon size={14} />
          Run or debug
        </button>
      </ToolbarWrapper>
    </>
  );
};

type SidebarProps = {
  dispatch: (a: Action) => void;
  mode: "graph" | "code";
  workflow: Workflow | null;
  actions: BaseAction[];
};

const Sidebar = ({ dispatch, mode, workflow, actions }: SidebarProps) => {
  return (
    <SidebarContainer>
      <SidebarContent>
        <Mode
          onClick={() => {
            dispatch({
              type: "mode",
              mode: mode === "code" ? "graph" : "code",
            });
          }}
        >
          <span>Code</span>
          <Toggle
            checked={mode === "graph"}
            icons={false}
            onChange={(e: React.SyntheticEvent<HTMLInputElement>) => {
              dispatch({
                type: "mode",
                mode: (e.target as HTMLInputElement).checked ? "graph" : "code",
              });
            }}
          />
          <span>Graph</span>
        </Mode>
      </SidebarContent>
      <TriggerPanel triggers={workflow ? workflow.triggers : []} />

      <h4>Actions</h4>
      <ActionPanel
        actions={actions}
        onDragStart={(a) =>
          dispatch({
            type: "setDragAction",
            dragAction: a,
            moveAction: null,
          })
        }
      />
    </SidebarContainer>
  );
};

export default Layout;

const Wrapper = styled.div`
  display: flex;
  flex-direction: column;
  position: relative;

  /* Ensure the entire page doesnt scroll */
  overflow-y: hidden;

  .react-flow__controls {
    /* this avoids the save footer */
    position: fixed;
    bottom: 70px;
    left: 230px;
  }

  height: 100vh;
`;

const adjustTestHeightCSS = css`
  height: calc(100vh - 27px);
`;

const cueEditorStyles = css`
  /* 120px for page title, 57px for save footer */
  // height: calc(100vh - 57px);
  overflow: auto;

  > div {
    min-height: 100%;
  }
`;

const Header = styled.div`
  padding: 15px 20px;
  background: #fff;
  border-bottom: 1px solid #eee;
  height: 70px;
  display: flex;
  align-items: center;

  z-index: 6;
  top: 0;
  right: 0;
  width: 100%;
`;

const Left = styled.div`
  display: flex;
  align-items: center;

  input {
    border: 0;
    font-size: 18px;
    font-weight: bold;
  }

  > div {
    margin-left: 20px;
    min-width: 200px;
  }

  a {
    display: block;
    font-size: 11px;
    margin-top: 5px;
  }
`;

const ToolbarWrapper = styled.div`
  background: #fdfbf666;
  background: #fff;
  position: relative;
  border-bottom: 1px solid #92928122;
  height: 40px;
  display: flex;
  align-items: stretch;
  justify-content: flex-end;

  box-shadow: 0 0 20px rgba(0, 0, 0, 0.05);
  font-size: 12px;
  padding: 0 20px;

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

const TriggerWrapper = styled.div`
  flex: 1;
  display: grid;
`;

const Grid = styled.div`
  flex: 1;
  display: grid;
  overflow: auto;
  grid-template-columns: auto 400px;

  textarea {
    font-family: mono, monospace;
  }
`;

const CanvasWrapper = styled.div`
  position: relative;
  border-right: 1px solid #eee;
  overflow: auto;
`;

const ErrorBanner = styled.div`
  text-align: center;
  padding: 5px 40px;
  font-size: 12px;
  background: #cb2525;
  z-index: 25;
  position: absolute;

  // left + right menu bars
  width: calc(100vw - 400px - 220px);
  color: #fff;
`;

const SidebarContent = styled.div`
  padding: 0 20px;
`;

const SidebarContainer = styled.div`
  flex: 1;
  background: #fff;

  input {
    padding: 8px 12px;
  }

  small {
    opacity: 0.5;
  }

  h4.workflowName {
    margin-top: 30px;
  }

  h4 {
    margin: 40px 0 10px;
  }

  > h4 {
    padding-left: 20px;
  }

  overflow: auto;
`;

const Mode = styled.div`
  cursor: pointer;
  color: #999;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  margin: 20px 0 10px;

  .react-toggle-track {
    background: #999;
  }

  > div {
    transform: scale(0.6);
  }

  span:first-of-type {
    margin: 0 10px 0 0;
  }

  span:last-of-type {
    margin: 0 0 0 10px;
  }
`;
