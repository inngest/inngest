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
  addons: Addons;
  appliedAddons: AppliedAddons;
  billingEmail: Scalars['String'];
  createdAt: Scalars['Time'];
  entitlementUsage: EntitlementUsage;
  entitlements: Entitlements;
  id: Scalars['ID'];
  marketplace: Maybe<Marketplace>;
  name: Maybe<Scalars['NullString']>;
  paymentIntents: Array<PaymentIntent>;
  paymentMethods: Maybe<Array<PaymentMethod>>;
  plan: Maybe<BillingPlan>;
  quickSearch: QuickSearchResults;
  search: SearchResults;
  status: Scalars['String'];
  subscription: Maybe<BillingSubscription>;
  updatedAt: Scalars['Time'];
  users: Array<User>;
  vercelIntegration: Maybe<VercelIntegration>;
};


export type AccountQuickSearchArgs = {
  envSlug: Scalars['String'];
  term: Scalars['String'];
};


export type AccountSearchArgs = {
  opts: SearchInput;
};

export type Addon = {
  __typename?: 'Addon';
  available: Scalars['Boolean'];
  baseValue: Maybe<Scalars['Int']>;
  maxValue: Scalars['Int'];
  name: Scalars['String'];
  price: Maybe<Scalars['Int']>;
  purchaseCount: Scalars['Int'];
  quantityPer: Scalars['Int'];
};

export type AddonMulti = {
  __typename?: 'AddonMulti';
  billingPeriod: Scalars['BillingPeriod'];
  id: Scalars['ID'];
  name: Scalars['String'];
  price: Price;
  quantityPer: Scalars['Int'];
};

export type Addons = {
  __typename?: 'Addons';
  accountID: Maybe<Scalars['UUID']>;
  concurrency: Addon;
  hipaa: Addon;
  planID: Maybe<Scalars['UUID']>;
  userCount: Addon;
};

