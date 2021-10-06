import styled from "@emotion/styled";
import { useEffect } from "react";
import { init } from "./parse";
import Editor from "./Editor/Editor";
import { WorkflowStateProvider, useWorkflowContext } from "./state";

type Props = {
  config: string;
}

export default function Wrapper(props: Props) {
  return (
    <WorkflowStateProvider>
      <Viewer {...props} />
    </WorkflowStateProvider>
  )
}

const Viewer = (props: Props) => {
  // Create new state for the editor.
  const [state, dispatch] = useWorkflowContext();

  const setup = async () => {
    await init();
    dispatch({ type: "config", config: props.config });
  }

  useEffect(() => { setup() }, [props.config]);

  return (
    <Layout className="editor">
      <Editor state={state} dispatch={dispatch} />
    </Layout>
  )
}

const Layout = styled.div`
  font-size: 13px;
  background: #fff;
  border: 1px solid #e8e8e6;
  border-radius: 2px;
  * { box-sizing: border-box; }
`;
