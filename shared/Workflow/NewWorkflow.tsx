import React from "react";
import { WorkflowStateProvider, DefaultStateProps } from "./state";
import Layout from "./Layout";

type Props = DefaultStateProps;

// ContextWrapper is the entrypoint that provides the
const ContextWrapper: React.FC<Props> = (props) => (
  <WorkflowStateProvider {...props}>
    <Main />
  </WorkflowStateProvider>
);

export default ContextWrapper;

const Main = () => {
  return <Layout mode="new" />;
};