export type App = {
  __typename?: 'App';
  archivedAt: Maybe<Scalars['Time']>;
  connectionType: AppConnectionType;
  createdAt: Scalars['Time'];
  externalID: Scalars['String'];
  functionCount: Scalars['Int'];
  functions: Array<Workflow>;
  id: Scalars['UUID'];
  /** @deprecated connectionType is deprecated. Use method instead. */
  isArchived: Scalars['Boolean'];
  isParentArchived: Scalars['Boolean'];
  latestSync: Maybe<Deploy>;
  method: AppMethod;
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

export type AppCheckFieldBoolean = {
  __typename?: 'AppCheckFieldBoolean';
  value: Maybe<Scalars['Boolean']>;
};

export type AppCheckFieldString = {
  __typename?: 'AppCheckFieldString';
  value: Maybe<Scalars['String']>;
};

export type AppCheckResult = {
  __typename?: 'AppCheckResult';
  apiOrigin: Maybe<AppCheckFieldString>;
  appID: Maybe<AppCheckFieldString>;
  authenticationSucceeded: Maybe<AppCheckFieldBoolean>;
  env: Maybe<AppCheckFieldString>;
  error: Maybe<Scalars['String']>;
  eventAPIOrigin: Maybe<AppCheckFieldString>;
  eventKeyStatus: SecretCheck;
  extra: Maybe<Scalars['Map']>;
  framework: Maybe<AppCheckFieldString>;
  isReachable: Scalars['Boolean'];
  isSDK: Scalars['Boolean'];
  mode: Maybe<SdkMode>;
  respHeaders: Maybe<Scalars['Map']>;
  respStatusCode: Maybe<Scalars['Int']>;
  sdkLanguage: Maybe<AppCheckFieldString>;
  sdkVersion: Maybe<AppCheckFieldString>;
  serveOrigin: Maybe<AppCheckFieldString>;
  servePath: Maybe<AppCheckFieldString>;
  signingKeyFallbackStatus: SecretCheck;
  signingKeyStatus: SecretCheck;
};

export enum AppConnectionType {
  Connect = 'CONNECT',
  Serverless = 'SERVERLESS'
}

export enum AppMethod {
  Connect = 'CONNECT',
  Serve = 'SERVE'
}

export type AppliedAddonMulti = {
  __typename?: 'AppliedAddonMulti';
  addon: AddonMulti;
  quantity: Scalars['Int'];
};

export type AppliedAddons = {
  __typename?: 'AppliedAddons';
  concurrency: Maybe<AppliedAddonMulti>;
  users: Maybe<AppliedAddonMulti>;
};

export type AppsFilter = {
  archived?: InputMaybe<Scalars['Boolean']>;
  connectionType?: InputMaybe<AppConnectionType>;
  method?: InputMaybe<AppMethod>;
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
  skippedFunctionRuns: Array<SkippedFunctionRun>;
  source: Maybe<IngestKey>;
  version: Scalars['String'];
};

export type AvailableAddons = {
  __typename?: 'AvailableAddons';
  concurrency: Maybe<AddonMulti>;
  users: Maybe<AddonMulti>;
};

export type BillingPlan = {
  __typename?: 'BillingPlan';
  addons: Addons;
  amount: Scalars['Int'];
  availableAddons: AvailableAddons;
  billingPeriod: Scalars['BillingPeriod'];
  entitlements: Entitlements;
  features: Scalars['Map'];
  id: Scalars['ID'];
  name: Scalars['String'];
};

export type BillingSubscription = {
  __typename?: 'BillingSubscription';
  nextInvoiceDate: Scalars['Time'];
};

export type CdcConnection = {
  __typename?: 'CDCConnection';
  Host: Scalars['String'];
  createdAt: Scalars['Time'];
  description: Maybe<Scalars['String']>;
  engine: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  status: CdcStatus;
  statusDetail: Maybe<Scalars['Map']>;
  updatedAt: Scalars['Time'];
  watermark: Maybe<Scalars['Map']>;
};

export type CdcConnectionInput = {
  adminConn: Scalars['String'];
  engine: Scalars['String'];
  name: Scalars['String'];
  replicaConn?: InputMaybe<Scalars['String']>;
};

export type CdcSetupResponse = {
  __typename?: 'CDCSetupResponse';
  error: Maybe<Scalars['String']>;
  steps: Maybe<Scalars['Map']>;
};

export enum CdcStatus {
  Error = 'ERROR',
  Running = 'RUNNING',
  SetupComplete = 'SETUP_COMPLETE',
  SetupIncomplete = 'SETUP_INCOMPLETE',
  Stopped = 'STOPPED'
}

export type Cancellation = {
  __typename?: 'Cancellation';
  createdAt: Scalars['Time'];
  environmentID: Scalars['UUID'];
  expression: Maybe<Scalars['String']>;
  functionID: Scalars['UUID'];
  id: Scalars['ULID'];
  name: Maybe<Scalars['String']>;
  queuedAtMax: Scalars['Time'];
  queuedAtMin: Maybe<Scalars['Time']>;
};

export type CancellationConfiguration = {
  __typename?: 'CancellationConfiguration';
  condition: Maybe<Scalars['String']>;
  event: Scalars['String'];
  timeout: Maybe<Scalars['String']>;
};

export type CancellationConnection = {
  __typename?: 'CancellationConnection';
  edges: Array<CancellationEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type CancellationEdge = {
  __typename?: 'CancellationEdge';
  cursor: Scalars['String'];
  node: Cancellation;
};

export type CancellationRunCountInput = {
  queuedAtMax: Scalars['Time'];
  queuedAtMin?: InputMaybe<Scalars['Time']>;
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

export enum ConnectV1ConnectionStatus {
  Connected = 'CONNECTED',
  Disconnected = 'DISCONNECTED',
  Disconnecting = 'DISCONNECTING',
  Draining = 'DRAINING',
  Ready = 'READY'
}

export type ConnectV1WorkerConnection = {
  __typename?: 'ConnectV1WorkerConnection';
  app: Maybe<App>;
  appID: Maybe<Scalars['UUID']>;
  appVersion: Maybe<Scalars['String']>;
  buildId: Maybe<Scalars['String']>;
  connectedAt: Scalars['Time'];
  cpuCores: Scalars['Int'];
  /** @deprecated buildId is deprecated. Use appVersion instead. */
  deploy: Maybe<Deploy>;
  disconnectReason: Maybe<Scalars['String']>;
  disconnectedAt: Maybe<Scalars['Time']>;
  functionCount: Scalars['Int'];
  gatewayId: Scalars['ULID'];
  id: Scalars['ULID'];
  instanceId: Scalars['String'];
  lastHeartbeatAt: Maybe<Scalars['Time']>;
  memBytes: Scalars['Int'];
  os: Scalars['String'];
  sdkLang: Scalars['String'];
  sdkPlatform: Scalars['String'];
  sdkVersion: Scalars['String'];
  status: ConnectV1ConnectionStatus;
  workerIp: Scalars['String'];
};

export type ConnectV1WorkerConnectionEdge = {
  __typename?: 'ConnectV1WorkerConnectionEdge';
  cursor: Scalars['String'];
  node: ConnectV1WorkerConnection;
};

export type ConnectV1WorkerConnectionsConnection = {
  __typename?: 'ConnectV1WorkerConnectionsConnection';
  edges: Array<ConnectV1WorkerConnectionEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type ConnectV1WorkerConnectionsFilter = {
  appIDs?: InputMaybe<Array<Scalars['UUID']>>;
  from: Scalars['Time'];
  status?: InputMaybe<Array<ConnectV1ConnectionStatus>>;
  timeField?: InputMaybe<ConnectV1WorkerConnectionsOrderByField>;
  until?: InputMaybe<Scalars['Time']>;
};

export type ConnectV1WorkerConnectionsOrderBy = {
  direction: ConnectV1WorkerConnectionsOrderByDirection;
  field: ConnectV1WorkerConnectionsOrderByField;
};

export enum ConnectV1WorkerConnectionsOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export enum ConnectV1WorkerConnectionsOrderByField {
  ConnectedAt = 'CONNECTED_AT',
  DisconnectedAt = 'DISCONNECTED_AT',
  LastHeartbeatAt = 'LAST_HEARTBEAT_AT'
}

export type CreateCancellationInput = {
  envID: Scalars['UUID'];
  functionSlug: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
  queuedAtMax: Scalars['Time'];
  queuedAtMin?: InputMaybe<Scalars['Time']>;
  testOnly?: InputMaybe<CreateCancellationInputTestOnly>;
};

export type CreateCancellationInputTestOnly = {
  maxStepCount?: InputMaybe<Scalars['Int']>;
  queryLimit?: InputMaybe<Scalars['Int']>;
};

export type CreateFunctionReplayInput = {
  fromRange: Scalars['ULID'];
  name: Scalars['String'];
  statuses?: InputMaybe<Array<FunctionRunStatus>>;
  statusesV2?: InputMaybe<Array<ReplayRunStatus>>;
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
  originOverride?: InputMaybe<Scalars['String']>;
  path?: InputMaybe<Scalars['String']>;
  projectID: Scalars['String'];
  protectionBypassSecret?: InputMaybe<Scalars['String']>;
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
  idempotencyKey: Maybe<Scalars['String']>;
  lastSyncedAt: Scalars['Time'];
  metadata: Scalars['Map'];
  platform: Maybe<Scalars['String']>;
  prevFunctionCount: Maybe<Scalars['Int']>;
  removedFunctions: Array<Workflow>;
  repoURL: Maybe<Scalars['String']>;
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  status: Scalars['String'];
  syncKind: Maybe<Scalars['String']>;
  trustProbeStatus: Maybe<Scalars['String']>;
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

export type EntitlementBool = {
  __typename?: 'EntitlementBool';
  enabled: Scalars['Boolean'];
};

export type EntitlementConcurrency = {
  __typename?: 'EntitlementConcurrency';
  limit: Scalars['Int'];
  usage: Scalars['Int'];
};

export type EntitlementInt = {
  __typename?: 'EntitlementInt';
  limit: Scalars['Int'];
};

export type EntitlementRunCount = {
  __typename?: 'EntitlementRunCount';
  limit: Maybe<Scalars['Int']>;
  overageAllowed: Scalars['Boolean'];
  usage: Scalars['Int'];
};

export type EntitlementStepCount = {
  __typename?: 'EntitlementStepCount';
  limit: Maybe<Scalars['Int']>;
  overageAllowed: Scalars['Boolean'];
  usage: Scalars['Int'];
};

export type EntitlementUsage = {
  __typename?: 'EntitlementUsage';
  accountConcurrencyLimitHits: Scalars['Int'];
  runCount: EntitlementUsageRunCount;
  stepCount: EntitlementUsageStepCount;
};

export type EntitlementUsageRunCount = {
  __typename?: 'EntitlementUsageRunCount';
  current: Scalars['Int'];
  limit: Maybe<Scalars['Int']>;
  overageAllowed: Scalars['Boolean'];
};

export type EntitlementUsageStepCount = {
  __typename?: 'EntitlementUsageStepCount';
  current: Scalars['Int'];
  limit: Maybe<Scalars['Int']>;
  overageAllowed: Scalars['Boolean'];
};

export type EntitlementUserCount = {
  __typename?: 'EntitlementUserCount';
  limit: Maybe<Scalars['Int']>;
  usage: Scalars['Int'];
};

export type Entitlements = {
  __typename?: 'Entitlements';
  accountID: Maybe<Scalars['UUID']>;
  concurrency: EntitlementConcurrency;
  eventBatchCount: EntitlementInt;
  eventSize: EntitlementInt;
  hipaa: EntitlementBool;
  history: EntitlementInt;
  metricsExport: EntitlementBool;
  metricsExportFreshness: EntitlementInt;
  metricsExportGranularity: EntitlementInt;
  planID: Maybe<Scalars['UUID']>;
  runCount: EntitlementRunCount;
  stepCount: EntitlementStepCount;
  userCount: EntitlementUserCount;
};

export type EnvEdge = {
  __typename?: 'EnvEdge';
  cursor: Scalars['String'];
  node: Workspace;
};

export enum EnvironmentType {
  BranchChild = 'BRANCH_CHILD',
  BranchParent = 'BRANCH_PARENT',
  Production = 'PRODUCTION',
  Test = 'TEST'
}

export type EnvsConnection = {
  __typename?: 'EnvsConnection';
  edges: Array<EnvEdge>;
  pageInfo: PageInfo;
};

export type EnvsFilter = {
  archived?: InputMaybe<Scalars['Boolean']>;
  envTypes?: InputMaybe<Array<EnvironmentType>>;
};

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
  key: Maybe<Scalars['String']>;
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
  /** The function is paused. */
  Paused = 'PAUSED',
  /** The function run has been scheduled. */
  Queued = 'QUEUED',
  /** The function run is currently running. */
  Running = 'RUNNING',
  /** The function run was skipped */
  Skipped = 'SKIPPED'
}

export enum FunctionRunTimeField {
  EndedAt = 'ENDED_AT',
  Mixed = 'MIXED',
  StartedAt = 'STARTED_AT'
}

export type FunctionRunV2 = {
  __typename?: 'FunctionRunV2';
  accountID: Scalars['UUID'];
  app: App;
  appID: Scalars['UUID'];
  batchCreatedAt: Maybe<Scalars['Time']>;
  cronSchedule: Maybe<Scalars['String']>;
  endedAt: Maybe<Scalars['Time']>;
  eventName: Maybe<Scalars['String']>;
  function: Workflow;
  functionID: Scalars['UUID'];
  hasAI: Scalars['Boolean'];
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

export enum Marketplace {
  Vercel = 'VERCEL'
}

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

export enum MetricsScope {
  App = 'APP',
  Env = 'ENV',
  Fn = 'FN'
}

export type Mutation = {
  __typename?: 'Mutation';
  archiveApp: App;
  archiveEnvironment: Workspace;
  archiveEvent: Maybe<Event>;
  archiveWorkflow: Maybe<WorkflowResponse>;
  cancelRun: FunctionRun;
  cdcAutoSetup: CdcSetupResponse;
  cdcDelete: DeleteResponse;
  cdcManualSetup: CdcSetupResponse;
  cdcTestCredentials: CdcSetupResponse;
  cdcTestLogicalReplication: CdcSetupResponse;
  cdcTestSetup: CdcSetupResponse;
  completeAWSMarketplaceSetup: Maybe<AwsMarketplaceSetupResponse>;
  createCancellation: Cancellation;
  createFunctionReplay: Replay;
  createIngestKey: IngestKey;
  createSigningKey: SigningKey;
  createStripeSubscription: CreateStripeSubscriptionResponse;
  createUser: Maybe<CreateUserPayload>;
  createVercelApp: Maybe<CreateVercelAppResponse>;
  createWorkspace: Array<Maybe<Workspace>>;
  deleteCancellation: Scalars['ULID'];
  deleteIngestKey: Maybe<DeleteResponse>;
  deleteSigningKey: SigningKey;
  disableEnvironmentAutoArchive: Workspace;
  editWorkflow: Maybe<WorkflowVersionResponse>;
  enableEnvironmentAutoArchive: Workspace;
  invokeFunction: Maybe<Scalars['Boolean']>;
  pauseFunction: Workflow;
  removeVercelApp: Maybe<RemoveVercelAppResponse>;
  rerun: Scalars['ULID'];
  resyncApp: SyncResponse;
  retryWorkflowRun: Maybe<StartWorkflowResponse>;
  rotateSigningKey: SigningKey;
  setAccountEntitlement: Scalars['UUID'];
  setUpAccount: Maybe<SetUpAccountPayload>;
  syncNewApp: SyncResponse;
  unarchiveApp: App;
  unarchiveEnvironment: Workspace;
  unpauseFunction: Workflow;
  updateAccount: Account;
  updateAccountAddonQuantity: Addon;
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


export type MutationCdcAutoSetupArgs = {
  envID: Scalars['UUID'];
  input: CdcConnectionInput;
};


export type MutationCdcDeleteArgs = {
  envID: Scalars['UUID'];
  id: Scalars['UUID'];
};


export type MutationCdcManualSetupArgs = {
  envID: Scalars['UUID'];
  input: CdcConnectionInput;
};


export type MutationCdcTestCredentialsArgs = {
  envID: Scalars['UUID'];
  input: CdcConnectionInput;
};


export type MutationCdcTestLogicalReplicationArgs = {
  envID: Scalars['UUID'];
  input: CdcConnectionInput;
};


export type MutationCdcTestSetupArgs = {
  envID: Scalars['UUID'];
  input: CdcConnectionInput;
};


export type MutationCompleteAwsMarketplaceSetupArgs = {
  input: AwsMarketplaceSetupInput;
};


export type MutationCreateCancellationArgs = {
  input: CreateCancellationInput;
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


export type MutationDeleteCancellationArgs = {
  cancellationID: Scalars['ULID'];
  envID: Scalars['UUID'];
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
  user: InputMaybe<Scalars['Map']>;
};


export type MutationPauseFunctionArgs = {
  cancelRunning: InputMaybe<Scalars['Boolean']>;
  fnID: Scalars['ID'];
};


export type MutationRemoveVercelAppArgs = {
  input: RemoveVercelAppInput;
};


export type MutationRerunArgs = {
  fromStep: InputMaybe<RerunFromStepInput>;
  runID: Scalars['ULID'];
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


export type MutationSetAccountEntitlementArgs = {
  entitlementName: Scalars['String'];
  overrideStrategy: Scalars['String'];
  value: Scalars['Int'];
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


export type MutationUpdateAccountAddonQuantityArgs = {
  addonName: Scalars['String'];
  quantity: Scalars['Int'];
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

export type Price = {
  __typename?: 'Price';
  usCents: Scalars['Int'];
};

export type Query = {
  __typename?: 'Query';
  account: Account;
  billableStepTimeSeries: Array<TimeSeries>;
  defaultEnv: Workspace;
  deploy: Deploy;
  deploys: Maybe<Array<Deploy>>;
  envBySlug: Maybe<Workspace>;
  envs: EnvsConnection;
  events: Maybe<PaginatedEvents>;
  plans: Array<Maybe<BillingPlan>>;
  runCountTimeSeries: Array<TimeSeries>;
  session: Maybe<Session>;
  workspace: Workspace;
  workspaces: Maybe<Array<Workspace>>;
};


export type QueryBillableStepTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryDeployArgs = {
  id: Scalars['ID'];
};


export type QueryDeploysArgs = {
  workspaceID: InputMaybe<Scalars['ID']>;
};


export type QueryEnvBySlugArgs = {
  slug: Scalars['String'];
};


export type QueryEnvsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: InputMaybe<EnvsFilter>;
  first?: Scalars['Int'];
};


export type QueryEventsArgs = {
  query: InputMaybe<EventQuery>;
};


export type QueryRunCountTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryWorkspaceArgs = {
  id: Scalars['ID'];
};

export type QuickSearchApp = {
  __typename?: 'QuickSearchApp';
  name: Scalars['String'];
};

export type QuickSearchEnv = {
  __typename?: 'QuickSearchEnv';
  name: Scalars['String'];
  slug: Scalars['String'];
};

export type QuickSearchEvent = {
  __typename?: 'QuickSearchEvent';
  id: Scalars['ULID'];
  name: Scalars['String'];
};

export type QuickSearchEventType = {
  __typename?: 'QuickSearchEventType';
  name: Scalars['String'];
};

export type QuickSearchFunction = {
  __typename?: 'QuickSearchFunction';
  name: Scalars['String'];
  slug: Scalars['String'];
};

export type QuickSearchResults = {
  __typename?: 'QuickSearchResults';
  apps: Array<QuickSearchApp>;
  event: Maybe<QuickSearchEvent>;
  eventTypes: Array<QuickSearchEventType>;
  functions: Array<QuickSearchFunction>;
  run: Maybe<QuickSearchRun>;
};

export type QuickSearchRun = {
  __typename?: 'QuickSearchRun';
  id: Scalars['ULID'];
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

export type ReplayRunCounts = {
  __typename?: 'ReplayRunCounts';
  cancelledCount: Scalars['Int'];
  completedCount: Scalars['Int'];
  failedCount: Scalars['Int'];
  skippedPausedCount: Scalars['Int'];
};

export enum ReplayRunStatus {
  All = 'ALL',
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  SkippedPaused = 'SKIPPED_PAUSED'
}

export enum ReplayType {
  Event = 'EVENT',
  Function = 'FUNCTION'
}

export type RerunFromStepInput = {
  input?: InputMaybe<Scalars['Bytes']>;
  stepID: Scalars['String'];
};

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
  stepID: Maybe<Scalars['String']>;
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
  input: Maybe<Scalars['Bytes']>;
};

export enum RunTraceSpanStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Paused = 'PAUSED',
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
  totalCount: Scalars['Int'];
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

export enum SdkMode {
  Cloud = 'CLOUD',
  Dev = 'DEV'
}

export type ScopedFunctionStatusResponse = {
  __typename?: 'ScopedFunctionStatusResponse';
  cancelled: Scalars['Int'];
  completed: Scalars['Int'];
  failed: Scalars['Int'];
  from: Scalars['Time'];
  queued: Scalars['Int'];
  running: Scalars['Int'];
  skipped: Scalars['Int'];
  to: Scalars['Time'];
};

export type ScopedMetric = {
  __typename?: 'ScopedMetric';
  data: Array<MetricsData>;
  id: Scalars['UUID'];
  tagName: Maybe<Scalars['String']>;
  tagValue: Maybe<Scalars['String']>;
};

export type ScopedMetricsFilter = {
  appIDs?: InputMaybe<Array<Scalars['UUID']>>;
  from: Scalars['Time'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']>>;
  groupBy?: InputMaybe<Scalars['String']>;
  name: Scalars['String'];
  scope: MetricsScope;
  until?: InputMaybe<Scalars['Time']>;
};

export type ScopedMetricsResponse = {
  __typename?: 'ScopedMetricsResponse';
  from: Scalars['Time'];
  granularity: Scalars['String'];
  metrics: Array<ScopedMetric>;
  scope: MetricsScope;
  to: Scalars['Time'];
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

export enum SecretCheck {
  Correct = 'CORRECT',
  Incorrect = 'INCORRECT',
  Missing = 'MISSING',
  Unknown = 'UNKNOWN'
}

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
  signingKeyFallbackState: SecretCheck;
  signingKeyState: SecretCheck;
};

export enum SkipReason {
  FunctionPaused = 'FUNCTION_PAUSED',
  None = 'NONE'
}

export type SkippedFunctionRun = {
  __typename?: 'SkippedFunctionRun';
  accountID: Scalars['UUID'];
  batchID: Maybe<Scalars['ULID']>;
  eventID: Maybe<Scalars['ULID']>;
  id: Scalars['ULID'];
  skipReason: SkipReason;
  skippedAt: Scalars['Time'];
  workflowID: Scalars['UUID'];
  workspaceID: Scalars['UUID'];
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
  AiGateway = 'AI_GATEWAY',
  Invoke = 'INVOKE',
  Run = 'RUN',
  Sleep = 'SLEEP',
  WaitForEvent = 'WAIT_FOR_EVENT'
}

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

export type TimeSeriesOptions = {
  interval?: InputMaybe<Scalars['String']>;
  month: Scalars['Int'];
  year: Scalars['Int'];
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
  originOverride?: InputMaybe<Scalars['String']>;
  path: Scalars['String'];
  projectID: Scalars['String'];
  protectionBypassSecret?: InputMaybe<Scalars['String']>;
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
  originOverride: Maybe<Scalars['String']>;
  path: Maybe<Scalars['String']>;
  projectID: Scalars['String'];
  protectionBypassSecret: Maybe<Scalars['String']>;
  workspaceID: Scalars['UUID'];
};

export enum VercelDeploymentProtection {
  All = 'ALL',
  Disabled = 'DISABLED',
  Preview = 'PREVIEW',
  ProdDeploymentUrlsAndAllPreviews = 'PROD_DEPLOYMENT_URLS_AND_ALL_PREVIEWS',
  Unknown = 'UNKNOWN'
}

export type VercelIntegration = {
  __typename?: 'VercelIntegration';
  isMarketplace: Scalars['Boolean'];
  projects: Array<VercelProject>;
};

export type VercelProject = {
  __typename?: 'VercelProject';
  canChangeEnabled: Scalars['Boolean'];
  deploymentProtection: VercelDeploymentProtection;
  isEnabled: Scalars['Boolean'];
  name: Scalars['String'];
  originOverride: Maybe<Scalars['String']>;
  projectID: Scalars['String'];
  protectionBypassSecret: Maybe<Scalars['String']>;
  servePath: Scalars['String'];
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
  cancellationRunCount: Scalars['Int'];
  cancellations: CancellationConnection;
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
  replayCounts: ReplayRunCounts;
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
  triggers: Array<WorkflowTrigger>;
  url: Scalars['String'];
  usage: Usage;
};


export type WorkflowCancellationRunCountArgs = {
  input: CancellationRunCountInput;
};


export type WorkflowCancellationsArgs = {
  after: InputMaybe<Scalars['String']>;
  first?: Scalars['Int'];
};


export type WorkflowMetricsArgs = {
  opts: MetricsRequest;
};


export type WorkflowReplayCountsArgs = {
  from: Scalars['Time'];
  to: Scalars['Time'];
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
  appCheck: AppCheckResult;
  apps: Array<App>;
  archivedEvent: Maybe<ArchivedEvent>;
  cdcConnections: Array<CdcConnection>;
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
  scopedFunctionStatus: ScopedFunctionStatusResponse;
  scopedMetrics: ScopedMetricsResponse;
  signingKeys: Array<SigningKey>;
  slug: Scalars['String'];
  test: Scalars['Boolean'];
  traceOutput: Scalars['Bytes'];
  type: EnvironmentType;
  unattachedSyncs: Array<Deploy>;
  vercelApps: Array<VercelApp>;
  webhookSigningKey: Scalars['String'];
  workerConnection: Maybe<ConnectV1WorkerConnection>;
  workerConnections: ConnectV1WorkerConnectionsConnection;
  workflow: Maybe<Workflow>;
  workflowBySlug: Maybe<Workflow>;
  workflows: PaginatedWorkflows;
};


export type WorkspaceAppByExternalIdArgs = {
  externalID: Scalars['String'];
};


export type WorkspaceAppCheckArgs = {
  url: Scalars['String'];
};


export type WorkspaceAppsArgs = {
  filter: InputMaybe<AppsFilter>;
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


export type WorkspaceScopedFunctionStatusArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceScopedMetricsArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceTraceOutputArgs = {
  outputID: Scalars['String'];
};


export type WorkspaceUnattachedSyncsArgs = {
  after: InputMaybe<Scalars['Time']>;
  first?: Scalars['Int'];
};


export type WorkspaceWorkerConnectionArgs = {
  connectionId: Scalars['ULID'];
};


export type WorkspaceWorkerConnectionsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: ConnectV1WorkerConnectionsFilter;
  first?: Scalars['Int'];
  orderBy: Array<ConnectV1WorkerConnectionsOrderBy>;
};


export type WorkspaceWorkflowArgs = {
  id: Scalars['ID'];
};


export type WorkspaceWorkflowBySlugArgs = {
  slug: Scalars['String'];
};


export type WorkspaceWorkflowsArgs = {
  archived?: InputMaybe<Scalars['Boolean']>;
  search: InputMaybe<Scalars['String']>;
};

export type SetUpAccountMutationVariables = Exact<{ [key: string]: never; }>;


export type SetUpAccountMutation = { __typename?: 'Mutation', setUpAccount: { __typename?: 'SetUpAccountPayload', account: { __typename?: 'Account', id: string } | null } | null };

export type CreateUserMutationVariables = Exact<{ [key: string]: never; }>;


export type CreateUserMutation = { __typename?: 'Mutation', createUser: { __typename?: 'CreateUserPayload', user: { __typename?: 'User', id: string } | null } | null };

export type GetBillingInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', entitlements: { __typename?: 'Entitlements', stepCount: { __typename?: 'EntitlementStepCount', usage: number, limit: number | null }, runCount: { __typename?: 'EntitlementRunCount', usage: number, limit: number | null } } } };

export type CreateEnvironmentMutationVariables = Exact<{
  name: Scalars['String'];
}>;


export type CreateEnvironmentMutation = { __typename?: 'Mutation', createWorkspace: Array<{ __typename?: 'Workspace', id: string } | null> };

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

export type CheckAppQueryVariables = Exact<{
  envID: Scalars['ID'];
  url: Scalars['String'];
}>;


export type CheckAppQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', appCheck: { __typename?: 'AppCheckResult', error: string | null, eventKeyStatus: SecretCheck, extra: Record<string, unknown> | null, isReachable: boolean, isSDK: boolean, mode: SdkMode | null, respHeaders: Record<string, unknown> | null, respStatusCode: number | null, signingKeyStatus: SecretCheck, signingKeyFallbackStatus: SecretCheck, apiOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, appID: { __typename?: 'AppCheckFieldString', value: string | null } | null, authenticationSucceeded: { __typename?: 'AppCheckFieldBoolean', value: boolean | null } | null, env: { __typename?: 'AppCheckFieldString', value: string | null } | null, eventAPIOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, framework: { __typename?: 'AppCheckFieldString', value: string | null } | null, sdkLanguage: { __typename?: 'AppCheckFieldString', value: string | null } | null, sdkVersion: { __typename?: 'AppCheckFieldString', value: string | null } | null, serveOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, servePath: { __typename?: 'AppCheckFieldString', value: string | null } | null } } };

export type SyncQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
  syncID: Scalars['ID'];
}>;


export type SyncQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, name: string, method: AppMethod } }, sync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> } };

export type AppSyncsQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type AppSyncsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, syncs: Array<{ __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> }> } } };

export type AppQueryVariables = Exact<{
  envID: Scalars['ID'];
  externalAppID: Scalars['String'];
}>;


export type AppQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, name: string, method: AppMethod, functions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string, latestVersion: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } }>, latestSync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null } | null } } };

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


