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
  FilterType: string;
  IP: string;
  IngestSource: string;
  JSON: null | boolean | number | string | Record<string, unknown> | unknown[];
  Map: Record<string, unknown>;
  NullString: null | string;
  NullTime: null | string;
  Period: unknown;
  Role: unknown;
  Runtime: unknown;
  SchemaSource: unknown;
  SearchObject: unknown;
  Time: string;
  Timerange: unknown;
  ULID: string;
  UUID: string;
  Upload: unknown;
};

export type AwsMarketplaceSetupInput = {
  awsAccountID: Scalars['String'];
  customerID: Scalars['String'];
  productCode: Scalars['String'];
};

export type AwsMarketplaceSetupResponse = {
  __typename?: 'AWSMarketplaceSetupResponse';
  message: Scalars['String'];
};

export type Account = {
  __typename?: 'Account';
  billingEmail: Scalars['String'];
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  name: Maybe<Scalars['NullString']>;
  paymentIntents: Array<PaymentIntent>;
  paymentMethods: Maybe<Array<PaymentMethod>>;
  plan: Maybe<BillingPlan>;
  search: SearchResults;
  status: Scalars['String'];
  subscription: Maybe<BillingSubscription>;
  updatedAt: Scalars['Time'];
  users: Array<User>;
};


export type AccountSearchArgs = {
  opts: SearchInput;
};

export type App = {
  __typename?: 'App';
  archivedAt: Maybe<Scalars['Time']>;
  createdAt: Scalars['Time'];
  externalID: Scalars['String'];
  functionCount: Scalars['Int'];
  functions: Array<Workflow>;
  id: Scalars['UUID'];
  isArchived: Scalars['Boolean'];
  isParentArchived: Scalars['Boolean'];
  latestSync: Maybe<Deploy>;
  name: Scalars['String'];
  signingKeyRotationCheck: SigningKeyRotationCheck;
  syncs: Array<Deploy>;
};


export type AppLatestSyncArgs = {
  status: InputMaybe<SyncStatus>;
};


export type AppSyncsArgs = {
  after: InputMaybe<Scalars['Time']>;
  first?: Scalars['Int'];
};

export type ArchiveWorkflowInput = {
  archive: Scalars['Boolean'];
  workflowID: Scalars['ID'];
};

export type ArchivedEvent = {
  __typename?: 'ArchivedEvent';
  event: Scalars['Bytes'];
  eventModel: Event;
  eventVersion: EventType;
  functionRuns: Array<FunctionRun>;
  id: Scalars['ULID'];
  ingestSourceID: Maybe<Scalars['ID']>;
  name: Scalars['String'];
  occurredAt: Scalars['Time'];
  receivedAt: Scalars['Time'];
  source: Maybe<IngestKey>;
  version: Scalars['String'];
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

export type CancellationConfiguration = {
  __typename?: 'CancellationConfiguration';
  condition: Maybe<Scalars['String']>;
  event: Scalars['String'];
  timeout: Maybe<Scalars['String']>;
};

export type CodedError = {
  __typename?: 'CodedError';
  code: Scalars['String'];
  data: Maybe<Scalars['JSON']>;
  message: Scalars['String'];
};

export type ConcurrencyConfiguration = {
  __typename?: 'ConcurrencyConfiguration';
  key: Maybe<Scalars['String']>;
  limit: ConcurrencyLimitConfiguration;
  scope: ConcurrencyScope;
};

export type ConcurrencyLimitConfiguration = {
  __typename?: 'ConcurrencyLimitConfiguration';
  isPlanLimit: Maybe<Scalars['Boolean']>;
  value: Scalars['Int'];
};

export enum ConcurrencyScope {
  Account = 'ACCOUNT',
  Environment = 'ENVIRONMENT',
  Function = 'FUNCTION'
}

export type CreateFunctionReplayInput = {
  fromRange: Scalars['ULID'];
  name: Scalars['String'];
  statuses?: InputMaybe<Array<FunctionRunStatus>>;
  toRange: Scalars['ULID'];
  workflowID: Scalars['UUID'];
  workspaceID: Scalars['UUID'];
};

export type CreateStripeSubscriptionResponse = {
  __typename?: 'CreateStripeSubscriptionResponse';
  clientSecret: Scalars['String'];
  message: Scalars['String'];
};

export type CreateUserPayload = {
  __typename?: 'CreateUserPayload';
  user: Maybe<User>;
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

export type DebounceConfiguration = {
  __typename?: 'DebounceConfiguration';
  key: Maybe<Scalars['String']>;
  period: Scalars['String'];
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
  authorID: Maybe<Scalars['UUID']>;
  checksum: Scalars['String'];
  commitAuthor: Maybe<Scalars['String']>;
  commitHash: Maybe<Scalars['String']>;
  commitMessage: Maybe<Scalars['String']>;
  commitRef: Maybe<Scalars['String']>;
  createdAt: Scalars['Time'];
  deployedFunctions: Array<Workflow>;
  dupeCount: Scalars['Int'];
  error: Maybe<Scalars['String']>;
  framework: Maybe<Scalars['String']>;
  functionCount: Maybe<Scalars['Int']>;
  id: Scalars['UUID'];
  lastSyncedAt: Scalars['Time'];
  metadata: Scalars['Map'];
  platform: Maybe<Scalars['String']>;
  prevFunctionCount: Maybe<Scalars['Int']>;
  removedFunctions: Array<Workflow>;
  repoURL: Maybe<Scalars['String']>;
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  status: Scalars['String'];
  url: Maybe<Scalars['String']>;
  vercelDeploymentID: Maybe<Scalars['String']>;
  vercelDeploymentURL: Maybe<Scalars['String']>;
  vercelProjectID: Maybe<Scalars['String']>;
  vercelProjectURL: Maybe<Scalars['String']>;
  workspaceID: Scalars['UUID'];
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
  description: Maybe<Scalars['String']>;
  events: Maybe<EventConnection>;
  firstSeen: Maybe<Scalars['Time']>;
  integrationName: Maybe<Scalars['String']>;
  name: Scalars['String'];
  recent: Array<ArchivedEvent>;
  schemaSource: Maybe<Scalars['SchemaSource']>;
  usage: Usage;
  versionCount: Scalars['Int'];
  versions: Array<Maybe<EventType>>;
  workflows: Array<Workflow>;
  workspaceID: Maybe<Scalars['UUID']>;
};


export type EventEventsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: EventsFilter;
  first?: Scalars['Int'];
};


export type EventRecentArgs = {
  count: InputMaybe<Scalars['Int']>;
};


export type EventUsageArgs = {
  opts: InputMaybe<UsageInput>;
};


export type EventVersionsArgs = {
  versions: InputMaybe<Array<Scalars['String']>>;
};

export type EventConnection = {
  __typename?: 'EventConnection';
  edges: Maybe<Array<Maybe<EventEdge>>>;
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

export type EventSearchConnection = {
  __typename?: 'EventSearchConnection';
  edges: Maybe<Array<Maybe<EventSearchItemEdge>>>;
  pageInfo: PageInfo;
};

export type EventSearchFilter = {
  lowerTime: Scalars['Time'];
  query: Scalars['String'];
  upperTime: Scalars['Time'];
};

export type EventSearchItem = {
  __typename?: 'EventSearchItem';
  id: Scalars['ULID'];
  name: Scalars['String'];
  receivedAt: Scalars['Time'];
};

export type EventSearchItemEdge = {
  __typename?: 'EventSearchItemEdge';
  cursor: Scalars['String'];
  node: EventSearchItem;
};

export type EventType = {
  __typename?: 'EventType';
  createdAt: Maybe<Scalars['Time']>;
  cueType: Scalars['String'];
  id: Scalars['ID'];
  jsonSchema: Scalars['Map'];
  name: Scalars['String'];
  typescript: Scalars['String'];
  updatedAt: Maybe<Scalars['Time']>;
  version: Scalars['String'];
};

export type EventsBatchConfiguration = {
  __typename?: 'EventsBatchConfiguration';
  /** The maximum number of events a batch can have. */
  maxSize: Scalars['Int'];
  /** How long to wait before running the function with the batch. */
  timeout: Scalars['String'];
};

export type EventsFilter = {
  lowerTime: Scalars['Time'];
};

export type FilterList = {
  __typename?: 'FilterList';
  events: Maybe<Array<Scalars['String']>>;
  ips: Maybe<Array<Scalars['IP']>>;
  type: Maybe<Scalars['FilterType']>;
};

export type FilterListInput = {
  events?: InputMaybe<Array<Scalars['String']>>;
  ips?: InputMaybe<Array<Scalars['IP']>>;
  type?: InputMaybe<Scalars['FilterType']>;
};

export type FunctionConfiguration = {
  __typename?: 'FunctionConfiguration';
  cancellations: Maybe<Array<CancellationConfiguration>>;
  concurrency: Array<ConcurrencyConfiguration>;
  debounce: Maybe<DebounceConfiguration>;
  eventsBatch: Maybe<EventsBatchConfiguration>;
  priority: Maybe<Scalars['String']>;
  rateLimit: Maybe<RateLimitConfiguration>;
  retries: RetryConfiguration;
  throttle: Maybe<ThrottleConfiguration>;
};

export type FunctionReplay = {
  __typename?: 'FunctionReplay';
  createdAt: Scalars['Time'];
  endedAt: Scalars['Time'];
  id: Scalars['UUID'];
  name: Maybe<Scalars['String']>;
  scheduledRunCount: Scalars['Int'];
  totalRunCount: Maybe<Scalars['Int']>;
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  accountID: Scalars['UUID'];
  batchID: Maybe<Scalars['ULID']>;
  canRerun: Maybe<Scalars['Boolean']>;
  endedAt: Maybe<Scalars['Time']>;
  event: Maybe<ArchivedEvent>;
  eventID: Maybe<Scalars['ULID']>;
  events: Maybe<Array<ArchivedEvent>>;
  function: Workflow;
  history: Array<RunHistoryItem>;
  historyItemOutput: Maybe<Scalars['String']>;
  id: Scalars['ULID'];
  output: Maybe<Scalars['Bytes']>;
  startedAt: Scalars['Time'];
  status: FunctionRunStatus;
  workflowID: Scalars['UUID'];
  workflowVersion: Maybe<WorkflowVersion>;
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
  /** The function run has been scheduled. */
  Queued = 'QUEUED',
  /** The function run is currently running. */
  Running = 'RUNNING'
}

export enum FunctionRunTimeField {
  EndedAt = 'ENDED_AT',
  Mixed = 'MIXED',
  StartedAt = 'STARTED_AT'
}

export type FunctionRunV2 = {
  __typename?: 'FunctionRunV2';
  accountID: Scalars['UUID'];
  appID: Scalars['UUID'];
  batchCreatedAt: Maybe<Scalars['Time']>;
  cronSchedule: Maybe<Scalars['String']>;
  endedAt: Maybe<Scalars['Time']>;
  function: Maybe<Workflow>;
  functionID: Scalars['UUID'];
  id: Scalars['ULID'];
  isBatch: Scalars['Boolean'];
  output: Maybe<Scalars['Bytes']>;
  queuedAt: Scalars['Time'];
  sourceID: Maybe<Scalars['String']>;
  startedAt: Maybe<Scalars['Time']>;
  status: FunctionRunStatus;
  trace: Maybe<RunTraceSpan>;
  traceID: Scalars['String'];
  triggerIDs: Array<Scalars['ULID']>;
  triggers: Array<Scalars['Bytes']>;
  workspaceID: Scalars['UUID'];
};

export type FunctionRunV2Edge = {
  __typename?: 'FunctionRunV2Edge';
  cursor: Scalars['String'];
  node: FunctionRunV2;
};

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
  createdAt: Scalars['Time'];
  filter: FilterList;
  id: Scalars['ID'];
  metadata: Maybe<Scalars['Map']>;
  name: Scalars['NullString'];
  presharedKey: Scalars['String'];
  source: Scalars['IngestSource'];
  url: Maybe<Scalars['String']>;
};

export type IngestKeyFilter = {
  name?: InputMaybe<Scalars['String']>;
  source?: InputMaybe<Scalars['String']>;
};

export type InvokeStepInfo = {
  __typename?: 'InvokeStepInfo';
  functionID: Scalars['String'];
  returnEventID: Maybe<Scalars['ULID']>;
  runID: Maybe<Scalars['ULID']>;
  timedOut: Maybe<Scalars['Boolean']>;
  timeout: Scalars['Time'];
  triggeringEventID: Scalars['ULID'];
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
  archiveApp: App;
  archiveEnvironment: Workspace;
  archiveEvent: Maybe<Event>;
  archiveWorkflow: Maybe<WorkflowResponse>;
  cancelRun: FunctionRun;
  completeAWSMarketplaceSetup: Maybe<AwsMarketplaceSetupResponse>;
  createFunctionReplay: Replay;
  createIngestKey: IngestKey;
  createSigningKey: SigningKey;
  createStripeSubscription: CreateStripeSubscriptionResponse;
  createUser: Maybe<CreateUserPayload>;
  createVercelApp: Maybe<CreateVercelAppResponse>;
  createWorkspace: Array<Maybe<Workspace>>;
  deleteIngestKey: Maybe<DeleteResponse>;
  deleteSigningKey: SigningKey;
  disableEnvironmentAutoArchive: Workspace;
  editWorkflow: Maybe<WorkflowVersionResponse>;
  enableEnvironmentAutoArchive: Workspace;
  invokeFunction: Maybe<Scalars['Boolean']>;
  pauseFunction: Workflow;
  removeVercelApp: Maybe<RemoveVercelAppResponse>;
  resyncApp: SyncResponse;
  retryWorkflowRun: Maybe<StartWorkflowResponse>;
  rotateSigningKey: SigningKey;
  setUpAccount: Maybe<SetUpAccountPayload>;
  syncNewApp: SyncResponse;
  unarchiveApp: App;
  unarchiveEnvironment: Workspace;
  unpauseFunction: Workflow;
  updateAccount: Account;
  updateIngestKey: IngestKey;
  updatePaymentMethod: Maybe<Array<PaymentMethod>>;
  updatePlan: Account;
  updateVercelApp: Maybe<UpdateVercelAppResponse>;
};


export type MutationArchiveAppArgs = {
  id: Scalars['UUID'];
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


export type MutationCancelRunArgs = {
  envID: Scalars['UUID'];
  runID: Scalars['ULID'];
};


export type MutationCompleteAwsMarketplaceSetupArgs = {
  input: AwsMarketplaceSetupInput;
};


export type MutationCreateFunctionReplayArgs = {
  input: CreateFunctionReplayInput;
};


export type MutationCreateIngestKeyArgs = {
  input: NewIngestKey;
};


export type MutationCreateSigningKeyArgs = {
  envID: Scalars['UUID'];
};


export type MutationCreateStripeSubscriptionArgs = {
  input: StripeSubscriptionInput;
};


export type MutationCreateVercelAppArgs = {
  input: CreateVercelAppInput;
};


export type MutationCreateWorkspaceArgs = {
  input: NewWorkspaceInput;
};


export type MutationDeleteIngestKeyArgs = {
  input: DeleteIngestKey;
};


export type MutationDeleteSigningKeyArgs = {
  id: Scalars['UUID'];
};


export type MutationDisableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID'];
};


export type MutationEditWorkflowArgs = {
  input: EditWorkflowInput;
};


export type MutationEnableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID'];
};


