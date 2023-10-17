/* eslint-disable */
import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  BillingPeriod: unknown;
  Bytes: string;
  DSN: unknown;
  EdgeType: unknown;
  FilterType: unknown;
  IP: string;
  IngestSource: string;
  JSON: null | boolean | number | string | Record<string, unknown> | unknown[];
  Map: Record<string, unknown>;
  NullString: null | string;
  NullTime: null | string;
  Period: unknown;
  Role: unknown;
  Runtime: unknown;
  SchemaSource: string;
  SearchObject: unknown;
  SegmentType: unknown;
  Time: string;
  Timerange: unknown;
  ULID: string;
  UUID: string;
  Upload: unknown;
};

export type ApiKey = {
  __typename?: 'APIKey';
  author: User;
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
};

export type Account = {
  __typename?: 'Account';
  apiKeys: Array<Maybe<ApiKey>>;
  billingEmail: Scalars['String'];
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  identifier?: Maybe<AccountIdentifier>;
  name?: Maybe<Scalars['NullString']>;
  paymentIntents: Array<PaymentIntent>;
  paymentMethods?: Maybe<Array<PaymentMethod>>;
  plan?: Maybe<BillingPlan>;
  search: SearchResults;
  setting?: Maybe<AccountSetting>;
  settings?: Maybe<Array<AccountSetting>>;
  status: Scalars['String'];
  subscription?: Maybe<BillingSubscription>;
  updatedAt: Scalars['Time'];
  users: Array<User>;
};


export type AccountSearchArgs = {
  opts: SearchInput;
};


export type AccountSettingArgs = {
  name: Scalars['String'];
};

export type AccountIdentifier = {
  __typename?: 'AccountIdentifier';
  account: Account;
  accountID: Scalars['ID'];
  domain?: Maybe<Scalars['NullString']>;
  dsnPrefix?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  verifiedAt?: Maybe<Scalars['Time']>;
};

export type AccountIdentifierInput = {
  dsnPrefix?: InputMaybe<Scalars['String']>;
};

export type AccountSetting = {
  __typename?: 'AccountSetting';
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  name: Scalars['String'];
  updatedAt: Scalars['Time'];
  value: Scalars['String'];
};

export type Action = {
  __typename?: 'Action';
  accountID?: Maybe<Scalars['ID']>;
  aliases?: Maybe<Array<Scalars['String']>>;
  category: ActionCategory;
  dsn: Scalars['DSN'];
  latest: ActionVersion;
  name: Scalars['String'];
  secrets: Array<ActionSecret>;
  settings: Array<ActionSetting>;
  tagline: Scalars['String'];
  version?: Maybe<ActionVersion>;
};


export type ActionSecretsArgs = {
  workspaceID: Scalars['ID'];
};


export type ActionSettingsArgs = {
  category?: InputMaybe<Scalars['String']>;
  workspaceID: Scalars['ID'];
};


export type ActionVersionArgs = {
  major?: InputMaybe<Scalars['Int']>;
  minor?: InputMaybe<Scalars['Int']>;
};

export type ActionCategory = {
  __typename?: 'ActionCategory';
  actions: Array<Action>;
  name: Scalars['String'];
};

export type ActionFilter = {
  category?: InputMaybe<Scalars['String']>;
  dsn?: InputMaybe<Scalars['String']>;
  excludePublic?: InputMaybe<Scalars['Boolean']>;
};

export type ActionSecret = {
  __typename?: 'ActionSecret';
  actionDSN: Scalars['DSN'];
  createdAt: Scalars['Time'];
  dataPrefix: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  service: Scalars['String'];
  updatedAt: Scalars['Time'];
};

export type ActionSetting = {
  __typename?: 'ActionSetting';
  actionDSN: Scalars['DSN'];
  category: Scalars['String'];
  createdAt: Scalars['Time'];
  data: Scalars['JSON'];
  id: Scalars['ID'];
  name: Scalars['String'];
  updatedAt: Scalars['Time'];
};

export type ActionVersion = {
  __typename?: 'ActionVersion';
  Edges?: Maybe<Array<Edge>>;
  Response?: Maybe<Scalars['Map']>;
  Settings?: Maybe<Scalars['Map']>;
  WorkflowMetadata: Array<Maybe<ActionWorkflowMetadata>>;
  config: Scalars['String'];
  createdAt: Scalars['Time'];
  dsn: Scalars['DSN'];
  imageSha256?: Maybe<Scalars['String']>;
  name: Scalars['String'];
  runtime: Scalars['Runtime'];
  runtimeData: Scalars['Map'];
  usage: Usage;
  validFrom?: Maybe<Scalars['Time']>;
  validTo?: Maybe<Scalars['Time']>;
  versionMajor: Scalars['Int'];
  versionMinor: Scalars['Int'];
};


export type ActionVersionUsageArgs = {
  opts?: InputMaybe<UsageInput>;
  workspaceID: Scalars['ID'];
};

export type ActionVersionQualifier = {
  dsn: Scalars['DSN'];
  versionMajor: Scalars['Int'];
  versionMinor: Scalars['Int'];
};

export type ActionWorkflowMetadata = {
  __typename?: 'ActionWorkflowMetadata';
  expression?: Maybe<Scalars['String']>;
  form?: Maybe<Scalars['Map']>;
  name: Scalars['String'];
  required: Scalars['Boolean'];
  type: Scalars['String'];
};

export type Alert = {
  __typename?: 'Alert';
  workflow: Workflow;
  workflowID: Scalars['String'];
};

export type ArchiveWorkflowInput = {
  archive: Scalars['Boolean'];
  workflowID: Scalars['ID'];
};

export type ArchivedEvent = {
  __typename?: 'ArchivedEvent';
  contact?: Maybe<Contact>;
  contactID?: Maybe<Scalars['ID']>;
  event: Scalars['Bytes'];
  eventModel: Event;
  eventVersion: EventType;
  functionRuns: Array<FunctionRun>;
  id: Scalars['ULID'];
  ingestSourceID?: Maybe<Scalars['ID']>;
  name: Scalars['String'];
  occurredAt: Scalars['Time'];
  receivedAt: Scalars['Time'];
  source?: Maybe<IngestKey>;
  version: Scalars['String'];
};

export type AsyncEdge = {
  __typename?: 'AsyncEdge';
  event: Scalars['String'];
  match?: Maybe<Scalars['String']>;
  ttl: Scalars['String'];
};

export type BillingPlan = {
  __typename?: 'BillingPlan';
  amount: Scalars['Int'];
  billingPeriod: Scalars['BillingPeriod'];
  features: Scalars['Map'];
  id: Scalars['ID'];
  name: Scalars['String'];
};

export type BillingSubscription = {
  __typename?: 'BillingSubscription';
  nextInvoiceDate: Scalars['Time'];
};

