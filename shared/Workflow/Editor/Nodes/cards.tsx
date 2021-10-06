// TODO: Figure out a system for delegating the configuration
// UI to each particular action.  Either the action specifies
// the configuration fiels required, or we allow the action
// owners to specify their own UI to show here.  But we want
// to keep our branding, etc.  - so what do we do?
//
// COPIED FROM shared/Workflow/Configuration/Content/consts
import React from "react";
import { WorkflowAction } from "../../state";

export type P = {
  action: WorkflowAction;
};

export const getHoverContent = (dsn: string): React.FC<P> | undefined => {
  return actionContents[dsn];
};

const actionContents: { [dsn: string]: React.FC<P> } = {
  "com.inngest/some-action": () => <p>Custom content</p>,
};

export const Blank = () => <div />;