export type MutationInvokeFunctionArgs = {
  data: InputMaybe<Scalars['Map']>;
  envID: Scalars['UUID'];
  functionSlug: Scalars['String'];
};


export type MutationPauseFunctionArgs = {
  fnID: Scalars['ID'];
};


export type MutationRemoveVercelAppArgs = {
  input: RemoveVercelAppInput;
};


export type MutationResyncAppArgs = {
  appExternalID: Scalars['String'];
  appURL: InputMaybe<Scalars['String']>;
  envID: Scalars['UUID'];
};


export type MutationRetryWorkflowRunArgs = {
  input: StartWorkflowInput;
  workflowRunID: Scalars['ULID'];
};


export type MutationRotateSigningKeyArgs = {
  envID: Scalars['UUID'];
};


export type MutationSyncNewAppArgs = {
  appURL: Scalars['String'];
  envID: Scalars['UUID'];
};


export type MutationUnarchiveAppArgs = {
  id: Scalars['UUID'];
};


export type MutationUnarchiveEnvironmentArgs = {
  id: Scalars['ID'];
};


export type MutationUnpauseFunctionArgs = {
  fnID: Scalars['ID'];
};


export type MutationUpdateAccountArgs = {
  input: UpdateAccount;
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


export type MutationUpdateVercelAppArgs = {
  input: UpdateVercelAppInput;
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

export type NewWorkspaceInput = {
  name: Scalars['String'];
};

/** The pagination information in a connection. */
export type PageInfo = {
  __typename?: 'PageInfo';
  /** When paginating forward, the cursor to query the next page. */
  endCursor: Maybe<Scalars['String']>;
  /** Indicates if there are any pages subsequent to the current page. */
  hasNextPage: Scalars['Boolean'];
  /** Indicates if there are any pages prior to the current page. */
  hasPreviousPage: Scalars['Boolean'];
  /** When paginating backward, the cursor to query the previous page. */
  startCursor: Maybe<Scalars['String']>;
};

export type PageResults = {
  __typename?: 'PageResults';
  cursor: Maybe<Scalars['String']>;
  page: Scalars['Int'];
  perPage: Scalars['Int'];
  totalItems: Maybe<Scalars['Int']>;
  totalPages: Maybe<Scalars['Int']>;
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
  invoiceURL: Maybe<Scalars['String']>;
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
  billableStepTimeSeries: Array<TimeSeries>;
  deploy: Deploy;
  deploys: Maybe<Array<Deploy>>;
  events: Maybe<PaginatedEvents>;
  plans: Array<Maybe<BillingPlan>>;
  session: Maybe<Session>;
  workspace: Workspace;
  workspaces: Maybe<Array<Workspace>>;
};


export type QueryBillableStepTimeSeriesArgs = {
  timeOptions: StepUsageTimeOptions;
};


export type QueryDeployArgs = {
  id: Scalars['ID'];
};


export type QueryDeploysArgs = {
  workspaceID: InputMaybe<Scalars['ID']>;
};


export type QueryEventsArgs = {
  query: InputMaybe<EventQuery>;
};


export type QueryWorkspaceArgs = {
  id: Scalars['ID'];
};

export type RateLimitConfiguration = {
  __typename?: 'RateLimitConfiguration';
  key: Maybe<Scalars['String']>;
  limit: Scalars['Int'];
  period: Scalars['String'];
};

export type RemoveVercelAppInput = {
  projectID: Scalars['String'];
  workspaceID: Scalars['ID'];
};

export type RemoveVercelAppResponse = {
  __typename?: 'RemoveVercelAppResponse';
  success: Scalars['Boolean'];
};

export type Replay = {
  __typename?: 'Replay';
  createdAt: Scalars['Time'];
  endedAt: Maybe<Scalars['Time']>;
  /**
   * The event or function ID that starts the replay range.
   *
   * This is not inclusive.
   *
   * A DateTime can also be used by generating an ULID from it.
   */
  fromRange: Scalars['ULID'];
  /** The number of function runs created scheduled from the replay. */
  functionRunsScheduledCount: Scalars['Int'];
  id: Scalars['ID'];
  name: Scalars['String'];
  replayType: ReplayType;
  /**
   * The event or function ID that ends the replay range.
   *
   * This is inclusive.
   *
   * A DateTime can also be used by generating an ULID from it.
   */
  toRange: Scalars['ULID'];
  /** The total number of function runs expected to be created from the replay. */
  totalRunCount: Maybe<Scalars['Int']>;
  workflowID: Maybe<Scalars['UUID']>;
  workspaceID: Maybe<Scalars['UUID']>;
};

export enum ReplayType {
  Event = 'EVENT',
  Function = 'FUNCTION'
}

export type RetryConfiguration = {
  __typename?: 'RetryConfiguration';
  isDefault: Maybe<Scalars['Boolean']>;
  value: Scalars['Int'];
};

export type RunHistoryCancel = {
  __typename?: 'RunHistoryCancel';
  eventID: Maybe<Scalars['ULID']>;
  expression: Maybe<Scalars['String']>;
  userID: Maybe<Scalars['UUID']>;
};

export type RunHistoryInvokeFunction = {
  __typename?: 'RunHistoryInvokeFunction';
  correlationID: Scalars['String'];
  eventID: Scalars['ULID'];
  functionID: Scalars['String'];
  timeout: Scalars['Time'];
};

export type RunHistoryInvokeFunctionResult = {
  __typename?: 'RunHistoryInvokeFunctionResult';
  eventID: Maybe<Scalars['ULID']>;
  runID: Maybe<Scalars['ULID']>;
  timeout: Scalars['Boolean'];
};

export type RunHistoryItem = {
  __typename?: 'RunHistoryItem';
  attempt: Scalars['Int'];
  cancel: Maybe<RunHistoryCancel>;
  createdAt: Scalars['Time'];
  functionVersion: Scalars['Int'];
  groupID: Maybe<Scalars['UUID']>;
  id: Scalars['ULID'];
  invokeFunction: Maybe<RunHistoryInvokeFunction>;
  invokeFunctionResult: Maybe<RunHistoryInvokeFunctionResult>;
  result: Maybe<RunHistoryResult>;
  sleep: Maybe<RunHistorySleep>;
  stepName: Maybe<Scalars['String']>;
  stepType: Maybe<HistoryStepType>;
  type: HistoryType;
  url: Maybe<Scalars['String']>;
  waitForEvent: Maybe<RunHistoryWaitForEvent>;
  waitResult: Maybe<RunHistoryWaitResult>;
};

export type RunHistoryResult = {
  __typename?: 'RunHistoryResult';
  durationMS: Scalars['Int'];
  errorCode: Maybe<Scalars['String']>;
  framework: Maybe<Scalars['String']>;
  platform: Maybe<Scalars['String']>;
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  sizeBytes: Scalars['Int'];
};

export type RunHistorySleep = {
  __typename?: 'RunHistorySleep';
  until: Scalars['Time'];
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
  expression: Maybe<Scalars['String']>;
  timeout: Scalars['Time'];
};

export type RunHistoryWaitResult = {
  __typename?: 'RunHistoryWaitResult';
  eventID: Maybe<Scalars['ULID']>;
  timeout: Scalars['Boolean'];
};

export type RunListConnection = {
  __typename?: 'RunListConnection';
  edges: Maybe<Array<Maybe<RunListItemEdge>>>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type RunListItem = {
  __typename?: 'RunListItem';
  endedAt: Maybe<Scalars['Time']>;
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

export type RunTraceSpan = {
  __typename?: 'RunTraceSpan';
  account: Account;
  accountID: Scalars['UUID'];
  appID: Scalars['UUID'];
  attempts: Maybe<Scalars['Int']>;
  childrenSpans: Array<RunTraceSpan>;
  duration: Maybe<Scalars['Int']>;
  endedAt: Maybe<Scalars['Time']>;
  functionID: Scalars['UUID'];
  isRoot: Scalars['Boolean'];
  name: Scalars['String'];
  outputID: Maybe<Scalars['String']>;
  parentSpan: Maybe<RunTraceSpan>;
  parentSpanID: Maybe<Scalars['String']>;
  queuedAt: Scalars['Time'];
  run: FunctionRun;
  runID: Scalars['ULID'];
  spanID: Scalars['String'];
  startedAt: Maybe<Scalars['Time']>;
  status: RunTraceSpanStatus;
  stepInfo: Maybe<StepInfo>;
  stepOp: Maybe<StepOp>;
  traceID: Scalars['String'];
  workspace: Workspace;
  workspaceID: Scalars['UUID'];
};

export type RunTraceSpanOutput = {
  __typename?: 'RunTraceSpanOutput';
  data: Maybe<Scalars['Bytes']>;
  error: Maybe<StepError>;
};

export enum RunTraceSpanStatus {
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Running = 'RUNNING',
  Waiting = 'WAITING'
}

export type RunTraceTrigger = {
  __typename?: 'RunTraceTrigger';
  IDs: Array<Scalars['ULID']>;
  batchID: Maybe<Scalars['ULID']>;
  cron: Maybe<Scalars['String']>;
  eventName: Maybe<Scalars['String']>;
  isBatch: Scalars['Boolean'];
  payloads: Array<Scalars['Bytes']>;
  timestamp: Scalars['Time'];
};

export type RunsConnection = {
  __typename?: 'RunsConnection';
  edges: Array<FunctionRunV2Edge>;
  pageInfo: PageInfo;
};

export type RunsFilter = {
  lowerTime: Scalars['Time'];
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<FunctionRunTimeField>;
  upperTime: Scalars['Time'];
};

export type RunsFilterV2 = {
  appIDs?: InputMaybe<Array<Scalars['UUID']>>;
  fnSlug?: InputMaybe<Scalars['String']>;
  from: Scalars['Time'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']>>;
  query?: InputMaybe<Scalars['String']>;
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<RunsOrderByField>;
  until?: InputMaybe<Scalars['Time']>;
};

export type RunsOrderBy = {
  direction: RunsOrderByDirection;
  field: RunsOrderByField;
};

export enum RunsOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export enum RunsOrderByField {
  EndedAt = 'ENDED_AT',
  QueuedAt = 'QUEUED_AT',
  StartedAt = 'STARTED_AT'
}

export enum SdkSigningKeyState {
  Correct = 'CORRECT',
  Incorrect = 'INCORRECT',
  Missing = 'MISSING',
  Unknown = 'UNKNOWN'
}

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

export type Session = {
  __typename?: 'Session';
  expires: Maybe<Scalars['Time']>;
  user: User;
};

export type SetUpAccountPayload = {
  __typename?: 'SetUpAccountPayload';
  account: Maybe<Account>;
};

export type SigningKey = {
  __typename?: 'SigningKey';
  createdAt: Scalars['Time'];
  decryptedValue: Scalars['String'];
  id: Scalars['UUID'];
  isActive: Scalars['Boolean'];
  user: Maybe<User>;
};

export type SigningKeyRotationCheck = {
  __typename?: 'SigningKeyRotationCheck';
  sdkSupport: Scalars['Boolean'];
  signingKeyFallbackState: SdkSigningKeyState;
  signingKeyState: SdkSigningKeyState;
};

export type SleepStepInfo = {
  __typename?: 'SleepStepInfo';
  sleepUntil: Scalars['Time'];
};

export type StartWorkflowInput = {
  workflowID: Scalars['ID'];
  workflowVersion?: InputMaybe<Scalars['Int']>;
  workspaceID: Scalars['ID'];
};

export type StartWorkflowResponse = {
  __typename?: 'StartWorkflowResponse';
  id: Scalars['ULID'];
};

export type StepError = {
  __typename?: 'StepError';
  message: Scalars['String'];
  name: Maybe<Scalars['String']>;
  stack: Maybe<Scalars['String']>;
};

export type StepInfo = InvokeStepInfo | SleepStepInfo | WaitForEventStepInfo;

export enum StepOp {
  Invoke = 'INVOKE',
  Run = 'RUN',
  Sleep = 'SLEEP',
  WaitForEvent = 'WAIT_FOR_EVENT'
}

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

export type SyncResponse = {
  __typename?: 'SyncResponse';
  app: Maybe<App>;
  error: Maybe<CodedError>;
  sync: Maybe<Deploy>;
};

export enum SyncStatus {
  Duplicate = 'duplicate',
  Error = 'error',
  Pending = 'pending',
  Success = 'success'
}

export type ThrottleConfiguration = {
  __typename?: 'ThrottleConfiguration';
  burst: Scalars['Int'];
  key: Maybe<Scalars['String']>;
  limit: Scalars['Int'];
  period: Scalars['String'];
};

export type TimeSeries = {
  __typename?: 'TimeSeries';
  data: Array<TimeSeriesPoint>;
  name: Scalars['String'];
};

export type TimeSeriesPoint = {
  __typename?: 'TimeSeriesPoint';
  time: Scalars['Time'];
  value: Maybe<Scalars['Float']>;
};

export type UpdateAccount = {
  billingEmail?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type UpdateIngestKey = {
  filterList?: InputMaybe<FilterListInput>;
  metadata?: InputMaybe<Scalars['Map']>;
  name?: InputMaybe<Scalars['String']>;
};

export type UpdateVercelAppInput = {
  path: Scalars['String'];
  projectID: Scalars['String'];
};

export type UpdateVercelAppResponse = {
  __typename?: 'UpdateVercelAppResponse';
  success: Scalars['Boolean'];
  vercelApp: Maybe<VercelApp>;
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
  account: Maybe<Account>;
  createdAt: Scalars['Time'];
  email: Scalars['String'];
  id: Scalars['ID'];
  lastLoginAt: Maybe<Scalars['Time']>;
  name: Maybe<Scalars['NullString']>;
  passwordChangedAt: Maybe<Scalars['Time']>;
  roles: Maybe<Array<Maybe<Scalars['Role']>>>;
  updatedAt: Scalars['Time'];
};

export type VercelApp = {
  __typename?: 'VercelApp';
  id: Scalars['UUID'];
  path: Maybe<Scalars['String']>;
  projectID: Scalars['String'];
  workspaceID: Scalars['UUID'];
};

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String'];
  expression: Maybe<Scalars['String']>;
  foundEventID: Maybe<Scalars['ULID']>;
  timedOut: Maybe<Scalars['Boolean']>;
  timeout: Scalars['Time'];
};

export type Workflow = {
  __typename?: 'Workflow';
  app: App;
  appName: Maybe<Scalars['String']>;
  archivedAt: Maybe<Scalars['Time']>;
  configuration: Maybe<FunctionConfiguration>;
  current: Maybe<WorkflowVersion>;
  failureHandler: Maybe<Workflow>;
  id: Scalars['ID'];
  isArchived: Scalars['Boolean'];
  isParentArchived: Scalars['Boolean'];
  isPaused: Scalars['Boolean'];
  latestVersion: WorkflowVersion;
  metrics: MetricsResponse;
  name: Scalars['String'];
  previous: Array<Maybe<WorkflowVersion>>;
  /**
   * A list of all the function’s replays.
   *
   * This doesn't include environment-level replays.
   */
  replays: Array<Replay>;
  run: FunctionRun;
  runs: Maybe<RunListConnection>;
  runsV2: Maybe<RunListConnection>;
  slug: Scalars['String'];
  url: Scalars['String'];
  usage: Usage;
};


export type WorkflowMetricsArgs = {
  opts: MetricsRequest;
};


export type WorkflowRunArgs = {
  id: Scalars['ULID'];
};


export type WorkflowRunsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: RunsFilter;
  first?: Scalars['Int'];
};


export type WorkflowRunsV2Args = {
  after: InputMaybe<Scalars['String']>;
  filter: RunsFilter;
  first?: Scalars['Int'];
};


export type WorkflowUsageArgs = {
  event: InputMaybe<Scalars['String']>;
  opts: InputMaybe<UsageInput>;
};

export type WorkflowResponse = {
  __typename?: 'WorkflowResponse';
  workflow: Workflow;
};

export type WorkflowTrigger = {
  __typename?: 'WorkflowTrigger';
  condition: Maybe<Scalars['NullString']>;
  eventName: Maybe<Scalars['NullString']>;
  schedule: Maybe<Scalars['NullString']>;
};

export type WorkflowVersion = {
  __typename?: 'WorkflowVersion';
  createdAt: Scalars['Time'];
  deploy: Maybe<Deploy>;
  description: Maybe<Scalars['NullString']>;
  retries: Scalars['Int'];
  throttleCount: Scalars['Int'];
  throttlePeriod: Scalars['String'];
  triggers: Array<WorkflowTrigger>;
  updatedAt: Scalars['Time'];
  url: Scalars['String'];
  validFrom: Maybe<Scalars['Time']>;
  validTo: Maybe<Scalars['Time']>;
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
  appByExternalID: App;
  apps: Array<App>;
  archivedEvent: Maybe<ArchivedEvent>;
  createdAt: Scalars['Time'];
  event: Maybe<Event>;
  eventByNames: Array<EventType>;
  eventSearch: EventSearchConnection;
  eventTypes: PaginatedEventTypes;
  events: PaginatedEvents;
  functionCount: Scalars['Int'];
  id: Scalars['ID'];
  ingestKey: IngestKey;
  ingestKeys: Array<IngestKey>;
  isArchived: Scalars['Boolean'];
  isAutoArchiveEnabled: Scalars['Boolean'];
  lastDeployedAt: Maybe<Scalars['Time']>;
  name: Scalars['String'];
  parentID: Maybe<Scalars['ID']>;
  run: Maybe<FunctionRunV2>;
  runTraceSpanOutputByID: RunTraceSpanOutput;
  runTrigger: RunTraceTrigger;
  runs: RunsConnection;
  runsMetrics: MetricsResponse;
  signingKeys: Array<SigningKey>;
  test: Scalars['Boolean'];
  type: EnvironmentType;
  unattachedSyncs: Array<Deploy>;
  vercelApps: Array<VercelApp>;
  webhookSigningKey: Scalars['String'];
  workflow: Maybe<Workflow>;
  workflowBySlug: Maybe<Workflow>;
  workflows: PaginatedWorkflows;
};


export type WorkspaceAppByExternalIdArgs = {
  externalID: Scalars['String'];
};


export type WorkspaceArchivedEventArgs = {
  id: Scalars['ULID'];
};


export type WorkspaceEventArgs = {
  name: Scalars['String'];
};


export type WorkspaceEventByNamesArgs = {
  names: Array<Scalars['String']>;
};


export type WorkspaceEventSearchArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: EventSearchFilter;
  first?: Scalars['Int'];
};


export type WorkspaceEventsArgs = {
  prefix: InputMaybe<Scalars['String']>;
};


export type WorkspaceIngestKeyArgs = {
  id: Scalars['ID'];
};


export type WorkspaceIngestKeysArgs = {
  filter: InputMaybe<IngestKeyFilter>;
};


export type WorkspaceRunArgs = {
  runID: Scalars['String'];
};


export type WorkspaceRunTraceSpanOutputByIdArgs = {
  outputID: Scalars['String'];
};


export type WorkspaceRunTriggerArgs = {
  runID: Scalars['String'];
};


export type WorkspaceRunsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: RunsFilterV2;
  first?: Scalars['Int'];
  orderBy: Array<RunsOrderBy>;
};


export type WorkspaceRunsMetricsArgs = {
  filter: RunsFilterV2;
};


export type WorkspaceUnattachedSyncsArgs = {
  after: InputMaybe<Scalars['Time']>;
  first?: Scalars['Int'];
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

export type SetUpAccountMutationVariables = Exact<{ [key: string]: never; }>;


export type SetUpAccountMutation = { __typename?: 'Mutation', setUpAccount: { __typename?: 'SetUpAccountPayload', account: { __typename?: 'Account', id: string } | null } | null };

export type CreateUserMutationVariables = Exact<{ [key: string]: never; }>;


export type CreateUserMutation = { __typename?: 'Mutation', createUser: { __typename?: 'CreateUserPayload', user: { __typename?: 'User', id: string } | null } | null };

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

export type AchiveAppMutationVariables = Exact<{
  appID: Scalars['UUID'];
}>;


export type AchiveAppMutation = { __typename?: 'Mutation', archiveApp: { __typename?: 'App', id: string } };

export type UnachiveAppMutationVariables = Exact<{
  appID: Scalars['UUID'];
}>;


export type UnachiveAppMutation = { __typename?: 'Mutation', unarchiveApp: { __typename?: 'App', id: string } };

export type ResyncAppMutationVariables = Exact<{
  appExternalID: Scalars['String'];
  appURL: InputMaybe<Scalars['String']>;
  envID: Scalars['UUID'];
}>;


export type ResyncAppMutation = { __typename?: 'Mutation', resyncApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type SyncQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
  syncID: Scalars['ID'];
}>;


export type SyncQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, name: string } }, sync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> } };