export type CommsUnsubscribe = {
  __typename?: 'CommsUnsubscribe';
  commType: Scalars['String'];
  contactID: Scalars['ID'];
  id: Scalars['ID'];
  service: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type Communication = {
  __typename?: 'Communication';
  body?: Maybe<Scalars['NullString']>;
  bodyFile?: Maybe<File>;
  bodyFileID?: Maybe<Scalars['ID']>;
  category: Scalars['String'];
  clientID?: Maybe<Scalars['Int']>;
  commsType: Scalars['String'];
  contactID: Scalars['ID'];
  createdAt: Scalars['Time'];
  currentStatus: CommunicationStatus;
  id: Scalars['ID'];
  recipient: Scalars['String'];
  statuses: Array<CommunicationStatus>;
  title: Scalars['String'];
  workflow?: Maybe<Workflow>;
  workflowID?: Maybe<Scalars['ID']>;
  workflowVersion?: Maybe<Scalars['Int']>;
};

export type CommunicationStatus = {
  __typename?: 'CommunicationStatus';
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  status: Scalars['String'];
};

export type Contact = {
  __typename?: 'Contact';
  attributes: Array<Maybe<ContactAttribute>>;
  createdAt: Scalars['Time'];
  externalID?: Maybe<Scalars['NullString']>;
  id: Scalars['ID'];
  predefinedAttributes: Scalars['Map'];
  unsubscribed: Scalars['Boolean'];
  updatedAt: Scalars['Time'];
  workspaceID: Scalars['ID'];
};

export type ContactAttribute = {
  __typename?: 'ContactAttribute';
  contactID: Scalars['ID'];
  id: Scalars['ID'];
  name: Scalars['String'];
  validFrom: Scalars['Time'];
  validTo?: Maybe<Scalars['NullTime']>;
  value: Scalars['Bytes'];
};

export type ContactFilter = {
  attributes?: InputMaybe<Scalars['Map']>;
  email?: InputMaybe<Scalars['String']>;
  externalID?: InputMaybe<Scalars['String']>;
  phone?: InputMaybe<Scalars['String']>;
};

export type ContactSegment = {
  __typename?: 'ContactSegment';
  contact: Contact;
  contactID: Scalars['ID'];
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  segment: Segment;
  segmentID: Scalars['ID'];
};

export type ContactStats = {
  __typename?: 'ContactStats';
  total: Scalars['Int'];
  usage: Usage;
  workspaceID: Scalars['ID'];
};


export type ContactStatsUsageArgs = {
  opts?: InputMaybe<UsageInput>;
};

export type CreateStripeSubscriptionResponse = {
  __typename?: 'CreateStripeSubscriptionResponse';
  clientSecret: Scalars['String'];
  message: Scalars['String'];
};

export type CreateVercelAppInput = {
  path?: InputMaybe<Scalars['String']>;
  projectID: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type CreateVercelAppResponse = {
  __typename?: 'CreateVercelAppResponse';
  success: Scalars['Boolean'];
};

export type DeleteActionSecret = {
  secretID: Scalars['ID'];
  workspaceID: Scalars['ID'];
};

export type DeleteIngestKey = {
  id: Scalars['ID'];
  workspaceID: Scalars['ID'];
};

export type DeleteResponse = {
  __typename?: 'DeleteResponse';
  ids: Array<Scalars['ID']>;
};

export type Deploy = {
  __typename?: 'Deploy';
  appName: Scalars['String'];
  authorID?: Maybe<Scalars['UUID']>;
  checksum: Scalars['String'];
  createdAt: Scalars['Time'];
  deployedFunctions: Array<Workflow>;
  error?: Maybe<Scalars['String']>;
  framework?: Maybe<Scalars['String']>;
  functionCount?: Maybe<Scalars['Int']>;
  id: Scalars['UUID'];
  metadata: Scalars['Map'];
  prevFunctionCount?: Maybe<Scalars['Int']>;
  removedFunctions: Array<Workflow>;
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  status: Scalars['String'];
  url?: Maybe<Scalars['String']>;
  workspaceID: Scalars['UUID'];
};

export type Edge = {
  __typename?: 'Edge';
  async?: Maybe<AsyncEdge>;
  if: Scalars['String'];
  name: Scalars['String'];
  type: Scalars['EdgeType'];
};

export type EditWorkflowInput = {
  description?: InputMaybe<Scalars['String']>;
  disable?: InputMaybe<Scalars['Time']>;
  promote?: InputMaybe<Scalars['Time']>;
  version: Scalars['Int'];
  workflowID: Scalars['ID'];
};

export enum EnvironmentType {
  BranchChild = 'BRANCH_CHILD',
  BranchParent = 'BRANCH_PARENT',
  Production = 'PRODUCTION',
  Test = 'TEST'
}

export type Event = {
  __typename?: 'Event';
  description?: Maybe<Scalars['String']>;
  events?: Maybe<EventConnection>;
  firstSeen?: Maybe<Scalars['Time']>;
  integrationName?: Maybe<Scalars['String']>;
  name: Scalars['String'];
  recent: Array<ArchivedEvent>;
  schemaSource?: Maybe<Scalars['SchemaSource']>;
  usage: Usage;
  versionCount: Scalars['Int'];
  versions: Array<Maybe<EventType>>;
  workflows: Array<Workflow>;
  workspaceID?: Maybe<Scalars['UUID']>;
};


export type EventEventsArgs = {
  after?: InputMaybe<Scalars['String']>;
  filter: EventsFilter;
  first?: Scalars['Int'];
};


export type EventRecentArgs = {
  count?: InputMaybe<Scalars['Int']>;
};


export type EventUsageArgs = {
  opts?: InputMaybe<UsageInput>;
};


export type EventVersionsArgs = {
  versions?: InputMaybe<Array<Scalars['String']>>;
};

export type EventConnection = {
  __typename?: 'EventConnection';
  edges?: Maybe<Array<Maybe<EventEdge>>>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type EventEdge = {
  __typename?: 'EventEdge';
  cursor: Scalars['String'];
  node: ArchivedEvent;
};

export type EventQuery = {
  name?: InputMaybe<Scalars['String']>;
  prefix?: InputMaybe<Scalars['String']>;
  schemaSource?: InputMaybe<Scalars['SchemaSource']>;
  workspaceID?: InputMaybe<Scalars['ID']>;
};

export type EventType = {
  __typename?: 'EventType';
  createdAt?: Maybe<Scalars['Time']>;
  cueType: Scalars['String'];
  id: Scalars['ID'];
  jsonSchema: Scalars['Map'];
  name: Scalars['String'];
  typescript: Scalars['String'];
  updatedAt?: Maybe<Scalars['Time']>;
  version: Scalars['String'];
};

export type EventsFilter = {
  lowerTime: Scalars['Time'];
};

export type File = {
  __typename?: 'File';
  createdAt: Scalars['Time'];
  description?: Maybe<Scalars['NullString']>;
  filename: Scalars['String'];
  filetype: Scalars['String'];
  id: Scalars['ID'];
  presignedURL: Scalars['String'];
  private: Scalars['Boolean'];
  sha256: Scalars['String'];
  size: Scalars['Int'];
};

export type FilterList = {
  __typename?: 'FilterList';
  events?: Maybe<Array<Scalars['String']>>;
  ips?: Maybe<Array<Scalars['IP']>>;
  type?: Maybe<Scalars['FilterType']>;
};

export type FilterListInput = {
  events?: InputMaybe<Array<Scalars['String']>>;
  ips?: InputMaybe<Array<Scalars['IP']>>;
  type?: InputMaybe<Scalars['FilterType']>;
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  accountID: Scalars['UUID'];
  batchID?: Maybe<Scalars['ULID']>;
  canRerun: Scalars['Boolean'];
  contactID?: Maybe<Scalars['UUID']>;
  endedAt?: Maybe<Scalars['Time']>;
  event?: Maybe<ArchivedEvent>;
  eventID?: Maybe<Scalars['ULID']>;
  events: Array<ArchivedEvent>;
  function: Workflow;
  history: Array<RunHistoryItem>;
  historyItemOutput?: Maybe<Scalars['String']>;
  id: Scalars['ULID'];
  originalRunID?: Maybe<Scalars['ULID']>;
  output?: Maybe<Scalars['Bytes']>;
  startedAt: Scalars['Time'];
  status: FunctionRunStatus;
  timeline?: Maybe<Array<RunTimeline>>;
  workflowID: Scalars['UUID'];
  workflowVersion: WorkflowVersion;
  workflowVersionInt: Scalars['Int'];
  workspaceID: Scalars['UUID'];
};


export type FunctionRunHistoryItemOutputArgs = {
  id: Scalars['ULID'];
};

export enum FunctionRunStatus {
  /** The function run has been cancelled. */
  Cancelled = 'CANCELLED',
  /** The function run has completed. */
  Completed = 'COMPLETED',
  /** The function run has failed. */
  Failed = 'FAILED',
  /** The function run is currently running. */
  Running = 'RUNNING'
}

export enum FunctionRunTimeField {
  EndedAt = 'ENDED_AT',
  Mixed = 'MIXED',
  StartedAt = 'STARTED_AT'
}

export enum HistoryStepType {
  Run = 'Run',
  Send = 'Send',
  Sleep = 'Sleep',
  Wait = 'Wait'
}

export enum HistoryType {
  FunctionCancelled = 'FunctionCancelled',
  FunctionCompleted = 'FunctionCompleted',
  FunctionFailed = 'FunctionFailed',
  FunctionScheduled = 'FunctionScheduled',
  FunctionStarted = 'FunctionStarted',
  FunctionStatusUpdated = 'FunctionStatusUpdated',
  None = 'None',
  StepCompleted = 'StepCompleted',
  StepErrored = 'StepErrored',
  StepFailed = 'StepFailed',
  StepScheduled = 'StepScheduled',
  StepSleeping = 'StepSleeping',
  StepStarted = 'StepStarted',
  StepWaiting = 'StepWaiting'
}

export type IngestKey = {
  __typename?: 'IngestKey';
  author: User;
  createdAt: Scalars['Time'];
  filter: FilterList;
  id: Scalars['ID'];
  metadata?: Maybe<Scalars['Map']>;
  name: Scalars['NullString'];
  presharedKey: Scalars['String'];
  source: Scalars['IngestSource'];
  url?: Maybe<Scalars['String']>;
};

export type IngestKeyFilter = {
  name?: InputMaybe<Scalars['String']>;
  source?: InputMaybe<Scalars['String']>;
};

export type InsertActionSetting = {
  category: Scalars['String'];
  data: Scalars['String'];
  dsn: Scalars['DSN'];
  name: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type MetricsData = {
  __typename?: 'MetricsData';
  bucket: Scalars['Time'];
  value: Scalars['Float'];
};

export type MetricsRequest = {
  from: Scalars['Time'];
  name: Scalars['String'];
  to: Scalars['Time'];
};

export type MetricsResponse = {
  __typename?: 'MetricsResponse';
  data: Array<MetricsData>;
  from: Scalars['Time'];
  granularity: Scalars['String'];
  to: Scalars['Time'];
};

export type Mutation = {
  __typename?: 'Mutation';
  archiveEnvironment: Workspace;
  archiveEvent?: Maybe<Event>;
  archiveWorkflow?: Maybe<WorkflowResponse>;
  createAPIKey: VisibleApiKey;
  createAction: Action;
  createIngestKey: IngestKey;
  createIntegrationWebhook: IngestKey;
  createSegment: Segment;
  createStripeSubscription: CreateStripeSubscriptionResponse;
  createUser?: Maybe<User>;
  createVercelApp?: Maybe<CreateVercelAppResponse>;
  createWorkflow?: Maybe<WorkflowVersionResponse>;
  createWorkspace: Array<Maybe<Workspace>>;
  deleteActionSecret?: Maybe<DeleteResponse>;
  deleteIngestKey?: Maybe<DeleteResponse>;
  deleteUser: Scalars['ID'];
  disableEnvironmentAutoArchive: Workspace;
  editSegment: Segment;
  editWorkflow?: Maybe<WorkflowVersionResponse>;
  enableEnvironmentAutoArchive: Workspace;
  insertActionSetting: ActionSetting;
  register: RegisterResponse;
  removeVercelApp?: Maybe<RemoveVercelAppResponse>;
  retryWorkflowRun?: Maybe<StartWorkflowResponse>;
  startWorkflowRun?: Maybe<StartWorkflowResponse>;
  unarchiveEnvironment: Workspace;
  updateAccount: Account;
  updateAccountIdentifier: AccountIdentifier;
  updateAccountSetting?: Maybe<AccountSetting>;
  updateActionSetting: ActionSetting;
  updateActionVersion: ActionVersion;
  updateContact: Contact;
  updateIngestKey: IngestKey;
  updatePaymentMethod?: Maybe<Array<PaymentMethod>>;
  updatePlan: Account;
  updateUser?: Maybe<User>;
  updateVercelApp?: Maybe<UpdateVercelAppResponse>;
  upsertActionSecret: ActionSecret;
  upsertActionSetting: ActionSetting;
  upsertWorkflow?: Maybe<WorkflowVersionResponse>;
};


export type MutationArchiveEnvironmentArgs = {
  id: Scalars['ID'];
};


export type MutationArchiveEventArgs = {
  name: Scalars['String'];
  workspaceID: Scalars['ID'];
};


export type MutationArchiveWorkflowArgs = {
  input: ArchiveWorkflowInput;
};


export type MutationCreateApiKeyArgs = {
  name?: InputMaybe<Scalars['String']>;
};


export type MutationCreateActionArgs = {
  config: Scalars['String'];
};


export type MutationCreateIngestKeyArgs = {
  input: NewIngestKey;
};


export type MutationCreateIntegrationWebhookArgs = {
  service: Scalars['String'];
  workspaceID: Scalars['ID'];
};


export type MutationCreateSegmentArgs = {
  opts: SegmentOpts;
  workspaceID: Scalars['ID'];
};


export type MutationCreateStripeSubscriptionArgs = {
  input: StripeSubscriptionInput;
};


export type MutationCreateUserArgs = {
  input: NewUser;
};


export type MutationCreateVercelAppArgs = {
  input: CreateVercelAppInput;
};


export type MutationCreateWorkflowArgs = {
  input: NewWorkflowInput;
};


export type MutationCreateWorkspaceArgs = {
  input: NewWorkspaceInput;
};


export type MutationDeleteActionSecretArgs = {
  input: DeleteActionSecret;
};


export type MutationDeleteIngestKeyArgs = {
  input: DeleteIngestKey;
};


export type MutationDeleteUserArgs = {
  id: Scalars['ID'];
};


export type MutationDisableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID'];
};


export type MutationEditSegmentArgs = {
  id: Scalars['ID'];
  opts: SegmentOpts;
};


export type MutationEditWorkflowArgs = {
  input: EditWorkflowInput;
};


export type MutationEnableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID'];
};


export type MutationInsertActionSettingArgs = {
  input: InsertActionSetting;
};


export type MutationRegisterArgs = {
  input: RegisterInput;
};


export type MutationRemoveVercelAppArgs = {
  input: RemoveVercelAppInput;
};


export type MutationRetryWorkflowRunArgs = {
  input: StartWorkflowInput;
  workflowRunID: Scalars['ULID'];
};


export type MutationStartWorkflowRunArgs = {
  debug: Scalars['Boolean'];
  input: StartWorkflowInput;
};


export type MutationUnarchiveEnvironmentArgs = {
  id: Scalars['ID'];
};


export type MutationUpdateAccountArgs = {
  input: UpdateAccount;
};


export type MutationUpdateAccountIdentifierArgs = {
  input: AccountIdentifierInput;
};


export type MutationUpdateAccountSettingArgs = {
  name: Scalars['String'];
  value: Scalars['String'];
};


export type MutationUpdateActionSettingArgs = {
  input: UpdateActionSetting;
};


export type MutationUpdateActionVersionArgs = {
  enabled: Scalars['Boolean'];
  version: ActionVersionQualifier;
};


export type MutationUpdateContactArgs = {
  attributes: Scalars['Map'];
  id: Scalars['ID'];
};


export type MutationUpdateIngestKeyArgs = {
  id: Scalars['ID'];
  input: UpdateIngestKey;
};


export type MutationUpdatePaymentMethodArgs = {
  token: Scalars['String'];
};


export type MutationUpdatePlanArgs = {
  to: Scalars['ID'];
};


export type MutationUpdateUserArgs = {
  input: UpdateUser;
};


export type MutationUpdateVercelAppArgs = {
  input: UpdateVercelAppInput;
};


export type MutationUpsertActionSecretArgs = {
  input: UpsertActionSecret;
};


export type MutationUpsertActionSettingArgs = {
  input: UpsertActionSetting;
};


export type MutationUpsertWorkflowArgs = {
  input: UpsertWorkflowInput;
};

export type NewIngestKey = {
  filterList?: InputMaybe<FilterListInput>;
  metadata?: InputMaybe<Scalars['Map']>;
  name: Scalars['String'];
  source: Scalars['IngestSource'];
  workspaceID: Scalars['ID'];
};

export type NewUser = {
  email: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type NewWorkflowInput = {
  config: Scalars['String'];
  description?: InputMaybe<Scalars['String']>;
  draft: Scalars['Boolean'];
  workspaceID: Scalars['ID'];
};

export type NewWorkspaceInput = {
  name: Scalars['String'];
};

/** The pagination information in a connection. */
export type PageInfo = {
  __typename?: 'PageInfo';
  /** When paginating forward, the cursor to query the next page. */
  endCursor?: Maybe<Scalars['String']>;
  /** Indicates if there are any pages subsequent to the current page. */
  hasNextPage: Scalars['Boolean'];
  /** Indicates if there are any pages prior to the current page. */
  hasPreviousPage: Scalars['Boolean'];
  /** When paginating backward, the cursor to query the previous page. */
  startCursor?: Maybe<Scalars['String']>;
};

export type PageResults = {
  __typename?: 'PageResults';
  cursor?: Maybe<Scalars['String']>;
  page: Scalars['Int'];
  perPage: Scalars['Int'];
  totalItems?: Maybe<Scalars['Int']>;
  totalPages?: Maybe<Scalars['Int']>;
};

export type PaginatedContactSegments = {
  __typename?: 'PaginatedContactSegments';
  data: Array<ContactSegment>;
  page: PageResults;
};

export type PaginatedContacts = {
  __typename?: 'PaginatedContacts';
  data: Array<Contact>;
  page: PageResults;
};

export type PaginatedEventTypes = {
  __typename?: 'PaginatedEventTypes';
  data: Array<EventType>;
  page: PageResults;
};

export type PaginatedEvents = {
  __typename?: 'PaginatedEvents';
  data: Array<Event>;
  page: PageResults;
};

export type PaginatedWorkflows = {
  __typename?: 'PaginatedWorkflows';
  data: Array<Workflow>;
  page: PageResults;
};

export type PaymentIntent = {
  __typename?: 'PaymentIntent';
  amountLabel: Scalars['String'];
  createdAt: Scalars['Time'];
  description: Scalars['String'];
  invoiceURL?: Maybe<Scalars['String']>;
  status: Scalars['String'];
};

export type PaymentMethod = {
  __typename?: 'PaymentMethod';
  brand: Scalars['String'];
  createdAt: Scalars['Time'];
  default: Scalars['Boolean'];
  expMonth: Scalars['String'];
  expYear: Scalars['String'];
  last4: Scalars['String'];
};

export type Query = {
  __typename?: 'Query';
  account: Account;
  action: Action;
  actionCategories: Array<ActionCategory>;
  actions: Array<Action>;
  billableStepTimeSeries: Array<TimeSeries>;
  deploy: Deploy;
  deploys?: Maybe<Array<Deploy>>;
  events?: Maybe<PaginatedEvents>;
  plans: Array<Maybe<BillingPlan>>;
  session?: Maybe<Session>;
  workspace: Workspace;
  workspaces?: Maybe<Array<Workspace>>;
};


export type QueryActionArgs = {
  dsn: Scalars['String'];
};


export type QueryActionsArgs = {
  filter?: InputMaybe<ActionFilter>;
};


export type QueryBillableStepTimeSeriesArgs = {
  timeOptions: StepUsageTimeOptions;
};


export type QueryDeployArgs = {
  id: Scalars['ID'];
};


export type QueryDeploysArgs = {
  workspaceID?: InputMaybe<Scalars['ID']>;
};


export type QueryEventsArgs = {
  query?: InputMaybe<EventQuery>;
};


export type QueryWorkspaceArgs = {
  id: Scalars['ID'];
};

export type RegisterInput = {
  anonId?: InputMaybe<Scalars['String']>;
  company?: InputMaybe<Scalars['String']>;
  email: Scalars['String'];
  password: Scalars['String'];
  userID?: InputMaybe<Scalars['String']>;
};

export type RegisterResponse = {
  __typename?: 'RegisterResponse';
  account?: Maybe<Account>;
  user?: Maybe<User>;
};

export type RemoveVercelAppInput = {
  projectID: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type RemoveVercelAppResponse = {
  __typename?: 'RemoveVercelAppResponse';
  success: Scalars['Boolean'];
};

export type RunHistory = {
  __typename?: 'RunHistory';
  createdAt: Scalars['Time'];
  data?: Maybe<Scalars['Bytes']>;
  id: Scalars['ULID'];
  stepData?: Maybe<RunHistoryStepData>;
  type: RunHistoryType;
};

export type RunHistoryCancel = {
  __typename?: 'RunHistoryCancel';
  eventID?: Maybe<Scalars['ULID']>;
  expression?: Maybe<Scalars['String']>;
  userID?: Maybe<Scalars['UUID']>;
};

export type RunHistoryItem = {
  __typename?: 'RunHistoryItem';
  attempt: Scalars['Int'];
  cancel?: Maybe<RunHistoryCancel>;
  createdAt: Scalars['Time'];
  functionVersion: Scalars['Int'];
  groupID?: Maybe<Scalars['UUID']>;
  id: Scalars['ULID'];
  result?: Maybe<RunHistoryResult>;
  sleep?: Maybe<RunHistorySleep>;
  stepName?: Maybe<Scalars['String']>;
  stepType?: Maybe<HistoryStepType>;
  type: HistoryType;
  url?: Maybe<Scalars['String']>;
  waitForEvent?: Maybe<RunHistoryWaitForEvent>;
  waitResult?: Maybe<RunHistoryWaitResult>;
};

export type RunHistoryResult = {
  __typename?: 'RunHistoryResult';
  durationMS: Scalars['Int'];
  errorCode?: Maybe<Scalars['String']>;
  framework?: Maybe<Scalars['String']>;
  platform?: Maybe<Scalars['String']>;
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  sizeBytes: Scalars['Int'];
};

export type RunHistorySleep = {
  __typename?: 'RunHistorySleep';
  until: Scalars['Time'];
};

export type RunHistoryStepData = {
  __typename?: 'RunHistoryStepData';
  attempt: Scalars['Int'];
  data?: Maybe<Scalars['Bytes']>;
  id: Scalars['String'];
  name?: Maybe<Scalars['String']>;
};

export enum RunHistoryType {
  EventReceived = 'EVENT_RECEIVED',
  FunctionCancelled = 'FUNCTION_CANCELLED',
  FunctionCompleted = 'FUNCTION_COMPLETED',
  FunctionFailed = 'FUNCTION_FAILED',
  FunctionScheduled = 'FUNCTION_SCHEDULED',
  FunctionStarted = 'FUNCTION_STARTED',
  StepCompleted = 'STEP_COMPLETED',
  StepErrored = 'STEP_ERRORED',
  StepFailed = 'STEP_FAILED',
  StepScheduled = 'STEP_SCHEDULED',
  StepSleeping = 'STEP_SLEEPING',
  StepStarted = 'STEP_STARTED',
  StepWaiting = 'STEP_WAITING',
  Unknown = 'UNKNOWN'
}

export type RunHistoryWaitForEvent = {
  __typename?: 'RunHistoryWaitForEvent';
  eventName: Scalars['String'];
  expression?: Maybe<Scalars['String']>;
  timeout: Scalars['Time'];
};

export type RunHistoryWaitResult = {
  __typename?: 'RunHistoryWaitResult';
  eventID?: Maybe<Scalars['ULID']>;
  timeout: Scalars['Boolean'];
};

export type RunListConnection = {
  __typename?: 'RunListConnection';
  edges?: Maybe<Array<Maybe<RunListItemEdge>>>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type RunListItem = {
  __typename?: 'RunListItem';
  endedAt?: Maybe<Scalars['Time']>;
  eventID: Scalars['ULID'];
  id: Scalars['ULID'];
  startedAt: Scalars['Time'];
  status: FunctionRunStatus;
};

export type RunListItemEdge = {
  __typename?: 'RunListItemEdge';
  cursor: Scalars['String'];
  node: RunListItem;
};

export type RunTimeline = {
  __typename?: 'RunTimeline';
  durationInMillisconds?: Maybe<Scalars['Int']>;
  history?: Maybe<Array<RunHistory>>;
  latencyInMilliseconds?: Maybe<Scalars['Int']>;
  output?: Maybe<Scalars['Bytes']>;
  status: RunHistoryType;
  step: Scalars['Boolean'];
  stepName?: Maybe<Scalars['String']>;
};

export type RunsFilter = {
  lowerTime: Scalars['Time'];
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<FunctionRunTimeField>;
  upperTime: Scalars['Time'];
};

export type SearchInput = {
  term: Scalars['String'];
};

export type SearchResult = {
  __typename?: 'SearchResult';
  env: Workspace;
  kind: SearchResultType;
  value: SearchResultValue;
};

export enum SearchResultType {
  EventObject = 'EVENT_OBJECT',
  FunctionRun = 'FUNCTION_RUN'
}

export type SearchResultValue = ArchivedEvent | FunctionRun;

export type SearchResults = {
  __typename?: 'SearchResults';
  count: Scalars['Int'];
  results: Array<Maybe<SearchResult>>;
};

export type Segment = {
  __typename?: 'Segment';
  contacts?: Maybe<PaginatedContactSegments>;
  createdAt: Scalars['Time'];
  expression?: Maybe<Scalars['NullString']>;
  id: Scalars['ID'];
  name: Scalars['String'];
  ratio: Scalars['Int'];
  type: Scalars['SegmentType'];
  updatedAt: Scalars['Time'];
};

export type SegmentOpts = {
  expression?: InputMaybe<Scalars['String']>;
  name: Scalars['String'];
  ratio?: InputMaybe<Scalars['Int']>;
};

export type Session = {
  __typename?: 'Session';
  expires?: Maybe<Scalars['Time']>;
  user: User;
};

export type StartWorkflow = {
  workflowID: Scalars['ID'];
  workspaceID: Scalars['ID'];
};

export type StartWorkflowInput = {
  baggage?: InputMaybe<WorkflowBaggageInput>;
  fromActionID?: InputMaybe<Scalars['Int']>;
  workflowID: Scalars['ID'];
  workflowVersion?: InputMaybe<Scalars['Int']>;
  workspaceID: Scalars['ID'];
};

export type StartWorkflowResponse = {
  __typename?: 'StartWorkflowResponse';
  id: Scalars['ULID'];
};

export type StepUsageTimeOptions = {
  interval?: InputMaybe<Scalars['String']>;
  month?: InputMaybe<Scalars['Int']>;
  year?: InputMaybe<Scalars['Int']>;
};

export type StripeSubscriptionInput = {
  items: Array<StripeSubscriptionItemsInput>;
};

export type StripeSubscriptionItemsInput = {
  amount: Scalars['Int'];
  planID: Scalars['ID'];
  quantity: Scalars['Int'];
};

export type TimeSeries = {
  __typename?: 'TimeSeries';
  data: Array<TimeSeriesPoint>;
  name: Scalars['String'];
};

export type TimeSeriesPoint = {
  __typename?: 'TimeSeriesPoint';
  time: Scalars['Time'];
  value?: Maybe<Scalars['Float']>;
};

export type UpdateAccount = {
  billingEmail?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type UpdateActionSetting = {
  category: Scalars['String'];
  data: Scalars['String'];
  dsn: Scalars['DSN'];
  id: Scalars['ID'];
  name: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type UpdateIngestKey = {
  filterList?: InputMaybe<FilterListInput>;
  metadata?: InputMaybe<Scalars['Map']>;
  name?: InputMaybe<Scalars['String']>;
};

export type UpdateUser = {
  email?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
  password?: InputMaybe<Scalars['String']>;
};

export type UpdateVercelAppInput = {
  path: Scalars['String'];
  projectID: Scalars['String'];
};

export type UpdateVercelAppResponse = {
  __typename?: 'UpdateVercelAppResponse';
  success: Scalars['Boolean'];
  vercelApp?: Maybe<VercelApp>;
};

export type UpsertActionSecret = {
  data: Scalars['String'];
  dsn?: InputMaybe<Scalars['DSN']>;
  name: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type UpsertActionSetting = {
  category: Scalars['String'];
  data: Scalars['String'];
  dsn: Scalars['DSN'];
  name: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type UpsertWorkflowInput = {
  config?: InputMaybe<Scalars['String']>;
  description?: InputMaybe<Scalars['String']>;
  live?: InputMaybe<Scalars['Boolean']>;
  workspaceID: Scalars['ID'];
};

export type Usage = {
  __typename?: 'Usage';
  asOf: Scalars['Time'];
  data: Array<UsageSlot>;
  period: Scalars['Period'];
  range: Scalars['Timerange'];
  total: Scalars['Int'];
};

export type UsageInput = {
  from?: InputMaybe<Scalars['Time']>;
  period?: InputMaybe<Scalars['Period']>;
  range?: InputMaybe<Scalars['Timerange']>;
  to?: InputMaybe<Scalars['Time']>;
};

export type UsageSlot = {
  __typename?: 'UsageSlot';
  count: Scalars['Int'];
  slot: Scalars['Time'];
};

export type User = {
  __typename?: 'User';
  account?: Maybe<Account>;
  createdAt: Scalars['Time'];
  email: Scalars['String'];
  id: Scalars['ID'];
  lastLoginAt?: Maybe<Scalars['Time']>;
  name?: Maybe<Scalars['NullString']>;
  passwordChangedAt?: Maybe<Scalars['Time']>;
  roles?: Maybe<Array<Maybe<Scalars['Role']>>>;
  updatedAt: Scalars['Time'];
};

export type VercelApp = {
  __typename?: 'VercelApp';
  id: Scalars['UUID'];
  path?: Maybe<Scalars['String']>;
  projectID: Scalars['String'];
  workspaceID: Scalars['UUID'];
};

export type VisibleApiKey = {
  __typename?: 'VisibleAPIKey';
  author: User;
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  name: Scalars['NullString'];
  private: Scalars['String'];
};

export type Workflow = {
  __typename?: 'Workflow';
  appName?: Maybe<Scalars['String']>;
  archivedAt?: Maybe<Scalars['Time']>;
  communication: Array<Communication>;
  current?: Maybe<WorkflowVersion>;
  drafts: Array<Maybe<WorkflowVersion>>;
  id: Scalars['ID'];
  isArchived: Scalars['Boolean'];
  isPaused: Scalars['Boolean'];
  latest: WorkflowVersion;
  metrics: MetricsResponse;
  name: Scalars['String'];
  pauses: Array<WorkflowPause>;
  previous: Array<Maybe<WorkflowVersion>>;
  run: FunctionRun;
  runMetrics: Usage;
  runs?: Maybe<RunListConnection>;
  runsV2?: Maybe<RunListConnection>;
  slug: Scalars['String'];
  url: Scalars['String'];
  usage: Usage;
  version: WorkflowVersion;
};


export type WorkflowMetricsArgs = {
  opts: MetricsRequest;
};


export type WorkflowPausesArgs = {
  runID: Scalars['ULID'];
};


export type WorkflowRunArgs = {
  id: Scalars['ULID'];
};


export type WorkflowRunMetricsArgs = {
  opts?: InputMaybe<UsageInput>;
  status?: InputMaybe<Scalars['String']>;
};


export type WorkflowRunsArgs = {
  after?: InputMaybe<Scalars['String']>;
  filter: RunsFilter;
  first?: Scalars['Int'];
};


export type WorkflowRunsV2Args = {
  after?: InputMaybe<Scalars['String']>;
  filter: RunsFilter;
  first?: Scalars['Int'];
};


export type WorkflowUsageArgs = {
  event?: InputMaybe<Scalars['String']>;
  opts?: InputMaybe<UsageInput>;
};


export type WorkflowVersionArgs = {
  id: Scalars['Int'];
};

export type WorkflowBaggageInput = {
  actions?: InputMaybe<Scalars['Map']>;
  event: Scalars['Bytes'];
};

export type WorkflowEvent = {
  __typename?: 'WorkflowEvent';
  branchRunID: Scalars['ULID'];
  contact?: Maybe<Contact>;
  contactID?: Maybe<Scalars['ID']>;
  createdAt: Scalars['Time'];
  data?: Maybe<Scalars['String']>;
  eventID: Scalars['ULID'];
  fromActionID?: Maybe<Scalars['Int']>;
  id: Scalars['ULID'];
  name: Scalars['String'];
  workflow: Workflow;
  workflowID: Scalars['ID'];
  workflowRunID: Scalars['ULID'];
  workflowVersion: Scalars['Int'];
  workflowVersionInstance: WorkflowVersion;
  workspaceID: Scalars['UUID'];
};

export type WorkflowPause = {
  __typename?: 'WorkflowPause';
  actionID?: Maybe<Scalars['String']>;
  childActionID?: Maybe<Scalars['String']>;
  createdAt: Scalars['Time'];
  eventName?: Maybe<Scalars['NullString']>;
  expiresAt: Scalars['Time'];
  expression?: Maybe<Scalars['NullString']>;
  id: Scalars['ID'];
  parentActionID?: Maybe<Scalars['String']>;
  workflowRunID: Scalars['ULID'];
};

export type WorkflowResponse = {
  __typename?: 'WorkflowResponse';
  workflow: Workflow;
};

export type WorkflowStart = {
  __typename?: 'WorkflowStart';
  event: WorkflowEvent;
  status: Scalars['String'];
};

export type WorkflowTrigger = {
  __typename?: 'WorkflowTrigger';
  eventName?: Maybe<Scalars['NullString']>;
  schedule?: Maybe<Scalars['NullString']>;
  workflowID: Scalars['ID'];
  workflowVersion: Scalars['Int'];
};

export type WorkflowVersion = {
  __typename?: 'WorkflowVersion';
  actions: Array<Action>;
  alerts?: Maybe<Array<Alert>>;
  communication: Array<Communication>;
  config: Scalars['String'];
  configJSON: Scalars['String'];
  createdAt: Scalars['Time'];
  deploy?: Maybe<Deploy>;
  description?: Maybe<Scalars['NullString']>;
  retries: Scalars['Int'];
  throttleCount: Scalars['Int'];
  throttlePeriod: Scalars['String'];
  triggers: Array<WorkflowTrigger>;
  updatedAt: Scalars['Time'];
  url: Scalars['String'];
  validFrom?: Maybe<Scalars['Time']>;
  validTo?: Maybe<Scalars['Time']>;
  version: Scalars['Int'];
  workflowID: Scalars['ID'];
  workflowType: Scalars['String'];
};

export type WorkflowVersionResponse = {
  __typename?: 'WorkflowVersionResponse';
  version: WorkflowVersion;
  workflow: Workflow;
};

export type Workspace = {
  __typename?: 'Workspace';
  actionSettings: Array<ActionSetting>;
  archivedEvent?: Maybe<ArchivedEvent>;
  contact: Contact;
  contactStats: ContactStats;
  contacts: PaginatedContacts;
  createdAt: Scalars['Time'];
  event?: Maybe<Event>;
  eventByNames: Array<EventType>;
  eventTypes: PaginatedEventTypes;
  events: PaginatedEvents;
  functionCount: Scalars['Int'];
  id: Scalars['ID'];
  ingestKey: IngestKey;
  ingestKeys: Array<IngestKey>;
  integrations?: Maybe<Array<WorkspaceIntegration>>;
  isArchived: Scalars['Boolean'];
  isAutoArchiveEnabled: Scalars['Boolean'];
  lastDeployedAt?: Maybe<Scalars['Time']>;
  name: Scalars['String'];
  parentID?: Maybe<Scalars['ID']>;
  secrets: Array<Maybe<ActionSecret>>;
  segment: Segment;
  segments: Array<Maybe<Segment>>;
  test: Scalars['Boolean'];
  type: EnvironmentType;
  vercelApps: Array<VercelApp>;
  webhookSigningKey: Scalars['String'];
  workflow?: Maybe<Workflow>;
  workflowBySlug?: Maybe<Workflow>;
  workflows: PaginatedWorkflows;
};


export type WorkspaceActionSettingsArgs = {
  category?: InputMaybe<Scalars['String']>;
  dsn: Scalars['DSN'];
};


export type WorkspaceArchivedEventArgs = {
  id: Scalars['ULID'];
};


export type WorkspaceContactArgs = {
  id: Scalars['ID'];
};


export type WorkspaceContactsArgs = {
  filter?: InputMaybe<Scalars['Map']>;
};


export type WorkspaceEventArgs = {
  name: Scalars['String'];
};


export type WorkspaceEventByNamesArgs = {
  names: Array<Scalars['String']>;
};


export type WorkspaceEventsArgs = {
  prefix?: InputMaybe<Scalars['String']>;
};


export type WorkspaceIngestKeyArgs = {
  id: Scalars['ID'];
};


export type WorkspaceIngestKeysArgs = {
  filter?: InputMaybe<IngestKeyFilter>;
};


export type WorkspaceSecretsArgs = {
  service?: InputMaybe<Scalars['DSN']>;
};


export type WorkspaceSegmentArgs = {
  id: Scalars['ID'];
};


export type WorkspaceSegmentsArgs = {
  prefix?: InputMaybe<Scalars['String']>;
};


export type WorkspaceWorkflowArgs = {
  id: Scalars['ID'];
};


export type WorkspaceWorkflowBySlugArgs = {
  slug: Scalars['String'];
};


export type WorkspaceWorkflowsArgs = {
  archived?: InputMaybe<Scalars['Boolean']>;
};

export type WorkspaceIntegration = {
  __typename?: 'WorkspaceIntegration';
  actions: Scalars['Boolean'];
  events: Scalars['Boolean'];
  name: Scalars['String'];
  service: Scalars['String'];
  webhookEndpoints: Array<Scalars['String']>;
};

export type CreateEnvironmentMutationVariables = Exact<{
  name: Scalars['String'];
}>;


export type CreateEnvironmentMutation = { __typename?: 'Mutation', createWorkspace: Array<{ __typename?: 'Workspace', id: string } | null> };

export type ArchiveEnvironmentMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type ArchiveEnvironmentMutation = { __typename?: 'Mutation', archiveEnvironment: { __typename?: 'Workspace', id: string } };

export type UnarchiveEnvironmentMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type UnarchiveEnvironmentMutation = { __typename?: 'Mutation', unarchiveEnvironment: { __typename?: 'Workspace', id: string } };

export type DisableEnvironmentAutoArchiveDocumentMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type DisableEnvironmentAutoArchiveDocumentMutation = { __typename?: 'Mutation', disableEnvironmentAutoArchive: { __typename?: 'Workspace', id: string } };

export type EnableEnvironmentAutoArchiveMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type EnableEnvironmentAutoArchiveMutation = { __typename?: 'Mutation', enableEnvironmentAutoArchive: { __typename?: 'Workspace', id: string } };

export type GetDeployQueryVariables = Exact<{
  deployID: Scalars['ID'];
}>;


export type GetDeployQuery = { __typename?: 'Query', deploy: { __typename?: 'Deploy', id: string, appName: string, authorID?: string | null, checksum: string, createdAt: string, error?: string | null, framework?: string | null, metadata: Record<string, unknown>, sdkLanguage: string, sdkVersion: string, status: string, url?: string | null, deployedFunctions: Array<{ __typename?: 'Workflow', slug: string, name: string }>, removedFunctions: Array<{ __typename?: 'Workflow', slug: string, name: string }> } };

export type GetEventKeysForBlankSlateQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetEventKeysForBlankSlateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', name: null | string, presharedKey: string, createdAt: string }> } };

export type ArchiveEventMutationVariables = Exact<{
  environmentId: Scalars['ID'];
  name: Scalars['String'];
}>;


export type ArchiveEventMutation = { __typename?: 'Mutation', archiveEvent?: { __typename?: 'Event', name: string } | null };

export type GetLatestEventLogsQueryVariables = Exact<{
  name?: InputMaybe<Scalars['String']>;
  environmentID: Scalars['ID'];
}>;


export type GetLatestEventLogsQuery = { __typename?: 'Query', events?: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', recent: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string, event: string, source?: { __typename?: 'IngestKey', name: null | string } | null }> }> } | null };

export type GetEventKeysQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetEventKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventKeys: Array<{ __typename?: 'IngestKey', name: null | string, value: string }> } };

export type GetEventLogQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  eventName: Scalars['String'];
  cursor?: InputMaybe<Scalars['String']>;
  perPage: Scalars['Int'];
}>;


export type GetEventLogQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventType?: { __typename?: 'Event', events: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string }> } | null } };

export type EventPayloadFragment = { __typename?: 'ArchivedEvent', payload: string } & { ' $fragmentName'?: 'EventPayloadFragment' };

export type GetFunctionRunCardQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunCardQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', name: string, slug: string, run: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus, startedAt: string } } | null } };

export type GetEventQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  eventID: Scalars['ULID'];
}>;


export type GetEventQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', event?: (
      { __typename?: 'ArchivedEvent', receivedAt: string, functionRuns: Array<{ __typename?: 'FunctionRun', id: string, function: { __typename?: 'Workflow', id: string } }> }
      & { ' $fragmentRefs'?: { 'EventPayloadFragment': EventPayloadFragment } }
    ) | null } };

