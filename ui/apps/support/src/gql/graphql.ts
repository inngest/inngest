/* eslint-disable */
import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = T | null | undefined;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  BillingPeriod: { input: unknown; output: unknown; }
  Bytes: { input: unknown; output: unknown; }
  DSN: { input: unknown; output: unknown; }
  EdgeType: { input: unknown; output: unknown; }
  FilterType: { input: unknown; output: unknown; }
  HTTPHeaders: { input: unknown; output: unknown; }
  IP: { input: unknown; output: unknown; }
  IngestSource: { input: unknown; output: unknown; }
  InsightsDiagnosticCode: { input: unknown; output: unknown; }
  InsightsDiagnosticSeverity: { input: unknown; output: unknown; }
  Int64: { input: unknown; output: unknown; }
  JSON: { input: unknown; output: unknown; }
  Map: { input: unknown; output: unknown; }
  NullString: { input: null | string; output: null | string; }
  NullTime: { input: null | string; output: null | string; }
  Period: { input: unknown; output: unknown; }
  Role: { input: unknown; output: unknown; }
  Runtime: { input: unknown; output: unknown; }
  SchemaSource: { input: unknown; output: unknown; }
  SearchObject: { input: unknown; output: unknown; }
  SpanMetadataKind: { input: unknown; output: unknown; }
  SpanMetadataScope: { input: unknown; output: unknown; }
  SpanMetadataValues: { input: unknown; output: unknown; }
  Time: { input: string; output: string; }
  Timerange: { input: unknown; output: unknown; }
  ULID: { input: string; output: string; }
  UUID: { input: string; output: string; }
  Upload: { input: unknown; output: unknown; }
};

export type AwsMarketplaceSetupInput = {
  awsAccountID: Scalars['String']['input'];
  customerID: Scalars['String']['input'];
  productCode: Scalars['String']['input'];
};

export enum AppMethod {
  Api = 'API',
  Connect = 'CONNECT',
  Serve = 'SERVE'
}

export type AppsFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  method?: InputMaybe<AppMethod>;
};

export type ArchiveWorkflowInput = {
  archive: Scalars['Boolean']['input'];
  workflowID: Scalars['ID']['input'];
};

export enum BannerSeverity {
  Error = 'ERROR',
  Info = 'INFO',
  Success = 'SUCCESS',
  Warning = 'WARNING'
}

export type CdcConnectionInput = {
  adminConn: Scalars['String']['input'];
  engine: Scalars['String']['input'];
  name: Scalars['String']['input'];
  replicaConn?: InputMaybe<Scalars['String']['input']>;
};

export enum CdcStatus {
  Error = 'ERROR',
  Running = 'RUNNING',
  SetupComplete = 'SETUP_COMPLETE',
  SetupIncomplete = 'SETUP_INCOMPLETE',
  Stopped = 'STOPPED'
}

export type CancellationRunCountInput = {
  queuedAtMax: Scalars['Time']['input'];
  queuedAtMin?: InputMaybe<Scalars['Time']['input']>;
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

export type ConnectV1WorkerConnectionsFilter = {
  appIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  from?: InputMaybe<Scalars['Time']['input']>;
  status?: InputMaybe<Array<ConnectV1ConnectionStatus>>;
  timeField?: InputMaybe<ConnectV1WorkerConnectionsOrderByField>;
  until?: InputMaybe<Scalars['Time']['input']>;
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

export type ConnectV1WorkerMetricsFilter = {
  from: Scalars['Time']['input'];
  instanceIDs?: InputMaybe<Array<Scalars['String']['input']>>;
  name: Scalars['String']['input'];
  until?: InputMaybe<Scalars['Time']['input']>;
};

export type CreateApiKeyInput = {
  name: Scalars['String']['input'];
  workspaceID: Scalars['UUID']['input'];
};

export type CreateCancellationInput = {
  envID: Scalars['UUID']['input'];
  expression?: InputMaybe<Scalars['String']['input']>;
  functionSlug: Scalars['String']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
  queuedAtMax: Scalars['Time']['input'];
  queuedAtMin?: InputMaybe<Scalars['Time']['input']>;
  testOnly?: InputMaybe<CreateCancellationInputTestOnly>;
};

export type CreateCancellationInputTestOnly = {
  maxStepCount?: InputMaybe<Scalars['Int']['input']>;
  queryLimit?: InputMaybe<Scalars['Int']['input']>;
};

export type CreateFunctionReplayInput = {
  fromRange: Scalars['ULID']['input'];
  name: Scalars['String']['input'];
  statuses?: InputMaybe<Array<FunctionRunStatus>>;
  statusesV2?: InputMaybe<Array<ReplayRunStatus>>;
  toRange: Scalars['ULID']['input'];
  workflowID: Scalars['UUID']['input'];
  workspaceID: Scalars['UUID']['input'];
};

export type CreateVercelAppInput = {
  originOverride?: InputMaybe<Scalars['String']['input']>;
  path?: InputMaybe<Scalars['String']['input']>;
  projectID: Scalars['String']['input'];
  protectionBypassSecret?: InputMaybe<Scalars['String']['input']>;
  workspaceID: Scalars['ID']['input'];
};

export type DeleteIngestKey = {
  id: Scalars['ID']['input'];
  workspaceID: Scalars['ID']['input'];
};

export enum EnvironmentType {
  BranchChild = 'BRANCH_CHILD',
  BranchParent = 'BRANCH_PARENT',
  Production = 'PRODUCTION',
  Test = 'TEST'
}

export type EnvsFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  envTypes?: InputMaybe<Array<EnvironmentType>>;
};

export type EventQuery = {
  name?: InputMaybe<Scalars['String']['input']>;
  prefix?: InputMaybe<Scalars['String']['input']>;
  schemaSource?: InputMaybe<Scalars['SchemaSource']['input']>;
  workspaceID?: InputMaybe<Scalars['ID']['input']>;
};

export enum EventSchemaFormat {
  JsonSchema = 'JSON_SCHEMA',
  Typescript = 'TYPESCRIPT'
}

export type EventTypesFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  nameSearch?: InputMaybe<Scalars['String']['input']>;
};