export type AppSyncsQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type AppSyncsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, syncs: Array<{ __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> }> } } };

export type AppQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type AppQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, name: string, functions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string, latestVersion: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } }>, latestSync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null } | null } } };

export type AppNavDataQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type AppNavDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, isArchived: boolean, isParentArchived: boolean, name: string, latestSync: { __typename?: 'Deploy', platform: string | null, url: string | null } | null } } };

export type SyncNewAppMutationVariables = Exact<{
  appURL: Scalars['String'];
  envID: Scalars['UUID'];
}>;


export type SyncNewAppMutation = { __typename?: 'Mutation', syncNewApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', externalID: string, id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type AppsQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type AppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', id: string, externalID: string, functionCount: number, isArchived: boolean, name: string, latestSync: { __typename?: 'Deploy', error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null } | null }>, unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string }> } };

export type SearchEventsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  lowerTime: Scalars['Time'];
  query: Scalars['String'];
  upperTime: Scalars['Time'];
}>;


export type SearchEventsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', id: string, eventSearch: { __typename?: 'EventSearchConnection', edges: Array<{ __typename?: 'EventSearchItemEdge', node: { __typename?: 'EventSearchItem', id: string, name: string, receivedAt: string } } | null> | null, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

export type GetEventSearchEventQueryVariables = Exact<{
  envID: Scalars['ID'];
  eventID: Scalars['ULID'];
}>;