export type AppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', id: string, externalID: string, functionCount: number, isArchived: boolean, name: string, method: AppMethod, isParentArchived: boolean, latestSync: { __typename?: 'Deploy', error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null } | null, functions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string, triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> }> }> } };

export type LatestUnattachedSyncQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type LatestUnattachedSyncQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string }> } };

export type GetHistoryItemOutputQueryVariables = Exact<{
  envID: Scalars['ID'];
  functionID: Scalars['ID'];
  historyItemID: Scalars['ULID'];
  runID: Scalars['ULID'];
}>;


export type GetHistoryItemOutputQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', run: { __typename?: 'FunctionRun', historyItemOutput: string | null } } | null } };

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

export type GetEventLogQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  eventName: Scalars['String'];
  cursor: InputMaybe<Scalars['String']>;
  perPage: Scalars['Int'];
}>;


export type GetEventLogQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventType: { __typename?: 'Event', events: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string }> } | null } };

export type EventPayloadFragment = { __typename?: 'ArchivedEvent', payload: string } & { ' $fragmentName'?: 'EventPayloadFragment' };

export type GetFunctionNameSlugQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
}>;


export type GetFunctionNameSlugQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', name: string, slug: string } | null } };

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
      { __typename?: 'ArchivedEvent', receivedAt: string, functionRuns: Array<{ __typename?: 'FunctionRun', id: string, function: { __typename?: 'Workflow', id: string } }>, skippedFunctionRuns: Array<{ __typename?: 'SkippedFunctionRun', id: string, skipReason: SkipReason, workflowID: string, skippedAt: string }> }
      & { ' $fragmentRefs'?: { 'EventPayloadFragment': EventPayloadFragment } }
    ) | null } };

