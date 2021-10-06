import { Action } from "src/types";

// TODO: Figure out a system for delegating the configuration
// UI to each particular action.  Either the action specifies
// the configuration fiels required, or we allow the action
// owners to specify their own UI to show here.  But we want
// to keep our branding, etc.  - so what do we do?
import { WorkflowAction } from "../state";
import { State as ConfigurationState } from "./reducer";
import Unknown from "./Content/Unknown";
import Webhook from "src/scenes/Workflows/Action/com.datos.comms.webhook/SidePanel";
import Email from "src/scenes/Workflows/Action/com.datos.comms.email/SidePanel";
import SegmentAdd from "src/scenes/Workflows/Action/com.datos.contacts.segmentadd/SidePanel";
import Slack from "src/scenes/Workflows/Action/com.inngest.comms.slack/SidePanel";
import AttributeUpdate from "src/scenes/Workflows/Action/com.inngest.contact.attributeupdate/SidePanel";
import CreateClickupTask from "src/scenes/Workflows/Action/clickup.inngest.com/CreateTask";
import UpdateClickupTask from "src/scenes/Workflows/Action/clickup.inngest.com/UpdateTask";
import InlineJS from "src/scenes/Workflows/Action/inngest.com.inlinejs/SidePanel";
import SFSearchObject from "src/scenes/Workflows/Action/salesforce.inngest.com/SearchObject";
import SFCreateObject from "src/scenes/Workflows/Action/salesforce.inngest.com/CreateObject";
import SFUpdateObject from "src/scenes/Workflows/Action/salesforce.inngest.com/UpdateObject";

export type P = {
  workspaceID: string;
  action: WorkflowAction;
  // TODO: Change this to ActionVersion
  abstractAction: Action;
  setDirty: () => void;
  onChange: (metadata: Object) => void;
  onMetadataKeyChange: (key: string, value: string | number) => void;
  state: ConfigurationState;

  previewTemplates: boolean;
};

const actionContents: { [dsn: string]: React.FC<P> } = {
  "com.inngest/http": Webhook,
  "com.inngest/email": Email,
  "com.inngest/segmentadd": SegmentAdd,
  "com.inngest/segmentremove": SegmentAdd,
  "slack/message": Slack,
  "com.inngest/attributeupdate": AttributeUpdate,

  "clickup.inngest.com/create-task": CreateClickupTask,
  "clickup.inngest.com/update-task": UpdateClickupTask,
  "inngest.com/inline-js": InlineJS,

  "example-sep2020/create-task": CreateClickupTask,
  "example-sep2020/update-task": UpdateClickupTask,

  "salesforce.inngest.com/search-object": SFSearchObject,
  "salesforce.inngest.com/create-object": SFCreateObject,
  "salesforce.inngest.com/update-object": SFUpdateObject,
};

export const getContent = (dsn: string): React.FC<P> => {
  return actionContents[dsn] || Unknown;
};