export type GetEventSearchEventQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', event: { __typename?: 'ArchivedEvent', id: string, name: string, receivedAt: string, payload: string, runs: Array<{ __typename?: 'FunctionRun', id: string, output: string | null, status: FunctionRunStatus, function: { __typename?: 'Workflow', id: string, name: string } }> } | null } };

export type GetEventSearchRunQueryVariables = Exact<{
  envID: Scalars['ID'];
  functionID: Scalars['ID'];
  runID: Scalars['ULID'];
}>;


export type GetEventSearchRunQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', name: string, run: { __typename?: 'FunctionRun', canRerun: boolean | null, id: string, status: FunctionRunStatus, startedAt: string, endedAt: string | null, output: string | null, history: Array<{ __typename?: 'RunHistoryItem', attempt: number, createdAt: string, functionVersion: number, groupID: string | null, id: string, stepName: string | null, type: HistoryType, url: string | null, cancel: { __typename?: 'RunHistoryCancel', eventID: string | null, expression: string | null, userID: string | null } | null, sleep: { __typename?: 'RunHistorySleep', until: string } | null, waitForEvent: { __typename?: 'RunHistoryWaitForEvent', eventName: string, expression: string | null, timeout: string } | null, waitResult: { __typename?: 'RunHistoryWaitResult', eventID: string | null, timeout: boolean } | null }>, version: { __typename?: 'WorkflowVersion', url: string, validFrom: string | null, version: number, deploy: { __typename?: 'Deploy', id: string, createdAt: string } | null, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } | null } } | null } };

export type GetEventKeysForBlankSlateQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetEventKeysForBlankSlateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', name: null | string, presharedKey: string, createdAt: string }> } };

export type ArchiveEventMutationVariables = Exact<{
  environmentId: Scalars['ID'];
  name: Scalars['String'];
}>;


export type ArchiveEventMutation = { __typename?: 'Mutation', archiveEvent: { __typename?: 'Event', name: string } | null };

export type GetLatestEventLogsQueryVariables = Exact<{
  name: InputMaybe<Scalars['String']>;
  environmentID: Scalars['ID'];
}>;


export type GetLatestEventLogsQuery = { __typename?: 'Query', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', recent: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string, event: string, source: { __typename?: 'IngestKey', name: null | string } | null }> }> } | null };

export type GetEventKeysQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetEventKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventKeys: Array<{ __typename?: 'IngestKey', name: null | string, value: string }> } };

export type GetEventLogQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  eventName: Scalars['String'];
  cursor: InputMaybe<Scalars['String']>;
  perPage: Scalars['Int'];
}>;


export type GetEventLogQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventType: { __typename?: 'Event', events: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string }> } | null } };

export type EventPayloadFragment = { __typename?: 'ArchivedEvent', payload: string } & { ' $fragmentName'?: 'EventPayloadFragment' };

export type GetFunctionRunCardQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunCardQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', name: string, slug: string, run: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus, startedAt: string } } | null } };

export type GetEventQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  eventID: Scalars['ULID'];
}>;


export type GetEventQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', event: (
      { __typename?: 'ArchivedEvent', receivedAt: string, functionRuns: Array<{ __typename?: 'FunctionRun', id: string, function: { __typename?: 'Workflow', id: string } }> }
      & { ' $fragmentRefs'?: { 'EventPayloadFragment': EventPayloadFragment } }
    ) | null } };

export type GetBillingPlanQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingPlanQuery = { __typename?: 'Query', account: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, name: string, features: Record<string, unknown> } | null }, plans: Array<{ __typename?: 'BillingPlan', name: string, features: Record<string, unknown> } | null> };

export type GetFunctionRateLimitDocumentQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionRateLimitDocumentQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', ratelimit: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFunctionRunsMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionRunsMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', completed: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, canceled: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, failed: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetFnMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFnMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFailedFunctionRunsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  lowerTime: Scalars['Time'];
  upperTime: Scalars['Time'];
}>;


export type GetFailedFunctionRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', failedRuns: { __typename?: 'RunListConnection', edges: Array<{ __typename?: 'RunListItemEdge', node: { __typename?: 'RunListItem', id: string, endedAt: string | null } } | null> | null } | null } | null } };

export type GetSdkRequestMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetSdkRequestMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetStepBacklogMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetStepBacklogMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', scheduled: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, sleeping: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetStepsRunningMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  fnSlug: Scalars['String'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetStepsRunningMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', running: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, concurrencyLimit: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFunctionArchivalQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionArchivalQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow: { __typename?: 'Workflow', id: string, isArchived: boolean, name: string } | null } };

export type GetFunctionVersionNumberQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionVersionNumberQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow: { __typename?: 'Workflow', id: string, isPaused: boolean, name: string, archivedAt: string | null, current: { __typename?: 'WorkflowVersion', version: number } | null, previous: Array<{ __typename?: 'WorkflowVersion', version: number } | null> } | null } };

export type PauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID'];
}>;


export type PauseFunctionMutation = { __typename?: 'Mutation', pauseFunction: { __typename?: 'Workflow', id: string } };

export type UnpauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID'];
}>;


export type UnpauseFunctionMutation = { __typename?: 'Mutation', unpauseFunction: { __typename?: 'Workflow', id: string } };

export type InvokeFunctionMutationVariables = Exact<{
  envID: Scalars['UUID'];
  data: InputMaybe<Scalars['Map']>;
  functionSlug: Scalars['String'];
}>;


export type InvokeFunctionMutation = { __typename?: 'Mutation', invokeFunction: boolean | null };

export type RerunFunctionRunMutationVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
  functionRunID: Scalars['ULID'];
}>;


export type RerunFunctionRunMutation = { __typename?: 'Mutation', retryWorkflowRun: { __typename?: 'StartWorkflowResponse', id: string } | null };

export type GetHistoryItemOutputQueryVariables = Exact<{
  envID: Scalars['ID'];
  functionID: Scalars['ID'];
  historyItemID: Scalars['ULID'];
  runID: Scalars['ULID'];
}>;


export type GetHistoryItemOutputQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', run: { __typename?: 'FunctionRun', historyItemOutput: string | null } } | null } };

export type GetFunctionRunDetailsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunID: Scalars['ULID'];
}>;


export type GetFunctionRunDetailsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, name: string, slug: string, run: { __typename?: 'FunctionRun', batchID: string | null, canRerun: boolean | null, id: string, status: FunctionRunStatus, startedAt: string, endedAt: string | null, output: string | null, events: Array<{ __typename?: 'ArchivedEvent', id: string, name: string, receivedAt: string, payload: string }> | null, history: Array<{ __typename?: 'RunHistoryItem', attempt: number, createdAt: string, functionVersion: number, groupID: string | null, id: string, stepName: string | null, type: HistoryType, url: string | null, cancel: { __typename?: 'RunHistoryCancel', eventID: string | null, expression: string | null, userID: string | null } | null, sleep: { __typename?: 'RunHistorySleep', until: string } | null, waitForEvent: { __typename?: 'RunHistoryWaitForEvent', eventName: string, expression: string | null, timeout: string } | null, waitResult: { __typename?: 'RunHistoryWaitResult', eventID: string | null, timeout: boolean } | null, invokeFunction: { __typename?: 'RunHistoryInvokeFunction', eventID: string, functionID: string, correlationID: string, timeout: string } | null, invokeFunctionResult: { __typename?: 'RunHistoryInvokeFunctionResult', eventID: string | null, timeout: boolean, runID: string | null } | null }>, version: { __typename?: 'WorkflowVersion', url: string, validFrom: string | null, version: number, deploy: { __typename?: 'Deploy', id: string, createdAt: string } | null, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } | null } } | null } };

export type GetFunctionRunTriggersQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
}>;


export type GetFunctionRunTriggersQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', schedule: null | string | null }> } | null } | null } };

export type GetFunctionRunsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunStatuses: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  functionRunCursor: InputMaybe<Scalars['String']>;
  timeRangeStart: Scalars['Time'];
  timeRangeEnd: Scalars['Time'];
  timeField: FunctionRunTimeField;
}>;