export type GetAccountEntitlementsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAccountEntitlementsQuery = { __typename?: 'Query', account: { __typename?: 'Account', entitlements: { __typename?: 'Entitlements', history: { __typename?: 'EntitlementInt', limit: number } } } };

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

export type DeleteCancellationMutationVariables = Exact<{
  envID: Scalars['UUID'];
  cancellationID: Scalars['ULID'];
}>;


export type DeleteCancellationMutation = { __typename?: 'Mutation', deleteCancellation: string };

export type GetFnCancellationsQueryVariables = Exact<{
  after: InputMaybe<Scalars['String']>;
  envSlug: Scalars['String'];
  fnSlug: Scalars['String'];
}>;


export type GetFnCancellationsQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', fn: { __typename?: 'Workflow', cancellations: { __typename?: 'CancellationConnection', edges: Array<{ __typename?: 'CancellationEdge', cursor: string, node: { __typename?: 'Cancellation', createdAt: string, id: string, name: string | null, queuedAtMax: string, queuedAtMin: string | null, envID: string } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean } } } | null } | null };

export type InvokeFunctionMutationVariables = Exact<{
  envID: Scalars['UUID'];
  data: InputMaybe<Scalars['Map']>;
  functionSlug: Scalars['String'];
  user: InputMaybe<Scalars['Map']>;
}>;


export type InvokeFunctionMutation = { __typename?: 'Mutation', invokeFunction: boolean | null };

export type GetReplaysQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
}>;


export type GetReplaysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', id: string, function: { __typename?: 'Workflow', id: string, replays: Array<{ __typename?: 'Replay', id: string, name: string, createdAt: string, endedAt: string | null, functionRunsScheduledCount: number }> } | null } };

export type GetFunctionPauseStateQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
}>;


export type GetFunctionPauseStateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, isPaused: boolean } | null } };

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

export type VercelIntegrationQueryVariables = Exact<{ [key: string]: never; }>;


export type VercelIntegrationQuery = { __typename?: 'Query', account: { __typename?: 'Account', vercelIntegration: { __typename?: 'VercelIntegration', isMarketplace: boolean, projects: Array<{ __typename?: 'VercelProject', canChangeEnabled: boolean, deploymentProtection: VercelDeploymentProtection, isEnabled: boolean, name: string, originOverride: string | null, projectID: string, protectionBypassSecret: string | null, servePath: string }> } | null } };

export type GetSavedVercelProjectsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetSavedVercelProjectsQuery = { __typename?: 'Query', account: { __typename?: 'Account', marketplace: Marketplace | null }, environment: { __typename?: 'Workspace', savedVercelProjects: Array<{ __typename?: 'VercelApp', id: string, originOverride: string | null, projectID: string, protectionBypassSecret: string | null, path: string | null, workspaceID: string }> } };

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

export type GetArchivedFuncBannerDataQueryVariables = Exact<{
  envID: Scalars['ID'];
  funcID: Scalars['ID'];
}>;


export type GetArchivedFuncBannerDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, archivedAt: string | null } | null } };

export type UpdateAccountAddonQuantityMutationVariables = Exact<{
  addonName: Scalars['String'];
  quantity: Scalars['Int'];
}>;


export type UpdateAccountAddonQuantityMutation = { __typename?: 'Mutation', updateAccountAddonQuantity: { __typename?: 'Addon', purchaseCount: number } };

export type UpdateAccountMutationVariables = Exact<{
  input: UpdateAccount;
}>;


export type UpdateAccountMutation = { __typename?: 'Mutation', account: { __typename?: 'Account', billingEmail: string, name: null | string | null } };

export type UpdatePaymentMethodMutationVariables = Exact<{
  token: Scalars['String'];
}>;


export type UpdatePaymentMethodMutation = { __typename?: 'Mutation', updatePaymentMethod: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null };

export type GetPaymentIntentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPaymentIntentsQuery = { __typename?: 'Query', account: { __typename?: 'Account', paymentIntents: Array<{ __typename?: 'PaymentIntent', status: string, createdAt: string, amountLabel: string, description: string, invoiceURL: string | null }> } };

export type CreateStripeSubscriptionMutationVariables = Exact<{
  input: StripeSubscriptionInput;
}>;


export type CreateStripeSubscriptionMutation = { __typename?: 'Mutation', createStripeSubscription: { __typename?: 'CreateStripeSubscriptionResponse', clientSecret: string, message: string } };

export type UpdatePlanMutationVariables = Exact<{
  planID: Scalars['ID'];
}>;


export type UpdatePlanMutation = { __typename?: 'Mutation', updatePlan: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, name: string } | null } };

export type GetBillableStepsQueryVariables = Exact<{
  month: Scalars['Int'];
  year: Scalars['Int'];
}>;


export type GetBillableStepsQuery = { __typename?: 'Query', usage: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type GetBillableRunsQueryVariables = Exact<{
  month: Scalars['Int'];
  year: Scalars['Int'];
}>;


export type GetBillableRunsQuery = { __typename?: 'Query', usage: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type EntitlementUsageQueryVariables = Exact<{ [key: string]: never; }>;


export type EntitlementUsageQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, addons: { __typename?: 'Addons', concurrency: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, userCount: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number } }, entitlements: { __typename?: 'Entitlements', runCount: { __typename?: 'EntitlementRunCount', usage: number, limit: number | null, overageAllowed: boolean }, stepCount: { __typename?: 'EntitlementStepCount', usage: number, limit: number | null, overageAllowed: boolean }, concurrency: { __typename?: 'EntitlementConcurrency', usage: number, limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, userCount: { __typename?: 'EntitlementUserCount', usage: number, limit: number | null }, hipaa: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExport: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number } }, plan: { __typename?: 'BillingPlan', name: string } | null } };

export type GetCurrentPlanQueryVariables = Exact<{ [key: string]: never; }>;


export type GetCurrentPlanQuery = { __typename?: 'Query', account: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, runCount: { __typename?: 'EntitlementRunCount', limit: number | null }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null }, userCount: { __typename?: 'EntitlementUserCount', limit: number | null } }, addons: { __typename?: 'Addons', concurrency: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number }, userCount: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number } } } | null, subscription: { __typename?: 'BillingSubscription', nextInvoiceDate: string } | null } };

export type GetBillingDetailsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingDetailsQuery = { __typename?: 'Query', account: { __typename?: 'Account', billingEmail: string, name: null | string | null, paymentMethods: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null } };

export type GetPlansQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPlansQuery = { __typename?: 'Query', plans: Array<{ __typename?: 'BillingPlan', id: string, name: string, amount: number, billingPeriod: unknown, entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, runCount: { __typename?: 'EntitlementRunCount', limit: number | null }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null } } } | null> };

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

export type CreateCancellationMutationVariables = Exact<{
  input: CreateCancellationInput;
}>;


export type CreateCancellationMutation = { __typename?: 'Mutation', createCancellation: { __typename?: 'Cancellation', id: string } };

export type GetCancellationRunCountQueryVariables = Exact<{
  envID: Scalars['ID'];
  functionSlug: Scalars['String'];
  queuedAtMin: InputMaybe<Scalars['Time']>;
  queuedAtMax: Scalars['Time'];
}>;


export type GetCancellationRunCountQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', cancellationRunCount: number } | null } };

export type PauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID'];
  cancelRunning: InputMaybe<Scalars['Boolean']>;
}>;


