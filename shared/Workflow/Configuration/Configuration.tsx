import React, { useEffect } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import Button, { ButtonGroup } from "src/shared/Button";
import { categoryIcon } from "src/shared/Actions/icons";
import { Confirm, ConfirmModal } from "src/shared/Confirm";
import { Tabs, Tab } from "src/shared/Tabs";
import Tag from "src/shared/Tag";
import { useWorkflowContext } from "../state";
import Content from "./Content";
import Callers from "./Callers";
import Data from "./Data";
import ActionNode from "../Editor/Nodes/ActionNode";
import { useConfigReducer, State } from "./reducer";
import SelectEventModal from "./SelectEventModal";
import { showAvailableActionData } from "../data";

type Props = { defaultTab?: "configuration" | "callers" };

// Configuration displays the side panel configuration
const Configuration: React.FC<Props> = (props) => {
  const [state, dispatch] = useWorkflowContext();

  // config state and dispatch.  terrible var names.
  const [cs, cd] = useConfigReducer({
    tab: props.defaultTab || "configuration",
  });
  const [confirm, setConfirm] = React.useState<Confirm | false>(false);

  const action =
    state.workflow &&
    state.workflow.actions.find((c) => c.clientID === state.configuring);

  const abstractAction = state.actions.find(
    (a) => action && a.dsn === action.dsn
  );

  const Icon = abstractAction && categoryIcon(abstractAction.category.name);

  // whenever the action changes we must reset the reducer.
  useEffect(() => {
    cd({
      type: "reset",
      tab: props.defaultTab || "configuration",
      incomingEdges:
        (action && state.incomingActionEdges[action.clientID.toString()]) || [],
      metadata: action && action.metadata ? action.metadata : {},
    });
  }, [action]);

  useEffect(() => {
    if (!action) {
      cd({ type: "availableData" });
      return;
    }
    // Whenever the eaction or example event changes recalculate the data that we
    // have available to this action.
    //
    // We need this to preview templates and to show data autocompletion.
    const data = showAvailableActionData(
      state,
      action.clientID,
      state.exampleEvent
    );
    data.fromActionID(action.clientID);
    cd({ type: "availableData", data });
  }, [action, state.exampleEvent]);

  const isEventTrigger = React.useMemo(() => {
    if (!state.workflow) {
      return false;
    }
    return !!state.workflow.triggers.find((t) => !!t.event);
  }, [state.workflow]);

  if (!abstractAction) {
    return null;
  }

  return (
    <>
      <Wrapper css={[action && showCSS]}>
        <Header>
          {action && abstractAction && (
            <>
              <ActionNode action={action} Icon={Icon as any} />
              <div>
                <p>
                  <strong>{abstractAction.name}</strong>
                </p>
                <p>{abstractAction.tagline}</p>
              </div>
            </>
          )}
        </Header>
        <HeaderTabs>
          <Tab
            active={cs.tab === "configuration"}
            onClick={() => cd({ type: "tab", tab: "configuration" })}
          >
            Configuration
          </Tab>
          <Tab
            active={cs.tab === "callers"}
            onClick={() => cd({ type: "tab", tab: "callers" })}
          >
            Callers
          </Tab>
          <Tab
            active={cs.tab === "data"}
            onClick={() => cd({ type: "tab", tab: "data" })}
          >
            Data
          </Tab>
        </HeaderTabs>
        {action && cs.tab === "configuration" && (
          <Toolbar>
            {isEventTrigger && (
              <button
                onClick={() => {
                  cd({ type: "selectEvent", to: true });
                }}
              >
                {state.exampleEvent ? (
                  <>
                    Using example event{" "}
                    <Tag
                      kinds={["identifier", "grey"]}
                      style={{ margin: "0 0 0 5px" }}
                    >
                      {state.exampleEvent.id}
                    </Tag>
                  </>
                ) : (
                  "Set example event"
                )}
              </button>
            )}
            <button
              onClick={() => {
                cd({ type: "previewTemplates", to: !cs.previewTemplates });
              }}
            >
              {cs.previewTemplates
                ? "Previewing templates"
                : "Preview templates"}
            </button>
          </Toolbar>
        )}

        <Banner cs={cs} isEventTrigger={isEventTrigger} />

        <Inner>
          {action && cs.tab === "configuration" && (
            <Content
              abstractAction={abstractAction}
              previewTemplates={!!cs.previewTemplates}
              action={{
                ...action,
                metadata: cs.dirty ? cs.metadata : action.metadata,
                name: cs.dirty ? cs.name || action.name : action.name,
              }}
              setName={(name: string) => cd({ type: "name", name })}
              setDirty={() => cd({ type: "dirty", dirty: true })}
              state={cs}
              onMetadataKeyChange={(key: string, value: string | number) => {
                cd({ type: "metadataKey", key, value });
              }}
              onMetadataChange={(metadata) => {
                cd({ type: "metadata", metadata });
              }}
            />
          )}
          {action && cs.tab === "callers" && (
            <Callers action={action} configState={cs} configDispatch={cd} />
          )}
          {action && cs.tab === "data" && <Data action={action} cd={cd} />}
        </Inner>

        <Footer className="footer">
          <ButtonGroup>
            <Button
              kind="danger"
              onClick={() => {
                setConfirm({
                  prompt: "Are you sure you want to remove this action?",
                  kind: "danger",
                  onClose: () => setConfirm(false),
                  onConfirm: () => {
                    action &&
                      dispatch({
                        type: "removeAction",
                        clientID: action.clientID,
                      });
                  },
                });
              }}
            >
              Remove action
            </Button>
          </ButtonGroup>
          <ButtonGroup right>
            <Button
              kind="primary"
              onClick={() => {
                dispatch({
                  type: "updateAction",
                  metadata: cs.metadata,
                  name: cs.name,
                  incomingEdges: cs.incomingEdges,
                });
                dispatch({
                  type: "configure",
                  clientID: null,
                });
              }}
            >
              Save action
            </Button>
          </ButtonGroup>
        </Footer>
      </Wrapper>

      <BG
        css={[!action && hiddenBG]}
        onClick={(_e: React.SyntheticEvent) => {
          if (cs.dirty) {
            setConfirm({
              prompt: "Do you want to keep this configuration?",
              kind: "primary",
              onClose: () => {
                setConfirm(false);
              },
              onCancel: () => {
                dispatch({ type: "configure", clientID: null });
                setConfirm(false);
              },
              onConfirm: () => {
                dispatch({
                  type: "updateAction",
                  metadata: cs.metadata,
                  name: cs.name,
                  incomingEdges: cs.incomingEdges,
                });
                dispatch({ type: "configure", clientID: null });
                setConfirm(false);
              },
            });
            return;
          }
          dispatch({ type: "configure", clientID: null });
        }}
      />

      {confirm && <ConfirmModal {...confirm} />}
      {cs.showSelectEvent && (
        <SelectEventModal
          onClose={() => cd({ type: "selectEvent", to: false })}
        />
      )}
    </>
  );
};