export type GetBillingPlanQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingPlanQuery = { __typename?: 'Query', account: { __typename?: 'Account', plan?: { __typename?: 'BillingPlan', id: string, name: string, features: Record<string, unknown> } | null }, plans: Array<{ __typename?: 'BillingPlan', name: string, features: Record<string, unknown> } | null> };

export type GetFunctionRunsMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionRunsMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', completed: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, canceled: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, failed: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetFnMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFnMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, concurrencyLimit: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFailedFunctionRunsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  lowerTime: Scalars['Time'];
  upperTime: Scalars['Time'];
}>;


export type GetFailedFunctionRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', failedRuns?: { __typename?: 'RunListConnection', edges?: Array<{ __typename?: 'RunListItemEdge', node: { __typename?: 'RunListItem', id: string, endedAt?: string | null } } | null> | null } | null } | null } };

export type GetSdkRequestMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetSdkRequestMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type ArchiveFunctionMutationVariables = Exact<{
  input: ArchiveWorkflowInput;
}>;


export type ArchiveFunctionMutation = { __typename?: 'Mutation', archiveWorkflow?: { __typename?: 'WorkflowResponse', workflow: { __typename?: 'Workflow', id: string } } | null };

export type GetFunctionArchivalQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionArchivalQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow?: { __typename?: 'Workflow', id: string, isArchived: boolean, name: string } | null } };

export type GetFunctionVersionNumberQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionVersionNumberQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow?: { __typename?: 'Workflow', id: string, archivedAt?: string | null, current?: { __typename?: 'WorkflowVersion', version: number } | null, previous: Array<{ __typename?: 'WorkflowVersion', version: number } | null> } | null } };

export type PauseFunctionMutationVariables = Exact<{
  input: EditWorkflowInput;
}>;


export type PauseFunctionMutation = { __typename?: 'Mutation', editWorkflow?: { __typename?: 'WorkflowVersionResponse', workflow: { __typename?: 'Workflow', id: string, name: string } } | null };

export type GetFunctionRunsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunStatuses?: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  functionRunCursor?: InputMaybe<Scalars['String']>;
  timeRangeStart: Scalars['Time'];
  timeRangeEnd: Scalars['Time'];
  timeField: FunctionRunTimeField;
}>;


export type GetFunctionRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', id: string, runs?: { __typename?: 'RunListConnection', edges?: Array<{ __typename?: 'RunListItemEdge', node: { __typename?: 'RunListItem', id: string, status: FunctionRunStatus, startedAt: string, endedAt?: string | null } } | null> | null, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor?: string | null } } | null } | null } };

export type FunctionItemFragment = { __typename?: 'Workflow', id: string, slug: string } & { ' $fragmentName'?: 'FunctionItemFragment' };

export type RerunFunctionRunMutationVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
  functionRunID: Scalars['ULID'];
}>;


export type RerunFunctionRunMutation = { __typename?: 'Mutation', retryWorkflowRun?: { __typename?: 'StartWorkflowResponse', id: string } | null };

export type GetFunctionRunTimelineQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunTimelineQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: (
      { __typename?: 'Workflow', run: { __typename?: 'FunctionRun', canRerun: boolean, timeline?: Array<{ __typename?: 'RunTimeline', stepName?: string | null, output?: string | null, type: RunHistoryType, history?: Array<{ __typename?: 'RunHistory', id: string, type: RunHistoryType, createdAt: string, stepData?: { __typename?: 'RunHistoryStepData', data?: string | null } | null }> | null }> | null } }
      & { ' $fragmentRefs'?: { 'FunctionItemFragment': FunctionItemFragment } }
    ) | null } };

export type GetFunctionRunOutputQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunOutputQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', run: { __typename?: 'FunctionRun', output?: string | null } } | null } };

export type GetFunctionRunPayloadQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunPayloadQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', functionRun: { __typename?: 'FunctionRun', events: Array<{ __typename?: 'ArchivedEvent', payload: string }> } } | null } };

export type GetHistoryItemOutputQueryVariables = Exact<{
  envID: Scalars['ID'];
  functionID: Scalars['ID'];
  historyItemID: Scalars['ULID'];
  runID: Scalars['ULID'];
}>;