export type PauseFunctionMutation = { __typename?: 'Mutation', pauseFunction: { __typename?: 'Workflow', id: string } };

export type UnpauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID'];
}>;


export type UnpauseFunctionMutation = { __typename?: 'Mutation', unpauseFunction: { __typename?: 'Workflow', id: string } };

export type MetricsLookupsQueryVariables = Exact<{
  envSlug: Scalars['String'];
  page: InputMaybe<Scalars['Int']>;
  pageSize: InputMaybe<Scalars['Int']>;
}>;


export type MetricsLookupsQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', externalID: string, id: string, name: string, isArchived: boolean }>, workflows: { __typename?: 'PaginatedWorkflows', data: Array<{ __typename?: 'Workflow', name: string, id: string, slug: string }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null, perPage: number } } } | null };

export type AccountConcurrencyLookupQueryVariables = Exact<{ [key: string]: never; }>;


export type AccountConcurrencyLookupQuery = { __typename?: 'Query', account: { __typename?: 'Account', entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number } } } };

export type FunctionStatusMetricsQueryVariables = Exact<{
  workspaceId: Scalars['ID'];
  from: Scalars['Time'];
  functionIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  until: InputMaybe<Scalars['Time']>;
  scope: MetricsScope;
}>;


export type FunctionStatusMetricsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', scheduled: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, started: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, completed: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, completedByFunction: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, totals: { __typename?: 'ScopedFunctionStatusResponse', queued: number, running: number, completed: number, failed: number, cancelled: number, skipped: number } } };

export type VolumeMetricsQueryVariables = Exact<{
  workspaceId: Scalars['ID'];
  from: Scalars['Time'];
  functionIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  until: InputMaybe<Scalars['Time']>;
  scope: MetricsScope;
}>;


export type VolumeMetricsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runsThroughput: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputEnded: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputStarted: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputScheduled: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, stepThroughput: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, backlog: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, stepRunning: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, concurrency: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> } } };

export type QuickSearchQueryVariables = Exact<{
  term: Scalars['String'];
  envSlug: Scalars['String'];
}>;


export type QuickSearchQuery = { __typename?: 'Query', account: { __typename?: 'Account', quickSearch: { __typename?: 'QuickSearchResults', apps: Array<{ __typename?: 'QuickSearchApp', name: string }>, event: { __typename?: 'QuickSearchEvent', id: string, name: string } | null, eventTypes: Array<{ __typename?: 'QuickSearchEventType', name: string }>, functions: Array<{ __typename?: 'QuickSearchFunction', name: string, slug: string }>, run: { __typename?: 'QuickSearchRun', id: string } | null } } };

export type GetGlobalSearchQueryVariables = Exact<{
  opts: SearchInput;
}>;


export type GetGlobalSearchQuery = { __typename?: 'Query', account: { __typename?: 'Account', search: { __typename?: 'SearchResults', results: Array<{ __typename?: 'SearchResult', kind: SearchResultType, env: { __typename?: 'Workspace', name: string, id: string, type: EnvironmentType, slug: string }, value: { __typename?: 'ArchivedEvent', id: string, name: string } | { __typename?: 'FunctionRun', id: string, startedAt: string, functionID: string } } | null> } } };

export type GetFunctionSlugQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
}>;


export type GetFunctionSlugQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', slug: string, name: string } | null } };

export type SyncOnboardingAppMutationVariables = Exact<{
  appURL: Scalars['String'];
  envID: Scalars['UUID'];
}>;


export type SyncOnboardingAppMutation = { __typename?: 'Mutation', syncNewApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', externalID: string, id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type InvokeFunctionOnboardingMutationVariables = Exact<{
  envID: Scalars['UUID'];
  data: InputMaybe<Scalars['Map']>;
  functionSlug: Scalars['String'];
  user: InputMaybe<Scalars['Map']>;
}>;


export type InvokeFunctionOnboardingMutation = { __typename?: 'Mutation', invokeFunction: boolean | null };

export type InvokeFunctionLookupQueryVariables = Exact<{
  envSlug: Scalars['String'];
  page: InputMaybe<Scalars['Int']>;
  pageSize: InputMaybe<Scalars['Int']>;
}>;


export type InvokeFunctionLookupQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', data: Array<{ __typename?: 'Workflow', name: string, id: string, slug: string, current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null }> } | null }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null, perPage: number } } } | null };

export type GetVercelAppsQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type GetVercelAppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string, error: string | null, url: string | null, vercelDeploymentURL: string | null }>, apps: Array<{ __typename?: 'App', id: string, name: string, externalID: string, isArchived: boolean, latestSync: { __typename?: 'Deploy', error: string | null, id: string, platform: string | null, vercelDeploymentID: string | null, vercelProjectID: string | null, status: string } | null }> } };

export type ProductionAppsQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type ProductionAppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', id: string }>, unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string }> } };

export type GetPostgresIntegrationsQueryVariables = Exact<{
  envID: Scalars['ID'];
}>;


export type GetPostgresIntegrationsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', cdcConnections: Array<{ __typename?: 'CDCConnection', id: string, name: string, status: CdcStatus, statusDetail: Record<string, unknown> | null, description: string | null }> } };

export type TestCredentialsMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID'];
}>;


export type TestCredentialsMutation = { __typename?: 'Mutation', cdcTestCredentials: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type TestReplicationMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID'];
}>;


export type TestReplicationMutation = { __typename?: 'Mutation', cdcTestLogicalReplication: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type TestAutoSetupMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID'];
}>;


export type TestAutoSetupMutation = { __typename?: 'Mutation', cdcAutoSetup: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type CdcDeleteMutationVariables = Exact<{
  envID: Scalars['UUID'];
  id: Scalars['UUID'];
}>;


export type CdcDeleteMutation = { __typename?: 'Mutation', cdcDelete: { __typename?: 'DeleteResponse', ids: Array<string> } };

export type MetricsEntitlementsQueryVariables = Exact<{ [key: string]: never; }>;


export type MetricsEntitlementsQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, entitlements: { __typename?: 'Entitlements', metricsExport: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number } } } };

export type GetReplayRunCountsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  functionSlug: Scalars['String'];
  from: Scalars['Time'];
  to: Scalars['Time'];
}>;


export type GetReplayRunCountsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, replayCounts: { __typename?: 'ReplayRunCounts', completedCount: number, failedCount: number, cancelledCount: number, skippedPausedCount: number } } | null } };

export type CreateFunctionReplayMutationVariables = Exact<{
  environmentID: Scalars['UUID'];
  functionID: Scalars['UUID'];
  name: Scalars['String'];
  fromRange: Scalars['ULID'];
  toRange: Scalars['ULID'];
  statuses: InputMaybe<Array<ReplayRunStatus> | ReplayRunStatus>;
}>;


export type CreateFunctionReplayMutation = { __typename?: 'Mutation', createFunctionReplay: { __typename?: 'Replay', id: string } };

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, outputID: string | null, stepID: string | null, spanID: string, stepOp: StepOp | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null } | { __typename: 'SleepStepInfo', sleepUntil: string } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null } | null } & { ' $fragmentName'?: 'TraceDetailsFragment' };

export type GetRunTraceQueryVariables = Exact<{
  envID: Scalars['ID'];
  runID: Scalars['String'];
}>;


export type GetRunTraceQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', run: { __typename?: 'FunctionRunV2', hasAI: boolean, function: { __typename?: 'Workflow', id: string, name: string, slug: string, app: { __typename?: 'App', name: string, externalID: string } }, trace: (
        { __typename?: 'RunTraceSpan', childrenSpans: Array<(
          { __typename?: 'RunTraceSpan', childrenSpans: Array<(
            { __typename?: 'RunTraceSpan' }
            & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
          )> }
          & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
        )> }
        & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
      ) | null } | null } };

export type TraceResultQueryVariables = Exact<{
  envID: Scalars['ID'];
  traceID: Scalars['String'];
}>;


export type TraceResultQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runTraceSpanOutputByID: { __typename?: 'RunTraceSpanOutput', data: string | null, input: string | null, error: { __typename?: 'StepError', message: string, name: string | null, stack: string | null } | null } } };

export type GetRunTraceTriggerQueryVariables = Exact<{
  envID: Scalars['ID'];
  runID: Scalars['String'];
}>;


export type GetRunTraceTriggerQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runTrigger: { __typename?: 'RunTraceTrigger', IDs: Array<string>, payloads: Array<string>, timestamp: string, eventName: string | null, isBatch: boolean, batchID: string | null, cron: string | null } } };

export type GetRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  endTime: InputMaybe<Scalars['Time']>;
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsOrderByField;
  functionSlug: InputMaybe<Scalars['String']>;
  functionRunCursor?: InputMaybe<Scalars['String']>;
  celQuery?: InputMaybe<Scalars['String']>;
}>;


export type GetRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', runs: { __typename?: 'RunsConnection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', cronSchedule: string | null, eventName: string | null, id: string, isBatch: boolean, queuedAt: string, endedAt: string | null, startedAt: string | null, status: FunctionRunStatus, hasAI: boolean, app: { __typename?: 'App', externalID: string, name: string }, function: { __typename?: 'Workflow', name: string, slug: string } } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

export type CountRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  endTime: InputMaybe<Scalars['Time']>;
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsOrderByField;
  functionSlug: InputMaybe<Scalars['String']>;
  celQuery?: InputMaybe<Scalars['String']>;
}>;


export type CountRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', runs: { __typename?: 'RunsConnection', totalCount: number } } };

export type AppFilterQueryVariables = Exact<{
  envSlug: Scalars['String'];
}>;


export type AppFilterQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', externalID: string, id: string, name: string }> } | null };

export type GetWorkerConnectionsQueryVariables = Exact<{
  envID: Scalars['ID'];
  appID: Scalars['UUID'];
  startTime: Scalars['Time'];
  status: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
  timeField: ConnectV1WorkerConnectionsOrderByField;
  cursor?: InputMaybe<Scalars['String']>;
  orderBy?: InputMaybe<Array<ConnectV1WorkerConnectionsOrderBy> | ConnectV1WorkerConnectionsOrderBy>;
  first: Scalars['Int'];
}>;