export type GetFunctionRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, runs: { __typename?: 'RunListConnection', edges: Array<{ __typename?: 'RunListItemEdge', node: { __typename?: 'RunListItem', id: string, status: FunctionRunStatus, startedAt: string, endedAt: string | null } } | null> | null, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor: string | null } } | null } | null } };

export type GetFunctionEndedRunsCountQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  timeRangeStart: Scalars['Time'];
  timeRangeEnd: Scalars['Time'];
}>;


export type GetFunctionEndedRunsCountQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, failedRuns: { __typename?: 'RunListConnection', totalCount: number } | null, canceledRuns: { __typename?: 'RunListConnection', totalCount: number } | null, succeededRuns: { __typename?: 'RunListConnection', totalCount: number } | null } | null } };

export type CreateFunctionReplayMutationVariables = Exact<{
  environmentID: Scalars['UUID'];
  functionID: Scalars['UUID'];
  name: Scalars['String'];
  fromRange: Scalars['ULID'];
  toRange: Scalars['ULID'];
  statuses: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
}>;


export type CreateFunctionReplayMutation = { __typename?: 'Mutation', createFunctionReplay: { __typename?: 'Replay', id: string } };

export type GetFunctionRunsCountQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  functionRunStatuses: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeRangeStart: Scalars['Time'];
  timeRangeEnd: Scalars['Time'];
  timeField: FunctionRunTimeField;
}>;


export type GetFunctionRunsCountQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, runs: { __typename?: 'RunListConnection', totalCount: number } | null } | null } };

export type GetReplaysQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
}>;


export type GetReplaysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', id: string, function: { __typename?: 'Workflow', id: string, replays: Array<{ __typename?: 'Replay', id: string, name: string, createdAt: string, endedAt: string | null, functionRunsScheduledCount: number }> } | null } };

export type GetRunsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsOrderByField;
  functionSlug: Scalars['String'];
  functionRunCursor?: InputMaybe<Scalars['String']>;
}>;


export type GetRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', runs: { __typename?: 'RunsConnection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', id: string, queuedAt: string, endedAt: string | null, startedAt: string | null, status: FunctionRunStatus } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

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


export type UpdateIngestKeyMutation = { __typename?: 'Mutation', updateIngestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url: string | null, metadata: Record<string, unknown> | null, filter: { __typename?: 'FilterList', type: string | null, ips: Array<string> | null, events: Array<string> | null } } };

export type DeleteEventKeyMutationVariables = Exact<{
  input: DeleteIngestKey;
}>;


export type DeleteEventKeyMutation = { __typename?: 'Mutation', deleteIngestKey: { __typename?: 'DeleteResponse', ids: Array<string> } | null };

export type GetIngestKeyQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  keyID: Scalars['ID'];
}>;


export type GetIngestKeyQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url: string | null, metadata: Record<string, unknown> | null, source: string, filter: { __typename?: 'FilterList', type: string | null, ips: Array<string> | null, events: Array<string> | null } } } };

export type CreateSigningKeyMutationVariables = Exact<{
  envID: Scalars['UUID'];
}>;


export type CreateSigningKeyMutation = { __typename?: 'Mutation', createSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type DeleteSigningKeyMutationVariables = Exact<{
  signingKeyID: Scalars['UUID'];
}>;


export type DeleteSigningKeyMutation = { __typename?: 'Mutation', deleteSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type RotateSigningKeyMutationVariables = Exact<{
  envID: Scalars['UUID'];
}>;


export type RotateSigningKeyMutation = { __typename?: 'Mutation', rotateSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type GetSigningKeyQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetSigningKeyQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', webhookSigningKey: string } };

export type GetSigningKeysQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type GetSigningKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', signingKeys: Array<{ __typename?: 'SigningKey', createdAt: string, decryptedValue: string, id: string, isActive: boolean, user: { __typename?: 'User', email: string, name: null | string | null } | null }> } };

export type UnattachedSyncQueryVariables = Exact<{
  syncID: Scalars['ID'];
}>;


export type UnattachedSyncQuery = { __typename?: 'Query', sync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> } };

export type UnattachedSyncsQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type UnattachedSyncsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', syncs: Array<{ __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null }> } };

export type GetBillableStepsQueryVariables = Exact<{
  month: Scalars['Int'];
  year: Scalars['Int'];
}>;


export type GetBillableStepsQuery = { __typename?: 'Query', billableStepTimeSeries: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type UpdateAccountMutationVariables = Exact<{
  input: UpdateAccount;
}>;


export type UpdateAccountMutation = { __typename?: 'Mutation', account: { __typename?: 'Account', billingEmail: string, name: null | string | null } };

export type CreateStripeSubscriptionMutationVariables = Exact<{
  input: StripeSubscriptionInput;
}>;


export type CreateStripeSubscriptionMutation = { __typename?: 'Mutation', createStripeSubscription: { __typename?: 'CreateStripeSubscriptionResponse', clientSecret: string, message: string } };

export type UpdatePlanMutationVariables = Exact<{
  planID: Scalars['ID'];
}>;


export type UpdatePlanMutation = { __typename?: 'Mutation', updatePlan: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, name: string } | null } };

export type GetPaymentIntentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPaymentIntentsQuery = { __typename?: 'Query', account: { __typename?: 'Account', paymentIntents: Array<{ __typename?: 'PaymentIntent', status: string, createdAt: string, amountLabel: string, description: string, invoiceURL: string | null }> } };

export type UpdatePaymentMethodMutationVariables = Exact<{
  token: Scalars['String'];
}>;


export type UpdatePaymentMethodMutation = { __typename?: 'Mutation', updatePaymentMethod: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null };

export type GetBillingInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', billingEmail: string, name: null | string | null, plan: { __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, features: Record<string, unknown> } | null, subscription: { __typename?: 'BillingSubscription', nextInvoiceDate: string } | null, paymentMethods: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null }, plans: Array<{ __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, features: Record<string, unknown> } | null> };

export type GetSavedVercelProjectsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetSavedVercelProjectsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', savedVercelProjects: Array<{ __typename?: 'VercelApp', id: string, projectID: string, path: string | null, workspaceID: string }> } };

export type CreateVercelAppMutationVariables = Exact<{
  input: CreateVercelAppInput;
}>;


export type CreateVercelAppMutation = { __typename?: 'Mutation', createVercelApp: { __typename?: 'CreateVercelAppResponse', success: boolean } | null };

export type UpdateVercelAppMutationVariables = Exact<{
  input: UpdateVercelAppInput;
}>;


export type UpdateVercelAppMutation = { __typename?: 'Mutation', updateVercelApp: { __typename?: 'UpdateVercelAppResponse', success: boolean } | null };

export type RemoveVercelAppMutationVariables = Exact<{
  input: RemoveVercelAppInput;
}>;


export type RemoveVercelAppMutation = { __typename?: 'Mutation', removeVercelApp: { __typename?: 'RemoveVercelAppResponse', success: boolean } | null };

export type CreateWebhookMutationVariables = Exact<{
  input: NewIngestKey;
}>;


export type CreateWebhookMutation = { __typename?: 'Mutation', key: { __typename?: 'IngestKey', id: string, url: string | null } };

export type CompleteAwsMarketplaceSetupMutationVariables = Exact<{
  input: AwsMarketplaceSetupInput;
}>;


export type CompleteAwsMarketplaceSetupMutation = { __typename?: 'Mutation', completeAWSMarketplaceSetup: { __typename?: 'AWSMarketplaceSetupResponse', message: string } | null };

export type GetAccountSupportInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAccountSupportInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, plan: { __typename?: 'BillingPlan', id: string, name: string, amount: number, features: Record<string, unknown> } | null } };

export type GetArchivedAppBannerDataQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type GetArchivedAppBannerDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', isArchived: boolean } } };

export type GetGlobalSearchQueryVariables = Exact<{
  opts: SearchInput;
}>;


export type GetGlobalSearchQuery = { __typename?: 'Query', account: { __typename?: 'Account', search: { __typename?: 'SearchResults', results: Array<{ __typename?: 'SearchResult', kind: SearchResultType, env: { __typename?: 'Workspace', name: string, id: string, type: EnvironmentType }, value: { __typename?: 'ArchivedEvent', id: string, name: string } | { __typename?: 'FunctionRun', id: string, functionID: string } } | null> } } };

export type GetFunctionSlugQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
}>;


export type GetFunctionSlugQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', slug: string, name: string } | null } };

export type GetRunTraceTriggerQueryVariables = Exact<{
  envID: Scalars['ID'];
  runID: Scalars['String'];
}>;


export type GetRunTraceTriggerQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runTrigger: { __typename?: 'RunTraceTrigger', IDs: Array<string>, payloads: Array<string>, timestamp: string, eventName: string | null, isBatch: boolean, batchID: string | null, cron: string | null } } };

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, outputID: string | null, spanID: string, stepOp: StepOp | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null } | { __typename: 'SleepStepInfo', sleepUntil: string } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null } | null } & { ' $fragmentName'?: 'TraceDetailsFragment' };

export type GetRunTraceQueryVariables = Exact<{
  envID: Scalars['ID'];
  runID: Scalars['String'];
}>;