export type GetHistoryItemOutputQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', run: { __typename?: 'FunctionRun', historyItemOutput?: string | null } } | null } };

export type GetFunctionRunDetailsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunDetailsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', id: string, name: string, run: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus, startedAt: string, endedAt?: string | null, output?: string | null, event?: { __typename?: 'ArchivedEvent', id: string, name: string, receivedAt: string, payload: string } | null, history: Array<{ __typename?: 'RunHistoryItem', attempt: number, createdAt: string, functionVersion: number, groupID?: string | null, id: string, stepName?: string | null, type: HistoryType, url?: string | null, cancel?: { __typename?: 'RunHistoryCancel', eventID?: string | null, expression?: string | null, userID?: string | null } | null, sleep?: { __typename?: 'RunHistorySleep', until: string } | null, waitForEvent?: { __typename?: 'RunHistoryWaitForEvent', eventName: string, expression?: string | null, timeout: string } | null, waitResult?: { __typename?: 'RunHistoryWaitResult', eventID?: string | null, timeout: boolean } | null }>, functionVersion: { __typename?: 'WorkflowVersion', validFrom?: string | null, version: number, deploy?: { __typename?: 'Deploy', id: string, createdAt: string } | null, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName?: null | string | null, schedule?: null | string | null }> }, version: { __typename?: 'WorkflowVersion', url: string, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName?: null | string | null, schedule?: null | string | null }> } } } | null } };

export type GetFunctionRunsCountQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunStatuses?: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeRangeStart: Scalars['Time'];
  timeRangeEnd: Scalars['Time'];
  timeField: FunctionRunTimeField;
}>;


export type GetFunctionRunsCountQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', id: string, runs?: { __typename?: 'RunListConnection', totalCount: number } | null } | null } };

export type NewIngestKeyMutationVariables = Exact<{
  input: NewIngestKey;
}>;


export type NewIngestKeyMutation = { __typename?: 'Mutation', key: { __typename?: 'IngestKey', id: string } };

export type GetIngestKeysQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetIngestKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, source: string }> } };

export type UpdateIngestKeyMutationVariables = Exact<{
  id: Scalars['ID'];
  input: UpdateIngestKey;
}>;


export type UpdateIngestKeyMutation = { __typename?: 'Mutation', updateIngestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url?: string | null, metadata?: Record<string, unknown> | null, filter: { __typename?: 'FilterList', type?: unknown | null, ips?: Array<string> | null, events?: Array<string> | null } } };

export type DeleteEventKeyMutationVariables = Exact<{
  input: DeleteIngestKey;
}>;


export type DeleteEventKeyMutation = { __typename?: 'Mutation', deleteIngestKey?: { __typename?: 'DeleteResponse', ids: Array<string> } | null };

export type GetIngestKeyQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  keyID: Scalars['ID'];
}>;


export type GetIngestKeyQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url?: string | null, metadata?: Record<string, unknown> | null, filter: { __typename?: 'FilterList', type?: unknown | null, ips?: Array<string> | null, events?: Array<string> | null } } } };

export type GetSigningKeyQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetSigningKeyQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', webhookSigningKey: string } };

export type UpdateAccountMutationVariables = Exact<{
  input: UpdateAccount;
}>;


export type UpdateAccountMutation = { __typename?: 'Mutation', account: { __typename?: 'Account', billingEmail: string, name?: null | string | null } };

export type CreateStripeSubscriptionMutationVariables = Exact<{
  input: StripeSubscriptionInput;
}>;


export type CreateStripeSubscriptionMutation = { __typename?: 'Mutation', createStripeSubscription: { __typename?: 'CreateStripeSubscriptionResponse', clientSecret: string, message: string } };

export type UpdatePlanMutationVariables = Exact<{
  planID: Scalars['ID'];
}>;


export type UpdatePlanMutation = { __typename?: 'Mutation', updatePlan: { __typename?: 'Account', plan?: { __typename?: 'BillingPlan', id: string, name: string } | null } };

export type GetPaymentIntentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPaymentIntentsQuery = { __typename?: 'Query', account: { __typename?: 'Account', paymentIntents: Array<{ __typename?: 'PaymentIntent', status: string, createdAt: string, amountLabel: string, description: string, invoiceURL?: string | null }> } };