export type EventsFilter = {
  eventNames?: InputMaybe<Array<Scalars['String']['input']>>;
  from: Scalars['Time']['input'];
  includeInternalEvents?: Scalars['Boolean']['input'];
  query?: InputMaybe<Scalars['String']['input']>;
  until?: InputMaybe<Scalars['Time']['input']>;
};

export type ExperimentScoringMetricInput = {
  displayName: Scalars['String']['input'];
  enabled: Scalars['Boolean']['input'];
  invert: Scalars['Boolean']['input'];
  key: Scalars['String']['input'];
  kind?: InputMaybe<ScoreKind>;
  labelBest: Scalars['String']['input'];
  labelWorst: Scalars['String']['input'];
  maxValue: Scalars['Float']['input'];
  minValue: Scalars['Float']['input'];
  points: Scalars['Int']['input'];
};

export type FilterListInput = {
  events?: InputMaybe<Array<Scalars['String']['input']>>;
  ips?: InputMaybe<Array<Scalars['IP']['input']>>;
  type?: InputMaybe<Scalars['FilterType']['input']>;
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
  Skipped = 'SKIPPED',
  Unknown = 'UNKNOWN'
}

export enum FunctionRunTimeField {
  EndedAt = 'ENDED_AT',
  Mixed = 'MIXED',
  StartedAt = 'STARTED_AT'
}

export enum FunctionTriggerTypes {
  Cron = 'CRON',
  Event = 'EVENT'
}

export type FunctionsFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  eventName?: InputMaybe<Scalars['String']['input']>;
};

export type IngestKeyFilter = {
  name?: InputMaybe<Scalars['String']['input']>;
  source?: InputMaybe<Scalars['String']['input']>;
};

export enum InsightsColumnType {
  Date = 'DATE',
  Dynamic = 'DYNAMIC',
  Number = 'NUMBER',
  String = 'STRING',
  Unknown = 'UNKNOWN'
}

export type InsightsMetricOrderBy = {
  column: Scalars['String']['input'];
  direction: InsightsMetricOrderByDirection;
};

export enum InsightsMetricOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export enum Marketplace {
  Aws = 'AWS',
  DigitalOcean = 'DIGITAL_OCEAN',
  Partner = 'PARTNER',
  Vercel = 'VERCEL'
}

export type MetricsOpts = {
  from: Scalars['Time']['input'];
  name: Scalars['String']['input'];
  to?: InputMaybe<Scalars['Time']['input']>;
};

export type MetricsRequest = {
  from: Scalars['Time']['input'];
  name: Scalars['String']['input'];
  to: Scalars['Time']['input'];
};

export enum MetricsScope {
  App = 'APP',
  Env = 'ENV',
  Fn = 'FN'
}

export type NewIngestKey = {
  filterList?: InputMaybe<FilterListInput>;
  metadata?: InputMaybe<Scalars['Map']['input']>;
  name: Scalars['String']['input'];
  source: Scalars['IngestSource']['input'];
  workspaceID: Scalars['ID']['input'];
};

export type NewInsightsQuery = {
  name: Scalars['String']['input'];
  sql: Scalars['String']['input'];
};

export type NewUser = {
  email: Scalars['String']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
};

export type NewWorkspaceInput = {
  name: Scalars['String']['input'];
};

export enum PaymentCollectionStage {
  /** already downgraded for non-payment */
  Downgraded = 'DOWNGRADED',
  /** scheduled to downgrade on actionDate */
  DowngradePending = 'DOWNGRADE_PENDING',
  /** overdue beyond stricter threshold */
  FinalNotice = 'FINAL_NOTICE',
  /** overdue, within grace window */
  PastDue = 'PAST_DUE',
  /** card declined / retrying, not yet past grace */
  PaymentFailed = 'PAYMENT_FAILED',
  /** account suspended for non-payment */
  Suspended = 'SUSPENDED'
}

export enum PaymentPendingAction {
  Downgrade = 'DOWNGRADE',
  Suspend = 'SUSPEND'
}