export type GetRunTraceQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', run: { __typename?: 'FunctionRunV2', function: { __typename?: 'Workflow', id: string, name: string, app: { __typename?: 'App', name: string, externalID: string } } | null, trace: (
        { __typename?: 'RunTraceSpan', childrenSpans: Array<(
          { __typename?: 'RunTraceSpan', childrenSpans: Array<(
            { __typename?: 'RunTraceSpan' }
            & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
          )> }
          & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
        )> }
        & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
      ) | null } | null } };

export type GetDeployssQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetDeployssQuery = { __typename?: 'Query', deploys: Array<{ __typename?: 'Deploy', id: string, appName: string, authorID: string | null, checksum: string, createdAt: string, error: string | null, framework: string | null, metadata: Record<string, unknown>, sdkLanguage: string, sdkVersion: string, status: string, deployedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }>, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }> }> | null };

export type GetEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetEnvironmentsQuery = { __typename?: 'Query', workspaces: Array<{ __typename?: 'Workspace', id: string, name: string, parentID: string | null, test: boolean, type: EnvironmentType, webhookSigningKey: string, createdAt: string, isArchived: boolean, functionCount: number, isAutoArchiveEnabled: boolean, lastDeployedAt: string | null }> | null };

export type GetEventTypesQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
}>;


export type GetEventTypesQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, functions: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string }>, dailyVolume: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null } } } };

export type GetEventTypeQueryVariables = Exact<{
  eventName: InputMaybe<Scalars['String']>;
  environmentID: Scalars['ID'];
}>;


export type GetEventTypeQuery = { __typename?: 'Query', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, usage: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, workflows: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string, current: { __typename?: 'WorkflowVersion', createdAt: string } | null }> }> } | null };

export type GetFunctionsUsageQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
  archived: InputMaybe<Scalars['Boolean']>;
  pageSize: InputMaybe<Scalars['Int']>;
}>;


export type GetFunctionsUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, dailyStarts: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyFailures: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }> } } };

export type GetFunctionsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
  archived: InputMaybe<Scalars['Boolean']>;
  pageSize: InputMaybe<Scalars['Int']>;
}>;


export type GetFunctionsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', appName: string | null, id: string, slug: string, name: string, isPaused: boolean, isArchived: boolean, current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } | null }> } } };

export type GetFunctionQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', id: string, workflow: { __typename?: 'Workflow', id: string, name: string, slug: string, isPaused: boolean, isArchived: boolean, appName: string | null, current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null, condition: null | string | null }>, deploy: { __typename?: 'Deploy', id: string, createdAt: string } | null } | null, failureHandler: { __typename?: 'Workflow', slug: string, name: string } | null, configuration: { __typename?: 'FunctionConfiguration', priority: string | null, cancellations: Array<{ __typename?: 'CancellationConfiguration', event: string, timeout: string | null, condition: string | null }> | null, retries: { __typename?: 'RetryConfiguration', value: number, isDefault: boolean | null }, eventsBatch: { __typename?: 'EventsBatchConfiguration', maxSize: number, timeout: string } | null, concurrency: Array<{ __typename?: 'ConcurrencyConfiguration', scope: ConcurrencyScope, key: string | null, limit: { __typename?: 'ConcurrencyLimitConfiguration', value: number, isPlanLimit: boolean | null } }>, rateLimit: { __typename?: 'RateLimitConfiguration', limit: number, period: string, key: string | null } | null, debounce: { __typename?: 'DebounceConfiguration', period: string, key: string | null } | null, throttle: { __typename?: 'ThrottleConfiguration', burst: number, key: string | null, limit: number, period: string } | null } | null } | null } };