export type UpdatePaymentMethodMutationVariables = Exact<{
  token: Scalars['String'];
}>;


export type UpdatePaymentMethodMutation = { __typename?: 'Mutation', updatePaymentMethod?: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null };

export type GetBillingInfoQueryVariables = Exact<{
  prevMonth: Scalars['Int'];
  thisMonth: Scalars['Int'];
}>;


export type GetBillingInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', billingEmail: string, name?: null | string | null, plan?: { __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, features: Record<string, unknown> } | null, subscription?: { __typename?: 'BillingSubscription', nextInvoiceDate: string } | null, paymentMethods?: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null }, plans: Array<{ __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, features: Record<string, unknown> } | null>, billableStepTimeSeriesPrevMonth: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value?: number | null }> }>, billableStepTimeSeriesThisMonth: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value?: number | null }> }> };

export type GetSavedVercelProjectsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetSavedVercelProjectsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', savedVercelProjects: Array<{ __typename?: 'VercelApp', projectID: string, path?: string | null }> } };

export type CreateVercelAppMutationVariables = Exact<{
  input: CreateVercelAppInput;
}>;


export type CreateVercelAppMutation = { __typename?: 'Mutation', createVercelApp?: { __typename?: 'CreateVercelAppResponse', success: boolean } | null };

export type UpdateVercelAppMutationVariables = Exact<{
  input: UpdateVercelAppInput;
}>;


export type UpdateVercelAppMutation = { __typename?: 'Mutation', updateVercelApp?: { __typename?: 'UpdateVercelAppResponse', success: boolean } | null };

export type RemoveVercelAppMutationVariables = Exact<{
  input: RemoveVercelAppInput;
}>;


export type RemoveVercelAppMutation = { __typename?: 'Mutation', removeVercelApp?: { __typename?: 'RemoveVercelAppResponse', success: boolean } | null };

export type DeleteUserMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type DeleteUserMutation = { __typename?: 'Mutation', deleteUser: string };

export type CreateUserMutationVariables = Exact<{
  input: NewUser;
}>;


export type CreateUserMutation = { __typename?: 'Mutation', createUser?: { __typename?: 'User', id: string } | null };

export type GetUsersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetUsersQuery = { __typename?: 'Query', account: { __typename?: 'Account', users: Array<{ __typename?: 'User', createdAt: string, email: string, id: string, lastLoginAt?: string | null, name?: null | string | null }> }, session?: { __typename?: 'Session', user: { __typename?: 'User', id: string } } | null };

export type GetAccountSupportInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAccountSupportInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, plan?: { __typename?: 'BillingPlan', id: string, name: string, amount: number, features: Record<string, unknown> } | null } };

export type GetAccountNameQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAccountNameQuery = { __typename?: 'Query', account: { __typename?: 'Account', name?: null | string | null } };

export type GetGlobalSearchQueryVariables = Exact<{
  opts: SearchInput;
}>;


export type GetGlobalSearchQuery = { __typename?: 'Query', account: { __typename?: 'Account', search: { __typename?: 'SearchResults', results: Array<{ __typename?: 'SearchResult', kind: SearchResultType, env: { __typename?: 'Workspace', name: string, id: string, type: EnvironmentType }, value: { __typename?: 'ArchivedEvent', id: string, name: string } | { __typename?: 'FunctionRun', id: string, functionID: string } } | null> } } };

export type GetFunctionSlugQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
}>;


export type GetFunctionSlugQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function?: { __typename?: 'Workflow', slug: string, name: string } | null } };

export type GetDeployssQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetDeployssQuery = { __typename?: 'Query', deploys?: Array<{ __typename?: 'Deploy', id: string, appName: string, authorID?: string | null, checksum: string, createdAt: string, error?: string | null, framework?: string | null, metadata: Record<string, unknown>, sdkLanguage: string, sdkVersion: string, status: string, deployedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }>, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }> }> | null };

export type GetEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetEnvironmentsQuery = { __typename?: 'Query', workspaces?: Array<{ __typename?: 'Workspace', id: string, name: string, parentID?: string | null, test: boolean, type: EnvironmentType, webhookSigningKey: string, createdAt: string, isArchived: boolean, functionCount: number, isAutoArchiveEnabled: boolean, lastDeployedAt?: string | null }> | null };

export type GetEventTypesQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page?: InputMaybe<Scalars['Int']>;
}>;


export type GetEventTypesQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, functions: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string }>, dailyVolume: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }>, page: { __typename?: 'PageResults', page: number, totalPages?: number | null } } } };

export type GetEventTypeQueryVariables = Exact<{
  eventName?: InputMaybe<Scalars['String']>;
  environmentID: Scalars['ID'];
}>;


export type GetEventTypeQuery = { __typename?: 'Query', events?: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, description?: string | null, schemaSource?: string | null, integrationName?: string | null, workflows: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string, current?: { __typename?: 'WorkflowVersion', createdAt: string } | null }>, recent: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string, occurredAt: string, name: string, event: string, version: string, contactID?: string | null, ingestSourceID?: string | null, contact?: { __typename?: 'Contact', predefinedAttributes: Record<string, unknown> } | null, source?: { __typename?: 'IngestKey', name: null | string } | null }>, usage: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } }> } | null };

export type GetFunctionsUsageQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page?: InputMaybe<Scalars['Int']>;
  archived?: InputMaybe<Scalars['Boolean']>;
}>;


export type GetFunctionsUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems?: number | null, totalPages?: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, dailyStarts: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyFailures: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }> } } };

export type GetFunctionsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page?: InputMaybe<Scalars['Int']>;
  archived?: InputMaybe<Scalars['Boolean']>;
}>;


export type GetFunctionsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems?: number | null, totalPages?: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string, isArchived: boolean, current?: { __typename?: 'WorkflowVersion', version: number, description?: null | string | null, validFrom?: string | null, validTo?: string | null, workflowType: string, throttlePeriod: string, throttleCount: number, alerts?: Array<{ __typename?: 'Alert', workflowID: string }> | null, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName?: null | string | null, schedule?: null | string | null }> } | null }> } } };

export type GetFunctionQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', id: string, workflow?: { __typename?: 'Workflow', id: string, name: string, slug: string, isArchived: boolean, url: string, current?: { __typename?: 'WorkflowVersion', workflowID: string, version: number, config: string, retries: number, validFrom?: string | null, validTo?: string | null, description?: null | string | null, updatedAt: string, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName?: null | string | null, schedule?: null | string | null }>, deploy?: { __typename?: 'Deploy', id: string, createdAt: string } | null } | null } | null } };

export type FunctionVersionFragment = { __typename?: 'WorkflowVersion', version: number, validFrom?: string | null, validTo?: string | null, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName?: null | string | null, schedule?: null | string | null }>, deploy?: { __typename?: 'Deploy', id: string } | null } & { ' $fragmentName'?: 'FunctionVersionFragment' };

export type GetFunctionVersionsQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionVersionsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow?: { __typename?: 'Workflow', archivedAt?: string | null, current?: (
        { __typename?: 'WorkflowVersion' }
        & { ' $fragmentRefs'?: { 'FunctionVersionFragment': FunctionVersionFragment } }
      ) | null, previous: Array<(
        { __typename?: 'WorkflowVersion' }
        & { ' $fragmentRefs'?: { 'FunctionVersionFragment': FunctionVersionFragment } }
      ) | null> } | null } };