export type GetWorkerConnectionsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', edges: Array<{ __typename?: 'ConnectV1WorkerConnectionEdge', node: { __typename?: 'ConnectV1WorkerConnection', id: string, gatewayId: string, workerIp: string, connectedAt: string, lastHeartbeatAt: string | null, disconnectedAt: string | null, disconnectReason: string | null, status: ConnectV1ConnectionStatus, sdkLang: string, sdkVersion: string, sdkPlatform: string, functionCount: number, cpuCores: number, memBytes: number, os: string, instanceID: string, appVersion: string | null, app: { __typename?: 'App', id: string } | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

export type GetWorkerCountConnectionsQueryVariables = Exact<{
  envID: Scalars['ID'];
  appID: Scalars['UUID'];
  startTime: Scalars['Time'];
  status?: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
  timeField: ConnectV1WorkerConnectionsOrderByField;
}>;


export type GetWorkerCountConnectionsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number } } };

export type GetDeployssQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetDeployssQuery = { __typename?: 'Query', deploys: Array<{ __typename?: 'Deploy', id: string, appName: string, authorID: string | null, checksum: string, createdAt: string, error: string | null, framework: string | null, metadata: Record<string, unknown>, sdkLanguage: string, sdkVersion: string, status: string, deployedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }>, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }> }> | null };

export type GetEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetEnvironmentsQuery = { __typename?: 'Query', workspaces: Array<{ __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, webhookSigningKey: string, createdAt: string, isArchived: boolean, isAutoArchiveEnabled: boolean, lastDeployedAt: string | null }> | null };

export type GetEnvironmentBySlugQueryVariables = Exact<{
  slug: Scalars['String'];
}>;


export type GetEnvironmentBySlugQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } | null };

export type GetDefaultEnvironmentQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDefaultEnvironmentQuery = { __typename?: 'Query', defaultEnv: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean } };

export type GetEventTypesQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
}>;


export type GetEventTypesQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, functions: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string }> }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null } } } };

export type GetEventTypesVolumeQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
}>;


export type GetEventTypesVolumeQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', name: string, dailyVolume: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null } } } };

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


export type GetFunctionsUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, dailyStarts: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyCompleted: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyCancelled: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyFailures: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }> } } };

export type GetFunctionsQueryVariables = Exact<{
  environmentID: Scalars['ID'];
  page: InputMaybe<Scalars['Int']>;
  archived: InputMaybe<Scalars['Boolean']>;
  search: InputMaybe<Scalars['String']>;
  pageSize: InputMaybe<Scalars['Int']>;
}>;


export type GetFunctionsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', appName: string | null, id: string, slug: string, name: string, isPaused: boolean, isArchived: boolean, current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null }> } | null }> } } };

export type GetFunctionQueryVariables = Exact<{
  slug: Scalars['String'];
  environmentID: Scalars['ID'];
}>;


export type GetFunctionQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', id: string, workflow: { __typename?: 'Workflow', id: string, name: string, slug: string, isPaused: boolean, isArchived: boolean, appName: string | null, current: { __typename?: 'WorkflowVersion', triggers: Array<{ __typename?: 'WorkflowTrigger', eventName: null | string | null, schedule: null | string | null, condition: null | string | null }>, deploy: { __typename?: 'Deploy', id: string, createdAt: string } | null } | null, failureHandler: { __typename?: 'Workflow', slug: string, name: string } | null, configuration: { __typename?: 'FunctionConfiguration', priority: string | null, cancellations: Array<{ __typename?: 'CancellationConfiguration', event: string, timeout: string | null, condition: string | null }> | null, retries: { __typename?: 'RetryConfiguration', value: number, isDefault: boolean | null }, eventsBatch: { __typename?: 'EventsBatchConfiguration', maxSize: number, timeout: string, key: string | null } | null, concurrency: Array<{ __typename?: 'ConcurrencyConfiguration', scope: ConcurrencyScope, key: string | null, limit: { __typename?: 'ConcurrencyLimitConfiguration', value: number, isPlanLimit: boolean | null } }>, rateLimit: { __typename?: 'RateLimitConfiguration', limit: number, period: string, key: string | null } | null, debounce: { __typename?: 'DebounceConfiguration', period: string, key: string | null } | null, throttle: { __typename?: 'ThrottleConfiguration', burst: number, key: string | null, limit: number, period: string } | null } | null } | null } };

export type GetFunctionUsageQueryVariables = Exact<{
  id: Scalars['ID'];
  environmentID: Scalars['ID'];
  startTime: Scalars['Time'];
  endTime: Scalars['Time'];
}>;


export type GetFunctionUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow: { __typename?: 'Workflow', dailyStarts: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyCancelled: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyCompleted: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyFailures: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetProductionWorkspaceQueryVariables = Exact<{ [key: string]: never; }>;


export type GetProductionWorkspaceQuery = { __typename?: 'Query', defaultEnv: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } };

export type ProfileQueryVariables = Exact<{ [key: string]: never; }>;


export type ProfileQuery = { __typename?: 'Query', account: { __typename?: 'Account', name: null | string | null, marketplace: Marketplace | null } };

export type CancelRunMutationVariables = Exact<{
  envID: Scalars['UUID'];
  runID: Scalars['ULID'];
}>;


export type CancelRunMutation = { __typename?: 'Mutation', cancelRun: { __typename?: 'FunctionRun', id: string } };

export type GetEventKeysForBlankSlateQueryVariables = Exact<{
  environmentID: Scalars['ID'];
}>;


export type GetEventKeysForBlankSlateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', name: null | string, presharedKey: string, createdAt: string }> } };

export type RerunFunctionRunMutationVariables = Exact<{
  environmentID: Scalars['ID'];
  functionID: Scalars['ID'];
  functionRunID: Scalars['ULID'];
}>;


export type RerunFunctionRunMutation = { __typename?: 'Mutation', retryWorkflowRun: { __typename?: 'StartWorkflowResponse', id: string } | null };

export type RerunMutationVariables = Exact<{
  runID: Scalars['ULID'];
  fromStep: InputMaybe<RerunFromStepInput>;
}>;


export type RerunMutation = { __typename?: 'Mutation', rerun: string };