export enum PaymentStatusSeverity {
  /** past stricter threshold, downgrade/suspension imminent or active */
  Critical = 'CRITICAL',
  /** failed payment / within grace window */
  Warning = 'WARNING'
}

export type RemoveVercelAppInput = {
  projectID: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
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
  input?: InputMaybe<Scalars['Bytes']['input']>;
  stepID: Scalars['String']['input'];
};

export enum RunDeferStatus {
  Aborted = 'ABORTED',
  Rejected = 'REJECTED',
  Scheduled = 'SCHEDULED'
}

export enum RunTraceSpanStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Paused = 'PAUSED',
  Running = 'RUNNING',
  Waiting = 'WAITING'
}

export type RunsFilter = {
  lowerTime: Scalars['Time']['input'];
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<FunctionRunTimeField>;
  upperTime: Scalars['Time']['input'];
};

export type RunsFilterV2 = {
  appIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  fnSlug?: InputMaybe<Scalars['String']['input']>;
  from: Scalars['Time']['input'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  isDeferred?: InputMaybe<Scalars['Boolean']['input']>;
  query?: InputMaybe<Scalars['String']['input']>;
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<RunsOrderByField>;
  until?: InputMaybe<Scalars['Time']['input']>;
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

export type ScopedMetricsFilter = {
  appIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  from: Scalars['Time']['input'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  groupBy?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
  scope: MetricsScope;
  until?: InputMaybe<Scalars['Time']['input']>;
};

export type ScoreFilter = {
  timeRange: TimeRangeInput;
};

export enum ScoreKind {
  Boolean = 'BOOLEAN',
  Numeric = 'NUMERIC'
}

export type SearchInput = {
  term: Scalars['String']['input'];
};

export enum SearchResultType {
  EventObject = 'EVENT_OBJECT',
  FunctionRun = 'FUNCTION_RUN'
}

export enum SecretCheck {
  Correct = 'CORRECT',
  Incorrect = 'INCORRECT',
  Missing = 'MISSING',
  Unknown = 'UNKNOWN'
}

export enum SingletonMode {
  Cancel = 'CANCEL',
  Skip = 'SKIP'
}

export enum SkipReason {
  FunctionPaused = 'FUNCTION_PAUSED',
  None = 'NONE'
}

export type StartWorkflowInput = {
  workflowID: Scalars['ID']['input'];
  workflowVersion?: InputMaybe<Scalars['Int']['input']>;
  workspaceID: Scalars['ID']['input'];
};

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
  amount: Scalars['Int']['input'];
  planID?: InputMaybe<Scalars['ID']['input']>;
  planSlug?: InputMaybe<Scalars['String']['input']>;
  quantity: Scalars['Int']['input'];
};

export enum SyncStatus {
  Duplicate = 'duplicate',
  Error = 'error',
  Pending = 'pending',
  Success = 'success'
}

export type TimeRangeInput = {
  from: Scalars['Time']['input'];
  to: Scalars['Time']['input'];
};

export type TimeSeriesOptions = {
  interval?: InputMaybe<Scalars['String']['input']>;
  month: Scalars['Int']['input'];
  year: Scalars['Int']['input'];
};

export type UpdateApiKeyInput = {
  id: Scalars['UUID']['input'];
  name: Scalars['String']['input'];
};

export type UpdateAccount = {
  billingEmail?: InputMaybe<Scalars['String']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
  securityEmail?: InputMaybe<Scalars['String']['input']>;
};

export type UpdateIngestKey = {
  filterList?: InputMaybe<FilterListInput>;
  metadata?: InputMaybe<Scalars['Map']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
};

export type UpdateInsightsQuery = {
  name: Scalars['String']['input'];
  sql: Scalars['String']['input'];
};

export type UpdateVercelAppInput = {
  originOverride?: InputMaybe<Scalars['String']['input']>;
  path: Scalars['String']['input'];
  projectID: Scalars['String']['input'];
  protectionBypassSecret?: InputMaybe<Scalars['String']['input']>;
};

export type UsageInput = {
  from?: InputMaybe<Scalars['Time']['input']>;
  period?: InputMaybe<Scalars['Period']['input']>;
  range?: InputMaybe<Scalars['Timerange']['input']>;
  to?: InputMaybe<Scalars['Time']['input']>;
};

export enum VercelDeploymentProtection {
  All = 'ALL',
  AllExceptCustomDomains = 'ALL_EXCEPT_CUSTOM_DOMAINS',
  Disabled = 'DISABLED',
  Preview = 'PREVIEW',
  ProdDeploymentUrlsAndAllPreviews = 'PROD_DEPLOYMENT_URLS_AND_ALL_PREVIEWS',
  Unknown = 'UNKNOWN'
}

export type GetEnvironmentBySlugQueryVariables = Exact<{
  slug: Scalars['String']['input'];
}>;


export type GetEnvironmentBySlugQuery = { __typename?: 'Query', envBySlug?: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID?: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt?: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } | null };


export const GetEnvironmentBySlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironmentBySlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentBySlugQuery, GetEnvironmentBySlugQueryVariables>;