export default Configuration;

const Banner = ({
  cs,
  isEventTrigger,
}: {
  cs: State;
  isEventTrigger: boolean;
}) => {
  const [workflowState] = useWorkflowContext();

  const banner = (() => {
    if (cs.previewTemplates && !workflowState.exampleEvent && isEventTrigger) {
      return {
        msg:
          "Previewing templates with no example event chosen.  Event data will not be shown.",
        type: "info",
      };
    }
    if (cs.previewTemplates) {
      return {
        msg: "While previewing templates fields are not editable.",
        type: "info",
      };
    }
  })();

  if (!banner) {
    return null;
  }

  return <BannerWrapper>{banner.msg}</BannerWrapper>;
};

const showCSS = css`
  right: 0 !important;

  .footer {
    right: 0 !important;
  }
`;

const Header = styled.div`
  background: #fdfbf6;
  padding: 30px 40px 25px;

  display: flex;
  flex-direction: row;

  > div:last-of-type {
    margin: 5px 40px;
    color: #222631aa;
    font-size: 12px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
`;

const HeaderTabs = styled(Tabs)`
  padding: 0 40px;
  border-bottom: 1px solid #92928133;
  background: #fdfbf6;
  font-size: 14px;
`;

const Toolbar = styled.div`
  background: #fdfbf666;
  position: relative;
  border-bottom: 1px solid #92928122;
  height: 40px;
  display: flex;
  align-items: stretch;
  justify-content: flex-end;

  font-size: 12px;

  button {
    background: #fdfbf666;
    border: 0;
    padding: 3px 24px 0;
    color: #555;
    border-left: 1px solid #92928122;
  }
`;

const BannerWrapper = styled.div`
  padding: 6px 20px 6px;
  text-align: center;
  font-size: 12px;
  background: #f8ba96;
`;

const Inner = styled.div`
  padding: 50px 40px;
`;

const Wrapper = styled.div`
  position: fixed;
  top: 0;
  z-index: 50;
  height: 100%;
  width: 50vw;
  right: -100%;
  transition: all 0.3s;
  background: #fff;
  box-shadow: 0 5px 40px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.05);
  overflow-y: auto;
  overflow-x: hidden;
  padding-bottom: 78px;
`;

const BG = styled.div`
  position: fixed;
  background: rgba(0, 0, 0, 0.1);
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 49;
  opacity: 1;
  transition: all 0.3s;
`;

const hiddenBG = css`
  pointer-events: none;
  opacity: 0;
`;

const Footer = styled.div`
  border-top: 1px solid #eaeae9;
  position: fixed;
  bottom: 0;
  width: 50vw;
  right: -100%;
  padding: 20px 40px;
  background: #fff;
  transition: all 0.3s;
  display: flex;
  justify-content: space-between;
`;