export const EventPayloadFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<EventPayloadFragment, unknown>;
export const TraceDetailsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"stepID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<TraceDetailsFragment, unknown>;
export const SetUpAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SetUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"setUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<SetUpAccountMutation, SetUpAccountMutationVariables>;
export const CreateUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<CreateUserMutation, CreateUserMutationVariables>;
export const GetBillingInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetBillingInfoQuery, GetBillingInfoQueryVariables>;
export const CreateEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createWorkspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateEnvironmentMutation, CreateEnvironmentMutationVariables>;
export const AchiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AchiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<AchiveAppMutation, AchiveAppMutationVariables>;
export const UnachiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnachiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnachiveAppMutation, UnachiveAppMutationVariables>;
export const ResyncAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResyncApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resyncApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appExternalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}}},{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<ResyncAppMutation, ResyncAppMutationVariables>;
export const CheckAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CheckApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"url"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appCheck"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"url"},"value":{"kind":"Variable","name":{"kind":"Name","value":"url"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apiOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"appID"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"authenticationSucceeded"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"env"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"eventAPIOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventKeyStatus"}},{"kind":"Field","name":{"kind":"Name","value":"extra"}},{"kind":"Field","name":{"kind":"Name","value":"framework"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"isReachable"}},{"kind":"Field","name":{"kind":"Name","value":"isSDK"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"respHeaders"}},{"kind":"Field","name":{"kind":"Name","value":"respStatusCode"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"serveOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"servePath"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"signingKeyStatus"}},{"kind":"Field","name":{"kind":"Name","value":"signingKeyFallbackStatus"}}]}}]}}]}}]} as unknown as DocumentNode<CheckAppQuery, CheckAppQueryVariables>;
export const SyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Sync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<SyncQuery, SyncQueryVariables>;
export const AppSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"syncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppSyncsQuery, AppSyncsQueryVariables>;
export const AppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"App"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"latestVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppQuery, AppQueryVariables>;
export const AppNavDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppNavData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isParentArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<AppNavDataQuery, AppNavDataQueryVariables>;
export const SyncNewAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncNewApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncNewApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<SyncNewAppMutation, SyncNewAppMutationVariables>;
export const AppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Apps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}},{"kind":"Field","name":{"kind":"Name","value":"isParentArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppsQuery, AppsQueryVariables>;
export const LatestUnattachedSyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LatestUnattachedSync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}}]}}]} as unknown as DocumentNode<LatestUnattachedSyncQuery, LatestUnattachedSyncQueryVariables>;
export const GetHistoryItemOutputDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetHistoryItemOutput"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"historyItemOutput"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"historyItemID"}}}]}]}}]}}]}}]}}]} as unknown as DocumentNode<GetHistoryItemOutputQuery, GetHistoryItemOutputQueryVariables>;
export const SearchEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"eventSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<SearchEventsQuery, SearchEventsQueryVariables>;
export const GetEventSearchEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventSearchEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"event"},"name":{"kind":"Name","value":"archivedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","alias":{"kind":"Name","value":"runs"},"name":{"kind":"Name","value":"functionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventSearchEventQuery, GetEventSearchEventQueryVariables>;
export const GetEventSearchRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventSearchRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"canRerun"}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"attempt"}},{"kind":"Field","name":{"kind":"Name","value":"cancel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionVersion"}},{"kind":"Field","name":{"kind":"Name","value":"groupID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"sleep"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"until"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"waitForEvent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}},{"kind":"Field","name":{"kind":"Name","value":"waitResult"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"output"}},{"kind":"Field","alias":{"kind":"Name","value":"version"},"name":{"kind":"Name","value":"workflowVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"validFrom"}},{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventSearchRunQuery, GetEventSearchRunQueryVariables>;
export const GetEventLogDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventLog"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventType"},"name":{"kind":"Name","value":"event"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"events"},"name":{"kind":"Name","value":"recent"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"cursored"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"cursor"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"perPage"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventLogQuery, GetEventLogQueryVariables>;
export const GetFunctionNameSlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionNameSlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionNameSlugQuery, GetFunctionNameSlugQueryVariables>;
export const GetFunctionRunCardDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunCard"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunCardQuery, GetFunctionRunCardQueryVariables>;
export const GetEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"event"},"name":{"kind":"Name","value":"archivedEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"EventPayload"}},{"kind":"Field","name":{"kind":"Name","value":"functionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"skippedFunctionRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"skipReason"}},{"kind":"Field","name":{"kind":"Name","value":"workflowID"}},{"kind":"Field","name":{"kind":"Name","value":"skippedAt"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"EventPayload"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"payload"},"name":{"kind":"Name","value":"event"}}]}}]} as unknown as DocumentNode<GetEventQuery, GetEventQueryVariables>;
export const GetAccountEntitlementsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountEntitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountEntitlementsQuery, GetAccountEntitlementsQueryVariables>;
export const GetFunctionRateLimitDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRateLimitDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"ratelimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_rate_limited_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRateLimitDocumentQuery, GetFunctionRateLimitDocumentQueryVariables>;
export const GetFunctionRunsMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"canceled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"failed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsMetricsQuery, GetFunctionRunsMetricsQueryVariables>;
export const GetFnMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnMetricsQuery, GetFnMetricsQueryVariables>;
export const GetFailedFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFailedFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"failedRuns"},"name":{"kind":"Name","value":"runsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"lowerTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lowerTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"FAILED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"ENDED_AT"}},{"kind":"ObjectField","name":{"kind":"Name","value":"upperTime"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upperTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFailedFunctionRunsQuery, GetFailedFunctionRunsQueryVariables>;
export const GetSdkRequestMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSDKRequestMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSdkRequestMetricsQuery, GetSdkRequestMetricsQueryVariables>;
export const GetStepBacklogMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepBacklogMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"scheduled"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_scheduled","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sleeping"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_sleeping","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepBacklogMetricsQuery, GetStepBacklogMetricsQueryVariables>;
export const GetStepsRunningMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepsRunningMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"running"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"concurrencyLimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepsRunningMetricsQuery, GetStepsRunningMetricsQueryVariables>;
export const DeleteCancellationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteCancellation"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cancellationID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteCancellation"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"cancellationID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cancellationID"}}}]}]}}]} as unknown as DocumentNode<DeleteCancellationMutation, DeleteCancellationMutationVariables>;
export const GetFnCancellationsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnCancellations"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"fn"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellations"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cursor"}},{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","alias":{"kind":"Name","value":"envID"},"name":{"kind":"Name","value":"environmentID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAtMax"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAtMin"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnCancellationsQuery, GetFnCancellationsQueryVariables>;
export const InvokeFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"InvokeFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"data"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"user"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"invokeFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"data"},"value":{"kind":"Variable","name":{"kind":"Name","value":"data"}}},{"kind":"Argument","name":{"kind":"Name","value":"functionSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"user"},"value":{"kind":"Variable","name":{"kind":"Name","value":"user"}}}]}]}}]} as unknown as DocumentNode<InvokeFunctionMutation, InvokeFunctionMutationVariables>;
export const GetReplaysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplays"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"replays"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsScheduledCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplaysQuery, GetReplaysQueryVariables>;
export const GetFunctionPauseStateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionPauseState"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionPauseStateQuery, GetFunctionPauseStateQueryVariables>;
export const NewIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"NewIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<NewIngestKeyMutation, NewIngestKeyMutationVariables>;
export const GetIngestKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeysQuery, GetIngestKeysQueryVariables>;
export const UpdateIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}}]}}]}}]} as unknown as DocumentNode<UpdateIngestKeyMutation, UpdateIngestKeyMutationVariables>;
export const DeleteEventKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteEventKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DeleteIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<DeleteEventKeyMutation, DeleteEventKeyMutationVariables>;
export const GetIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeyQuery, GetIngestKeyQueryVariables>;
export const CreateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<CreateSigningKeyMutation, CreateSigningKeyMutationVariables>;
export const DeleteSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<DeleteSigningKeyMutation, DeleteSigningKeyMutationVariables>;
export const RotateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RotateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rotateSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<RotateSigningKeyMutation, RotateSigningKeyMutationVariables>;
export const GetSigningKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSigningKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"signingKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"decryptedValue"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isActive"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSigningKeysQuery, GetSigningKeysQueryVariables>;
export const UnattachedSyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncQuery, UnattachedSyncQueryVariables>;
export const UnattachedSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"syncs"},"name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncsQuery, UnattachedSyncsQueryVariables>;
export const VercelIntegrationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"VercelIntegration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"vercelIntegration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"isMarketplace"}},{"kind":"Field","name":{"kind":"Name","value":"projects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"canChangeEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"deploymentProtection"}},{"kind":"Field","name":{"kind":"Name","value":"isEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}},{"kind":"Field","name":{"kind":"Name","value":"servePath"}}]}}]}}]}}]}}]} as unknown as DocumentNode<VercelIntegrationQuery, VercelIntegrationQueryVariables>;
export const GetSavedVercelProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSavedVercelProjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"marketplace"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"savedVercelProjects"},"name":{"kind":"Name","value":"vercelApps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"workspaceID"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}}]}}]}}]}}]} as unknown as DocumentNode<GetSavedVercelProjectsQuery, GetSavedVercelProjectsQueryVariables>;
export const CreateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<CreateVercelAppMutation, CreateVercelAppMutationVariables>;
export const UpdateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<UpdateVercelAppMutation, UpdateVercelAppMutationVariables>;
export const RemoveVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RemoveVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<RemoveVercelAppMutation, RemoveVercelAppMutationVariables>;
export const CreateWebhookDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateWebhook"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]} as unknown as DocumentNode<CreateWebhookMutation, CreateWebhookMutationVariables>;
export const CompleteAwsMarketplaceSetupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CompleteAWSMarketplaceSetup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSMarketplaceSetupInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"completeAWSMarketplaceSetup"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CompleteAwsMarketplaceSetupMutation, CompleteAwsMarketplaceSetupMutationVariables>;
export const GetAccountSupportInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountSupportInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountSupportInfoQuery, GetAccountSupportInfoQueryVariables>;
export const GetArchivedAppBannerDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetArchivedAppBannerData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"isArchived"}}]}}]}}]}}]} as unknown as DocumentNode<GetArchivedAppBannerDataQuery, GetArchivedAppBannerDataQueryVariables>;
export const GetArchivedFuncBannerDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetArchivedFuncBannerData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"funcID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"funcID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetArchivedFuncBannerDataQuery, GetArchivedFuncBannerDataQueryVariables>;
export const UpdateAccountAddonQuantityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccountAddonQuantity"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"addonName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"quantity"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateAccountAddonQuantity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"addonName"},"value":{"kind":"Variable","name":{"kind":"Name","value":"addonName"}}},{"kind":"Argument","name":{"kind":"Name","value":"quantity"},"value":{"kind":"Variable","name":{"kind":"Name","value":"quantity"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountAddonQuantityMutation, UpdateAccountAddonQuantityMutationVariables>;
export const UpdateAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAccount"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"account"},"name":{"kind":"Name","value":"updateAccount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountMutation, UpdateAccountMutationVariables>;
export const UpdatePaymentMethodDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePaymentMethod"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePaymentMethod"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]} as unknown as DocumentNode<UpdatePaymentMethodMutation, UpdatePaymentMethodMutationVariables>;
export const GetPaymentIntentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPaymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"paymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"amountLabel"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"invoiceURL"}}]}}]}}]}}]} as unknown as DocumentNode<GetPaymentIntentsQuery, GetPaymentIntentsQueryVariables>;
export const CreateStripeSubscriptionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripeSubscription"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StripeSubscriptionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createStripeSubscription"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clientSecret"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CreateStripeSubscriptionMutation, CreateStripeSubscriptionMutationVariables>;
export const UpdatePlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<UpdatePlanMutation, UpdatePlanMutationVariables>;
export const GetBillableStepsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableSteps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"usage"},"name":{"kind":"Name","value":"billableStepTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableStepsQuery, GetBillableStepsQueryVariables>;
export const GetBillableRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"usage"},"name":{"kind":"Name","value":"runCountTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableRunsQuery, GetBillableRunsQueryVariables>;
export const EntitlementUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EntitlementUsage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"addons"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"hipaa"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExport"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<EntitlementUsageQuery, EntitlementUsageQueryVariables>;
export const GetCurrentPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCurrentPlan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"addons"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nextInvoiceDate"}}]}}]}}]}}]} as unknown as DocumentNode<GetCurrentPlanQuery, GetCurrentPlanQueryVariables>;
export const GetBillingDetailsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingDetails"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"paymentMethods"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillingDetailsQuery, GetBillingDetailsQueryVariables>;
export const GetPlansDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPlans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetPlansQuery, GetPlansQueryVariables>;
export const ArchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<ArchiveEnvironmentMutation, ArchiveEnvironmentMutationVariables>;
export const UnarchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnarchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnarchiveEnvironmentMutation, UnarchiveEnvironmentMutationVariables>;
export const DisableEnvironmentAutoArchiveDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisableEnvironmentAutoArchiveDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<DisableEnvironmentAutoArchiveDocumentMutation, DisableEnvironmentAutoArchiveDocumentMutationVariables>;
export const EnableEnvironmentAutoArchiveDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EnableEnvironmentAutoArchive"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<EnableEnvironmentAutoArchiveMutation, EnableEnvironmentAutoArchiveMutationVariables>;
export const ArchiveEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}}},{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<ArchiveEventMutation, ArchiveEventMutationVariables>;
export const GetLatestEventLogsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLatestEventLogs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"recent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"count"},"value":{"kind":"IntValue","value":"5"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetLatestEventLogsQuery, GetLatestEventLogsQueryVariables>;
export const GetEventKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventKeys"},"name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"value"},"name":{"kind":"Name","value":"presharedKey"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysQuery, GetEventKeysQueryVariables>;
export const CreateCancellationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateCancellation"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateCancellationInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createCancellation"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateCancellationMutation, CreateCancellationMutationVariables>;
export const GetCancellationRunCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCancellationRunCount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMin"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMax"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellationRunCount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"queuedAtMin"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMin"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"queuedAtMax"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMax"}}}]}}]}]}}]}}]}}]} as unknown as DocumentNode<GetCancellationRunCountQuery, GetCancellationRunCountQueryVariables>;
export const PauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cancelRunning"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}},{"kind":"Argument","name":{"kind":"Name","value":"cancelRunning"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cancelRunning"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<PauseFunctionMutation, PauseFunctionMutationVariables>;
export const UnpauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnpauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unpauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnpauseFunctionMutation, UnpauseFunctionMutationVariables>;
export const MetricsLookupsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"MetricsLookups"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}}]}},{"kind":"Field","name":{"kind":"Name","value":"workflows"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}}]}}]}}]}}]}}]} as unknown as DocumentNode<MetricsLookupsQuery, MetricsLookupsQueryVariables>;
export const AccountConcurrencyLookupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AccountConcurrencyLookup"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AccountConcurrencyLookupQuery, AccountConcurrencyLookupQueryVariables>;
export const FunctionStatusMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"FunctionStatusMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"until"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"scope"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"MetricsScope"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"scheduled"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"groupBy"},"value":{"kind":"StringValue","value":"status","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completedByFunction"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"EnumValue","value":"FN"}},{"kind":"ObjectField","name":{"kind":"Name","value":"groupBy"},"value":{"kind":"StringValue","value":"status","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"totals"},"name":{"kind":"Name","value":"scopedFunctionStatus"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"EnumValue","value":"FN"}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queued"}},{"kind":"Field","name":{"kind":"Name","value":"running"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}},{"kind":"Field","name":{"kind":"Name","value":"cancelled"}},{"kind":"Field","name":{"kind":"Name","value":"cancelled"}},{"kind":"Field","name":{"kind":"Name","value":"skipped"}}]}}]}}]}}]} as unknown as DocumentNode<FunctionStatusMetricsQuery, FunctionStatusMetricsQueryVariables>;
export const VolumeMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"VolumeMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"until"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"scope"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"MetricsScope"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"runsThroughput"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputEnded"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputStarted"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputScheduled"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"stepThroughput"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"backlog"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_scheduled","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"stepRunning"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"concurrency"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<VolumeMetricsQuery, VolumeMetricsQueryVariables>;
export const QuickSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"QuickSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"term"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"quickSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"term"},"value":{"kind":"Variable","name":{"kind":"Name","value":"term"}}},{"kind":"Argument","name":{"kind":"Name","value":"envSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"event"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"run"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]}}]} as unknown as DocumentNode<QuickSearchQuery, QuickSearchQueryVariables>;
export const GetGlobalSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGlobalSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"opts"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"Variable","name":{"kind":"Name","value":"opts"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"env"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"ArchivedEvent"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"functionID"},"name":{"kind":"Name","value":"workflowID"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetGlobalSearchQuery, GetGlobalSearchQueryVariables>;
export const GetFunctionSlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionSlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionSlugQuery, GetFunctionSlugQueryVariables>;
export const SyncOnboardingAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncOnboardingApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncNewApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<SyncOnboardingAppMutation, SyncOnboardingAppMutationVariables>;
export const InvokeFunctionOnboardingDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"InvokeFunctionOnboarding"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"data"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"user"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"invokeFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"data"},"value":{"kind":"Variable","name":{"kind":"Name","value":"data"}}},{"kind":"Argument","name":{"kind":"Name","value":"functionSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"user"},"value":{"kind":"Variable","name":{"kind":"Name","value":"user"}}}]}]}}]} as unknown as DocumentNode<InvokeFunctionOnboardingMutation, InvokeFunctionOnboardingMutationVariables>;
export const InvokeFunctionLookupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"InvokeFunctionLookup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}}]}}]}}]}}]}}]} as unknown as DocumentNode<InvokeFunctionLookupQuery, InvokeFunctionLookupQueryVariables>;
export const GetVercelAppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetVercelApps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}}]}},{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetVercelAppsQuery, GetVercelAppsQueryVariables>;
export const ProductionAppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ProductionApps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}}]}}]} as unknown as DocumentNode<ProductionAppsQuery, ProductionAppsQueryVariables>;
export const GetPostgresIntegrationsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"getPostgresIntegrations"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"statusDetail"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode<GetPostgresIntegrationsQuery, GetPostgresIntegrationsQueryVariables>;
export const TestCredentialsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testCredentials"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcTestCredentials"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestCredentialsMutation, TestCredentialsMutationVariables>;
export const TestReplicationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testReplication"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcTestLogicalReplication"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestReplicationMutation, TestReplicationMutationVariables>;
export const TestAutoSetupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testAutoSetup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcAutoSetup"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestAutoSetupMutation, TestAutoSetupMutationVariables>;
export const CdcDeleteDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"cdcDelete"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcDelete"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<CdcDeleteMutation, CdcDeleteMutationVariables>;
export const MetricsEntitlementsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"MetricsEntitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metricsExport"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<MetricsEntitlementsQuery, MetricsEntitlementsQueryVariables>;
export const GetReplayRunCountsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplayRunCounts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"to"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"replayCounts"},"name":{"kind":"Name","value":"replayCounts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"Argument","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"to"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"cancelledCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedPausedCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplayRunCountsQuery, GetReplayRunCountsQueryVariables>;
export const CreateFunctionReplayDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateFunctionReplay"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ReplayRunStatus"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createFunctionReplay"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fromRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"toRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"statusesV2"},"value":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateFunctionReplayMutation, CreateFunctionReplayMutationVariables>;
export const GetRunTraceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTrace"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"trace"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"hasAI"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"stepID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceQuery, GetRunTraceQueryVariables>;
export const TraceResultDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TraceResult"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"traceID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runTraceSpanOutputByID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"outputID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"traceID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"input"}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"stack"}}]}}]}}]}}]}}]} as unknown as DocumentNode<TraceResultQuery, TraceResultQueryVariables>;
export const GetRunTraceTriggerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTraceTrigger"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runTrigger"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"IDs"}},{"kind":"Field","name":{"kind":"Name","value":"payloads"}},{"kind":"Field","name":{"kind":"Name","value":"timestamp"}},{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"isBatch"}},{"kind":"Field","name":{"kind":"Name","value":"batchID"}},{"kind":"Field","name":{"kind":"Name","value":"cron"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceTriggerQuery, GetRunTraceTriggerQueryVariables>;
export const GetRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RunsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"cronSchedule"}},{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isBatch"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"hasAI"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetRunsQuery, GetRunsQueryVariables>;
export const CountRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CountRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RunsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<CountRunsQuery, CountRunsQueryVariables>;
export const AppFilterDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppFilter"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<AppFilterQuery, AppFilterQueryVariables>;
export const GetWorkerConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetWorkerConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1ConnectionStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"orderBy"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderBy"}}}},"defaultValue":{"kind":"ListValue","values":[]}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"first"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workerConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"Variable","name":{"kind":"Name","value":"first"}}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"ListValue","values":[{"kind":"Variable","name":{"kind":"Name","value":"appID"}}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"Variable","name":{"kind":"Name","value":"orderBy"}}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"gatewayId"}},{"kind":"Field","alias":{"kind":"Name","value":"instanceID"},"name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"workerIp"}},{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastHeartbeatAt"}},{"kind":"Field","name":{"kind":"Name","value":"disconnectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"disconnectReason"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLang"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"sdkPlatform"}},{"kind":"Field","alias":{"kind":"Name","value":"appVersion"},"name":{"kind":"Name","value":"buildId"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"cpuCores"}},{"kind":"Field","name":{"kind":"Name","value":"memBytes"}},{"kind":"Field","name":{"kind":"Name","value":"os"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetWorkerConnectionsQuery, GetWorkerConnectionsQueryVariables>;
export const GetWorkerCountConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetWorkerCountConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1ConnectionStatus"}}}},"defaultValue":{"kind":"ListValue","values":[]}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderByField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workerConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"ListValue","values":[{"kind":"Variable","name":{"kind":"Name","value":"appID"}}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<GetWorkerCountConnectionsQuery, GetWorkerCountConnectionsQueryVariables>;
export const GetDeployssDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDeployss"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"checksum"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetDeployssQuery, GetDeployssQueryVariables>;
export const GetEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentsQuery, GetEnvironmentsQueryVariables>;
export const GetEnvironmentBySlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironmentBySlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentBySlugQuery, GetEnvironmentBySlugQueryVariables>;
export const GetDefaultEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDefaultEnvironment"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"defaultEnv"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}}]}}]}}]} as unknown as DocumentNode<GetDefaultEnvironmentQuery, GetDefaultEnvironmentQueryVariables>;
export const GetEventTypesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"functions"},"name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypesQuery, GetEventTypesQueryVariables>;
export const GetEventTypesVolumeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypesVolume"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"dailyVolume"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypesVolumeQuery, GetEventTypesVolumeQueryVariables>;
export const GetEventTypeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventType"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workflows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeQuery, GetEventTypeQueryVariables>;
export const GetFunctionsUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionsUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCompleted"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCancelled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsUsageQuery, GetFunctionsUsageQueryVariables>;
export const GetFunctionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}},{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsQuery, GetFunctionsQueryVariables>;
export const GetFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"current"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"schedule"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"deploy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"failureHandler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"configuration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"retries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isDefault"}}]}},{"kind":"Field","name":{"kind":"Name","value":"priority"}},{"kind":"Field","name":{"kind":"Name","value":"eventsBatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"maxSize"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"limit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isPlanLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"rateLimit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"debounce"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"throttle"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"burst"}},{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionQuery, GetFunctionQueryVariables>;
export const GetFunctionUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCancelled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCompleted"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionUsageQuery, GetFunctionUsageQueryVariables>;
export const GetProductionWorkspaceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetProductionWorkspace"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"defaultEnv"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetProductionWorkspaceQuery, GetProductionWorkspaceQueryVariables>;
export const ProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Profile"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"marketplace"}}]}}]}}]} as unknown as DocumentNode<ProfileQuery, ProfileQueryVariables>;
export const CancelRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CancelRunMutation, CancelRunMutationVariables>;
export const GetEventKeysForBlankSlateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeysForBlankSlate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"source"},"value":{"kind":"StringValue","value":"key","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysForBlankSlateQuery, GetEventKeysForBlankSlateQueryVariables>;
export const RerunFunctionRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RerunFunctionRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryWorkflowRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"workflowRunID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<RerunFunctionRunMutation, RerunFunctionRunMutationVariables>;
export const RerunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"Rerun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fromStep"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"RerunFromStepInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rerun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}},{"kind":"Argument","name":{"kind":"Name","value":"fromStep"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fromStep"}}}]}]}}]} as unknown as DocumentNode<RerunMutation, RerunMutationVariables>;