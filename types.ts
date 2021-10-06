import type monaco from "monaco-editor";

// eslint-disable-next-line
declare const process: {
  env: {
    REACT_APP_API_HOST: string;
  };
};

export type Usage = {
  period: string;
  asOf: string;
  total: number;
  data: Array<{
    slot: string;
    count: number;
  }>;
};

export type Pagination = {
  page: number;
  perPage: number;
  totalPages: number;
  totalItems: number;
};

export type Shape = {
  [key: string]: Typedef;
};

export type Typedef = {
  scalar: Scalar;
  compound: Compound;
  fields?: Shape;
};

export type Scalar = "unknown" | "int" | "float" | "string" | "boolean";
export type Compound = "none" | "array" | "map";

export type Action = {
  dsn: string;
  name?: string;
  tagline: string;
  category: {
    name: string;
  };

  latest: ActionVersion;
  versions?: ActionVersion[];

  settings?: Array<ActionSetting>;
};

export type ActionVersion = {
  dsn: string;
  name: string;
  version: number;
  createdAt: string;

  Settings: { [key: string]: any };
  WorkflowMetadata: WorkflowMetadata[];
  Response: {
    [key: string]: { name: "string"; type: Scalar; optional: boolean };
  };
  Edges: ActionEdge[];
};

export type ActionEdge = {
  name?: string;
  if?: string;
} & (EdgeTypeEdge | EdgeTypeAsync);

export type EdgeTypeEdge = {
  type: "edge";
};

export type EdgeTypeAsync = {
  type: "async";
  async: {
    ttl: string;
    event: string;
    match?: string;
  };
};

export type WorkflowMetadata = {
  name: string;
  expression?: string | null;
  required: boolean;
  type: string;
  Settings: { [key: string]: any };
  form: WorkflowForm;
};

export type WorkflowForm =
  | WorkflowFormTextarea
  | WorkflowFormInput
  | WorkflowFormDatetime
  | WorkflowFormSelect
  | WorkflowFormToggle;

type BaseFormProps = {
  title: string;
  hint?: string;
  placeholder?: string;
};

export type WorkflowFormTextarea = BaseFormProps & {
  type: "textarea";
};

export type WorkflowFormInput = BaseFormProps & {
  type: "input";
};

export type WorkflowFormDatetime = BaseFormProps & {
  type: "datetime";
};

export type WorkflowFormToggle = BaseFormProps & {
  type: "toggle";
};

export type WorkflowFormSelect = BaseFormProps & {
  type: "select";
  formselect: {
    choices: Array<{ name: string; value: string }>;
    eval?: string;
  };
};

export type ActionSetting = {
  id: string;
  actionDSN: string;
  category: string;
  name: string;
  data: string;
  createdAt: string;
  updatedAt: string;
};

export type ActionSecret = {
  id: string;
  actionDSN: string;
  service: string;
  name: string;
  dataPrefix: string;
  createdAt: string;
  updatedAt: string;
};

export type Workspace = {
  id: string;
  name: string;
  test: boolean;
  integrations: WorkspaceIntegration[];
  secrets?: ActionSecret[];
};

export type WorkspaceIntegration = {
  name: string;
  service: string;
  events: boolean;
  actions: boolean;
  webhookEndpoints: string[];
};

export type Contact = {
  id: string;

  predefinedAttributes: {
    external_id: string | null;
    first_name: string | null;
    last_name: string | null;
    email: string | null;
    phone: string | null;
    status: string | null;
    name: string | null;
  };

  unsubscribedFrom: CommsUnsubscribe[];
  createdAt: string;
  updatedAt: string;

  segments?: Array<{
    createdAt: string;
    segment: { id: string; name: string; type: string };
  }>;

  // TODO
  status?: string;
  lastActive: string;

  attributes: ContactAttribute[];
};

type JSON = string;

export type ContactAttribute = {
  id: string;
  name: string;
  value: JSON;
  validFrom: string;
  validTo?: string;
};

export type CommsUnsubscribe = {
  commType: string;
  service: "sms" | "email";
};

export type IntegrationEvent = {
  name: string;
  title: string;
  description: string;
  fields: { [field: string]: IntegrationField };
  expressions: Array<IntegrationExpression>;
};

export type IntegrationField = {
  field: string;
  title: string;
  description: string;
  type: string; // TODO: enum
  items?: { type: string };
  examples?: Array<any>;
};

export type IntegrationExpression = {
  name: string;
  expression: string;
};

export interface CompletionItemSubset {
  label: string | monaco.languages.CompletionItemLabel;
  kind: monaco.languages.CompletionItemKind;
  insertText: string;
  tags?: ReadonlyArray<monaco.languages.CompletionItemTag>;
  detail?: string;
  documentation?: string | monaco.IMarkdownString;
  sortText?: string;
  filterText?: string;
  preselect?: boolean;
}