export type GetFunctionUsageQueryVariables = Exact<{
  id: Scalars['ID'];
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow: { __typename?: 'Workflow', dailyStarts: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyFailures: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetAllEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAllEnvironmentsQuery = { __typename?: 'Query', workspaces: Array<{ __typename?: 'Workspace', id: string, name: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean }> | null };

export type CancelRunMutationVariables = Exact<{
  envID: Scalars['UUID'];
  runID: Scalars['ULID'];
}>;


export type CancelRunMutation = { __typename?: 'Mutation', cancelRun: { __typename?: 'FunctionRun', id: string } };

export const EventPayloadFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<EventPayloadFragment, unknown>;
export const TraceDetailsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<TraceDetailsFragment, unknown>;
export const SetUpAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SetUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"setUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<SetUpAccountMutation, SetUpAccountMutationVariables>;
export const CreateUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<CreateUserMutation, CreateUserMutationVariables>;
export const CreateEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createWorkspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateEnvironmentMutation, CreateEnvironmentMutationVariables>;
export const ArchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<ArchiveEnvironmentMutation, ArchiveEnvironmentMutationVariables>;
export const UnarchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnarchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnarchiveEnvironmentMutation, UnarchiveEnvironmentMutationVariables>;
export const DisableEnvironmentAutoArchiveDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisableEnvironmentAutoArchiveDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<DisableEnvironmentAutoArchiveDocumentMutation, DisableEnvironmentAutoArchiveDocumentMutationVariables>;
export const EnableEnvironmentAutoArchiveDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EnableEnvironmentAutoArchive"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<EnableEnvironmentAutoArchiveMutation, EnableEnvironmentAutoArchiveMutationVariables>;
export const AchiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AchiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<AchiveAppMutation, AchiveAppMutationVariables>;
export const UnachiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnachiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnachiveAppMutation, UnachiveAppMutationVariables>;
export const ResyncAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResyncApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resyncApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appExternalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}}},{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<ResyncAppMutation, ResyncAppMutationVariables>;
export const SyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Sync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<SyncQuery, SyncQueryVariables>;
export const AppSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"syncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppSyncsQuery, AppSyncsQueryVariables>;
export const AppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"App"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"latestVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppQuery, AppQueryVariables>;
export const AppNavDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppNavData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isParentArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<AppNavDataQuery, AppNavDataQueryVariables>;
export const SyncNewAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncNewApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncNewApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<SyncNewAppMutation, SyncNewAppMutationVariables>;
export const AppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Apps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}}]}}]} as unknown as DocumentNode<AppsQuery, AppsQueryVariables>;
export const SearchEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"eventSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<SearchEventsQuery, SearchEventsQueryVariables>;
export const GetEventSearchEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventSearchEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"event"},"name":{"kind":"Name","value":"archivedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"functionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventSearchEventQuery, GetEventSearchEventQueryVariables>;
export const GetEventSearchRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventSearchRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"canRerun"}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"attempt"}},{"kind":"Field","name":{"kind":"Name","value":"cancel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionVersion"}},{"kind":"Field","name":{"kind":"Name","value":"groupID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"sleep"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"until"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"waitForEvent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"waitResult"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","alias":{"kind":"Name","value":"version"},"name":{"kind":"Name","value":"workflowVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventSearchRunQuery, GetEventSearchRunQueryVariables>;
export const GetEventKeysForBlankSlateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeysForBlankSlate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"source"},"value":{"kind":"StringValue","value":"key","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysForBlankSlateQuery, GetEventKeysForBlankSlateQueryVariables>;
export const ArchiveEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}}},{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<ArchiveEventMutation, ArchiveEventMutationVariables>;
export const GetLatestEventLogsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLatestEventLogs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"recent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"count"},"value":{"kind":"IntValue","value":"5"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetLatestEventLogsQuery, GetLatestEventLogsQueryVariables>;
export const GetEventKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventKeys"},"name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"value"},"name":{"kind":"Name","value":"presharedKey"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysQuery, GetEventKeysQueryVariables>;
export const GetEventLogDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventLog"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventType"},"name":{"kind":"Name","value":"event"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"events"},"name":{"kind":"Name","value":"recent"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"cursored"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"cursor"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventLogQuery, GetEventLogQueryVariables>;
export const GetFunctionRunCardDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunCard"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunCardQuery, GetFunctionRunCardQueryVariables>;
export const GetEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"event"},"name":{"kind":"Name","value":"archivedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"EventPayload"}},{"kind":"Field","name":{"kind":"Name","value":"functionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<GetEventQuery, GetEventQueryVariables>;
export const GetBillingPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingPlan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]} as unknown as DocumentNode<GetBillingPlanQuery, GetBillingPlanQueryVariables>;
export const GetFunctionRateLimitDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRateLimitDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"ratelimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_rate_limited_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRateLimitDocumentQuery, GetFunctionRateLimitDocumentQueryVariables>;
export const GetFunctionRunsMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"canceled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"failed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsMetricsQuery, GetFunctionRunsMetricsQueryVariables>;
export const GetFnMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnMetricsQuery, GetFnMetricsQueryVariables>;
export const GetFailedFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFailedFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"failedRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"FAILED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"ENDED_AT"}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFailedFunctionRunsQuery, GetFailedFunctionRunsQueryVariables>;
export const GetSdkRequestMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSDKRequestMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSdkRequestMetricsQuery, GetSdkRequestMetricsQueryVariables>;
export const GetStepBacklogMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepBacklogMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"scheduled"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_scheduled","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sleeping"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_sleeping","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepBacklogMetricsQuery, GetStepBacklogMetricsQueryVariables>;
export const GetStepsRunningMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepsRunningMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"running"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"concurrencyLimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepsRunningMetricsQuery, GetStepsRunningMetricsQueryVariables>;
export const GetFunctionArchivalDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionArchival"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionArchivalQuery, GetFunctionArchivalQueryVariables>;
export const GetFunctionVersionNumberDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionVersionNumber"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}}]}},{"kind":"Field","name":{"kind":"Name","value":"previous"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionVersionNumberQuery, GetFunctionVersionNumberQueryVariables>;
export const PauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<PauseFunctionMutation, PauseFunctionMutationVariables>;
export const UnpauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnpauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unpauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnpauseFunctionMutation, UnpauseFunctionMutationVariables>;
export const InvokeFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"InvokeFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"data"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"invokeFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"data"},"value":{"kind":"Variable","name":{"kind":"Name","value":"data"}}},{"kind":"Argument","name":{"kind":"Name","value":"functionSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}]}]}}]} as unknown as DocumentNode<InvokeFunctionMutation, InvokeFunctionMutationVariables>;
export const RerunFunctionRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RerunFunctionRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryWorkflowRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"workflowRunID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<RerunFunctionRunMutation, RerunFunctionRunMutationVariables>;
export const GetHistoryItemOutputDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetHistoryItemOutput"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"historyItemOutput"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}}}]}]}}]}}]}}]}}]} as unknown as DocumentNode<GetHistoryItemOutputQuery, GetHistoryItemOutputQueryVariables>;
export const GetFunctionRunDetailsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunDetails"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchID"}},{"kind":"Field","name":{"kind":"Name","value":"canRerun"}},{"kind":"Field","name":{"kind":"Name","value":"events"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"attempt"}},{"kind":"Field","name":{"kind":"Name","value":"cancel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionVersion"}},{"kind":"Field","name":{"kind":"Name","value":"groupID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"sleep"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"until"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"waitForEvent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"waitResult"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"invokeFunction"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"correlationID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"invokeFunctionResult"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","alias":{"kind":"Name","value":"version"},"name":{"kind":"Name","value":"workflowVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunDetailsQuery, GetFunctionRunDetailsQueryVariables>;
export const GetFunctionRunTriggersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunTriggers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunTriggersQuery, GetFunctionRunTriggersQueryVariables>;
export const GetFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunTimeField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsQuery, GetFunctionRunsQueryVariables>;
export const GetFunctionEndedRunsCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionEndedRunsCount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"failedRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"FAILED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"STARTED_AT"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"canceledRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"CANCELLED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"STARTED_AT"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"succeededRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"COMPLETED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"STARTED_AT"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionEndedRunsCountQuery, GetFunctionEndedRunsCountQueryVariables>;
export const CreateFunctionReplayDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateFunctionReplay"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createFunctionReplay"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fromRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"toRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"statuses"},"value":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateFunctionReplayMutation, CreateFunctionReplayMutationVariables>;
export const GetFunctionRunsCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsCount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunTimeField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunStatuses"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeStart"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeRangeEnd"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsCountQuery, GetFunctionRunsCountQueryVariables>;
export const GetReplaysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplays"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"replays"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsScheduledCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplaysQuery, GetReplaysQueryVariables>;
export const GetRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RunsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetRunsQuery, GetRunsQueryVariables>;
export const NewIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"NewIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<NewIngestKeyMutation, NewIngestKeyMutationVariables>;
export const GetIngestKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeysQuery, GetIngestKeysQueryVariables>;
export const UpdateIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}}]}}]}}]} as unknown as DocumentNode<UpdateIngestKeyMutation, UpdateIngestKeyMutationVariables>;
export const DeleteEventKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteEventKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DeleteIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<DeleteEventKeyMutation, DeleteEventKeyMutationVariables>;
export const GetIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeyQuery, GetIngestKeyQueryVariables>;
export const CreateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<CreateSigningKeyMutation, CreateSigningKeyMutationVariables>;
export const DeleteSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<DeleteSigningKeyMutation, DeleteSigningKeyMutationVariables>;
export const RotateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RotateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rotateSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<RotateSigningKeyMutation, RotateSigningKeyMutationVariables>;
export const GetSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetSigningKeyQuery, GetSigningKeyQueryVariables>;
export const GetSigningKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSigningKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"signingKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"decryptedValue"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isActive"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSigningKeysQuery, GetSigningKeysQueryVariables>;
export const UnattachedSyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncQuery, UnattachedSyncQueryVariables>;
export const UnattachedSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"syncs"},"name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncsQuery, UnattachedSyncsQueryVariables>;
export const GetBillableStepsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableSteps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billableStepTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableStepsQuery, GetBillableStepsQueryVariables>;
export const UpdateAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAccount"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"account"},"name":{"kind":"Name","value":"updateAccount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountMutation, UpdateAccountMutationVariables>;
export const CreateStripeSubscriptionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripeSubscription"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StripeSubscriptionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createStripeSubscription"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clientSecret"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CreateStripeSubscriptionMutation, CreateStripeSubscriptionMutationVariables>;
export const UpdatePlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<UpdatePlanMutation, UpdatePlanMutationVariables>;
export const GetPaymentIntentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPaymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"paymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"amountLabel"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"invoiceURL"}}]}}]}}]}}]} as unknown as DocumentNode<GetPaymentIntentsQuery, GetPaymentIntentsQueryVariables>;
export const UpdatePaymentMethodDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePaymentMethod"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePaymentMethod"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]} as unknown as DocumentNode<UpdatePaymentMethodMutation, UpdatePaymentMethodMutationVariables>;
export const GetBillingInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nextInvoiceDate"}}]}},{"kind":"Field","name":{"kind":"Name","value":"paymentMethods"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]} as unknown as DocumentNode<GetBillingInfoQuery, GetBillingInfoQueryVariables>;
export const GetSavedVercelProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSavedVercelProjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"savedVercelProjects"},"name":{"kind":"Name","value":"vercelApps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"workspaceID"}}]}}]}}]}}]} as unknown as DocumentNode<GetSavedVercelProjectsQuery, GetSavedVercelProjectsQueryVariables>;
export const CreateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<CreateVercelAppMutation, CreateVercelAppMutationVariables>;
export const UpdateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<UpdateVercelAppMutation, UpdateVercelAppMutationVariables>;
export const RemoveVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RemoveVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<RemoveVercelAppMutation, RemoveVercelAppMutationVariables>;
export const CreateWebhookDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateWebhook"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]} as unknown as DocumentNode<CreateWebhookMutation, CreateWebhookMutationVariables>;
export const CompleteAwsMarketplaceSetupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CompleteAWSMarketplaceSetup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSMarketplaceSetupInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"completeAWSMarketplaceSetup"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CompleteAwsMarketplaceSetupMutation, CompleteAwsMarketplaceSetupMutationVariables>;
export const GetAccountSupportInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountSupportInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountSupportInfoQuery, GetAccountSupportInfoQueryVariables>;
export const GetArchivedAppBannerDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetArchivedAppBannerData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"isArchived"}}]}}]}}]}}]} as unknown as DocumentNode<GetArchivedAppBannerDataQuery, GetArchivedAppBannerDataQueryVariables>;
export const GetGlobalSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGlobalSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"opts"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"Variable","name":{"kind":"Name","value":"opts"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"env"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"functionID"},"name":{"kind":"Name","value":"workflowID"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetGlobalSearchQuery, GetGlobalSearchQueryVariables>;
export const GetFunctionSlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionSlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionSlugQuery, GetFunctionSlugQueryVariables>;
export const GetRunTraceTriggerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTraceTrigger"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runTrigger"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"IDs"}},{"kind":"Field","name":{"kind":"Name","value":"payloads"}},{"kind":"Field","name":{"kind":"Name","value":"timestamp"}},{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"isBatch"}},{"kind":"Field","name":{"kind":"Name","value":"batchID"}},{"kind":"Field","name":{"kind":"Name","value":"cron"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceTriggerQuery, GetRunTraceTriggerQueryVariables>;
export const GetRunTraceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTrace"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"trace"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}}]}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceQuery, GetRunTraceQueryVariables>;
export const GetDeployssDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDeployss"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"checksum"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetDeployssQuery, GetDeployssQueryVariables>;
export const GetEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentsQuery, GetEnvironmentsQueryVariables>;
export const GetEventTypesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"functions"},"name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyVolume"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypesQuery, GetEventTypesQueryVariables>;
export const GetEventTypeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventType"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeQuery, GetEventTypeQueryVariables>;
export const GetFunctionsUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionsUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsUsageQuery, GetFunctionsUsageQueryVariables>;
export const GetFunctionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsQuery, GetFunctionsQueryVariables>;
export const GetFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"failureHandler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"configuration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"retries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isDefault"}}]}},{"kind":"Field","name":{"kind":"Name","value":"priority"}},{"kind":"Field","name":{"kind":"Name","value":"eventsBatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"maxSize"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"limit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isPlanLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"rateLimit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"debounce"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"throttle"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"burst"}},{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionQuery, GetFunctionQueryVariables>;
export const GetFunctionUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionUsageQuery, GetFunctionUsageQueryVariables>;
export const GetAllEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAllEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}}]}}]}}]} as unknown as DocumentNode<GetAllEnvironmentsQuery, GetAllEnvironmentsQueryVariables>;
export const CancelRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CancelRunMutation, CancelRunMutationVariables>;