export type GetFunctionUsageQueryVariables = Exact<{
  id: Scalars['ID'];
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow?: { __typename?: 'Workflow', dailyStarts: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyFailures: { __typename?: 'Usage', period: unknown, total: number, asOf: string, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetAllEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAllEnvironmentsQuery = { __typename?: 'Query', workspaces?: Array<{ __typename?: 'Workspace', id: string, name: string, parentID?: string | null, test: boolean, type: EnvironmentType, createdAt: string, isArchived: boolean, isAutoArchiveEnabled: boolean }> | null };

export const EventPayloadFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<EventPayloadFragment, unknown>;
export const FunctionItemFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FunctionItem"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Workflow"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]} as unknown as DocumentNode<FunctionItemFragment, unknown>;
export const FunctionVersionFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FunctionVersion"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WorkflowVersion"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"validTo"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<FunctionVersionFragment, unknown>;
export const CreateEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createWorkspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateEnvironmentMutation, CreateEnvironmentMutationVariables>;
export const ArchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<ArchiveEnvironmentMutation, ArchiveEnvironmentMutationVariables>;
export const UnarchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnarchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnarchiveEnvironmentMutation, UnarchiveEnvironmentMutationVariables>;
export const DisableEnvironmentAutoArchiveDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisableEnvironmentAutoArchiveDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<DisableEnvironmentAutoArchiveDocumentMutation, DisableEnvironmentAutoArchiveDocumentMutationVariables>;
export const EnableEnvironmentAutoArchiveDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EnableEnvironmentAutoArchive"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<EnableEnvironmentAutoArchiveMutation, EnableEnvironmentAutoArchiveMutationVariables>;
export const GetDeployDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDeploy"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"deployID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"deployID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"checksum"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetDeployQuery, GetDeployQueryVariables>;
export const GetEventKeysForBlankSlateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeysForBlankSlate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"source"},"value":{"kind":"StringValue","value":"key","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysForBlankSlateQuery, GetEventKeysForBlankSlateQueryVariables>;
export const ArchiveEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}}},{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<ArchiveEventMutation, ArchiveEventMutationVariables>;
export const GetLatestEventLogsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLatestEventLogs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"recent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"count"},"value":{"kind":"IntValue","value":"5"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetLatestEventLogsQuery, GetLatestEventLogsQueryVariables>;
export const GetEventKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventKeys"},"name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"value"},"name":{"kind":"Name","value":"presharedKey"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysQuery, GetEventKeysQueryVariables>;
export const GetEventLogDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventLog"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventType"},"name":{"kind":"Name","value":"event"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"events"},"name":{"kind":"Name","value":"recent"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"cursored"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"cursor"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventLogQuery, GetEventLogQueryVariables>;
export const GetFunctionRunCardDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunCard"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunCardQuery, GetFunctionRunCardQueryVariables>;
export const GetEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"event"},"name":{"kind":"Name","value":"archivedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"EventPayload"}},{"kind":"Field","name":{"kind":"Name","value":"functionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<GetEventQuery, GetEventQueryVariables>;
export const GetBillingPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingPlan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]} as unknown as DocumentNode<GetBillingPlanQuery, GetBillingPlanQueryVariables>;
export const GetFunctionRunsMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"canceled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"failed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsMetricsQuery, GetFunctionRunsMetricsQueryVariables>;
export const GetFnMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"concurrencyLimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnMetricsQuery, GetFnMetricsQueryVariables>;
export const GetFailedFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFailedFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"failedRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"FAILED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"ENDED_AT"}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFailedFunctionRunsQuery, GetFailedFunctionRunsQueryVariables>;
export const GetSdkRequestMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSDKRequestMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSdkRequestMetricsQuery, GetSdkRequestMetricsQueryVariables>;
export const ArchiveFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ArchiveWorkflowInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveWorkflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<ArchiveFunctionMutation, ArchiveFunctionMutationVariables>;
export const GetFunctionArchivalDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionArchival"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionArchivalQuery, GetFunctionArchivalQueryVariables>;
export const GetFunctionVersionNumberDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionVersionNumber"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}}]}},{"kind":"Field","name":{"kind":"Name","value":"previous"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionVersionNumberQuery, GetFunctionVersionNumberQueryVariables>;
export const PauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"EditWorkflowInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"editWorkflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<PauseFunctionMutation, PauseFunctionMutationVariables>;
export const GetFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunTimeField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsQuery, GetFunctionRunsQueryVariables>;
export const RerunFunctionRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RerunFunctionRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryWorkflowRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"workflowRunID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<RerunFunctionRunMutation, RerunFunctionRunMutationVariables>;
export const GetFunctionRunTimelineDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunTimeline"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"FunctionItem"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"canRerun"}},{"kind":"Field","name":{"kind":"Name","value":"timeline"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stepName"}},{"kind":"Field","alias":{"kind":"Name","value":"type"},"name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"stepData"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"}}]}}]}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FunctionItem"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Workflow"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]} as unknown as DocumentNode<GetFunctionRunTimelineQuery, GetFunctionRunTimelineQueryVariables>;
export const GetFunctionRunOutputDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunOutput"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"output"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunOutputQuery, GetFunctionRunOutputQueryVariables>;
export const GetFunctionRunPayloadDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunPayload"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"functionRun"},"name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunPayloadQuery, GetFunctionRunPayloadQueryVariables>;
export const GetHistoryItemOutputDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetHistoryItemOutput"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"historyItemOutput"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}}}]}]}}]}}]}}]}}]} as unknown as DocumentNode<GetHistoryItemOutputQuery, GetHistoryItemOutputQueryVariables>;
export const GetFunctionRunDetailsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunDetails"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"attempt"}},{"kind":"Field","name":{"kind":"Name","value":"cancel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionVersion"}},{"kind":"Field","name":{"kind":"Name","value":"groupID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"sleep"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"until"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"waitForEvent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"waitResult"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","alias":{"kind":"Name","value":"functionVersion"},"name":{"kind":"Name","value":"workflowVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","alias":{"kind":"Name","value":"version"},"name":{"kind":"Name","value":"workflowVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunDetailsQuery, GetFunctionRunDetailsQueryVariables>;
export const GetFunctionRunsCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsCount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunTimeField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsCountQuery, GetFunctionRunsCountQueryVariables>;
export const NewIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"NewIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<NewIngestKeyMutation, NewIngestKeyMutationVariables>;
export const GetIngestKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeysQuery, GetIngestKeysQueryVariables>;
export const UpdateIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}}]}}]}}]} as unknown as DocumentNode<UpdateIngestKeyMutation, UpdateIngestKeyMutationVariables>;
export const DeleteEventKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteEventKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DeleteIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<DeleteEventKeyMutation, DeleteEventKeyMutationVariables>;
export const GetIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeyQuery, GetIngestKeyQueryVariables>;
export const GetSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetSigningKeyQuery, GetSigningKeyQueryVariables>;
export const UpdateAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAccount"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"account"},"name":{"kind":"Name","value":"updateAccount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountMutation, UpdateAccountMutationVariables>;
export const CreateStripeSubscriptionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripeSubscription"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StripeSubscriptionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createStripeSubscription"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clientSecret"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CreateStripeSubscriptionMutation, CreateStripeSubscriptionMutationVariables>;
export const UpdatePlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<UpdatePlanMutation, UpdatePlanMutationVariables>;
export const GetPaymentIntentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPaymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"paymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"amountLabel"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"invoiceURL"}}]}}]}}]}}]} as unknown as DocumentNode<GetPaymentIntentsQuery, GetPaymentIntentsQueryVariables>;
export const UpdatePaymentMethodDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePaymentMethod"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePaymentMethod"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]} as unknown as DocumentNode<UpdatePaymentMethodMutation, UpdatePaymentMethodMutationVariables>;
export const GetBillingInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingInfo"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"prevMonth"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"thisMonth"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nextInvoiceDate"}}]}},{"kind":"Field","name":{"kind":"Name","value":"paymentMethods"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"billableStepTimeSeriesPrevMonth"},"name":{"kind":"Name","value":"billableStepTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"prevMonth"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"billableStepTimeSeriesThisMonth"},"name":{"kind":"Name","value":"billableStepTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"thisMonth"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillingInfoQuery, GetBillingInfoQueryVariables>;
export const GetSavedVercelProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSavedVercelProjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"savedVercelProjects"},"name":{"kind":"Name","value":"vercelApps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"path"}}]}}]}}]}}]} as unknown as DocumentNode<GetSavedVercelProjectsQuery, GetSavedVercelProjectsQueryVariables>;
export const CreateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<CreateVercelAppMutation, CreateVercelAppMutationVariables>;
export const UpdateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<UpdateVercelAppMutation, UpdateVercelAppMutationVariables>;
export const RemoveVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RemoveVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<RemoveVercelAppMutation, RemoveVercelAppMutationVariables>;
export const DeleteUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<DeleteUserMutation, DeleteUserMutationVariables>;
export const CreateUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateUser"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewUser"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createUser"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateUserMutation, CreateUserMutationVariables>;
export const GetUsersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetUsers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"users"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastLoginAt"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"session"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<GetUsersQuery, GetUsersQueryVariables>;
export const GetAccountSupportInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountSupportInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountSupportInfoQuery, GetAccountSupportInfoQueryVariables>;
export const GetAccountNameDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountName"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<GetAccountNameQuery, GetAccountNameQueryVariables>;
export const GetGlobalSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGlobalSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"opts"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"Variable","name":{"kind":"Name","value":"opts"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"env"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"functionID"},"name":{"kind":"Name","value":"workflowID"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetGlobalSearchQuery, GetGlobalSearchQueryVariables>;
export const GetFunctionSlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionSlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionSlugQuery, GetFunctionSlugQueryVariables>;
export const GetDeployssDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDeployss"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"checksum"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetDeployssQuery, GetDeployssQueryVariables>;
export const GetEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentsQuery, GetEnvironmentsQueryVariables>;
export const GetEventTypesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"functions"},"name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyVolume"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypesQuery, GetEventTypesQueryVariables>;
export const GetEventTypeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventType"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schemaSource"}},{"kind":"Field","name":{"kind":"Name","value":"integrationName"}},{"kind":"Field","name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"recent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"count"},"value":{"kind":"IntValue","value":"5"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"occurredAt"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"contactID"}},{"kind":"Field","name":{"kind":"Name","value":"contact"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"predefinedAttributes"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ingestSourceID"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeQuery, GetEventTypeQueryVariables>;
export const GetFunctionsUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionsUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsUsageQuery, GetFunctionsUsageQueryVariables>;
export const GetFunctionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"validTo"}},{"kind":"Field","name":{"kind":"Name","value":"workflowType"}},{"kind":"Field","name":{"kind":"Name","value":"throttlePeriod"}},{"kind":"Field","name":{"kind":"Name","value":"throttleCount"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflowID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsQuery, GetFunctionsQueryVariables>;
export const GetFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflowID"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"config"}},{"kind":"Field","name":{"kind":"Name","value":"retries"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"validTo"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionQuery, GetFunctionQueryVariables>;
export const GetFunctionVersionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionVersions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"FunctionVersion"}}]}},{"kind":"Field","name":{"kind":"Name","value":"previous"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"FunctionVersion"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FunctionVersion"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WorkflowVersion"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"validTo"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<GetFunctionVersionsQuery, GetFunctionVersionsQueryVariables>;
export const GetFunctionUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"asOf"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionUsageQuery, GetFunctionUsageQueryVariables>;
export const GetAllEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAllEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}}]}}]}}]} as unknown as DocumentNode<GetAllEnvironmentsQuery, GetAllEnvironmentsQueryVariables>;