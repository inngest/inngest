/* eslint-disable */
import type { SpanMetadataKind } from '@inngest/components/RunDetailsV3/types';
import type { SpanMetadataScope } from '@inngest/components/RunDetailsV3/types';
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
  Bytes: { input: string; output: string; }
  DSN: { input: unknown; output: unknown; }
  EdgeType: { input: unknown; output: unknown; }
  FilterType: { input: string; output: string; }
  IP: { input: string; output: string; }
  IngestSource: { input: string; output: string; }
  Int64: { input: number; output: number; }
  JSON: { input: null | boolean | number | string | Record<string, unknown> | unknown[]; output: null | boolean | number | string | Record<string, unknown> | unknown[]; }
  Map: { input: Record<string, unknown>; output: Record<string, unknown>; }
  NullString: { input: null | string; output: null | string; }
  NullTime: { input: null | string; output: null | string; }
  Period: { input: unknown; output: unknown; }
  Role: { input: unknown; output: unknown; }
  Runtime: { input: unknown; output: unknown; }
  SchemaSource: { input: unknown; output: unknown; }
  SearchObject: { input: unknown; output: unknown; }
  SpanMetadataKind: { input: SpanMetadataKind; output: SpanMetadataKind; }
  SpanMetadataScope: { input: SpanMetadataScope; output: SpanMetadataScope; }
  SpanMetadataValues: { input: Record<string, any>; output: Record<string, any>; }
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

export type AwsMarketplaceSetupResponse = {
  __typename?: 'AWSMarketplaceSetupResponse';
  message: Scalars['String']['output'];
};

export type Account = {
  __typename?: 'Account';
  addons: Addons;
  appliedAddons: AppliedAddons;
  billingEmail: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  datadogConnections: Array<DatadogConnectionStatus>;
  datadogOrganizations: Array<DatadogOrganization>;
  entitlementUsage: EntitlementUsage;
  entitlements: Entitlements;
  id: Scalars['ID']['output'];
  insightsQueries: Array<InsightsQueryStatement>;
  marketplace: Maybe<Marketplace>;
  name: Maybe<Scalars['NullString']['output']>;
  paymentIntents: Array<PaymentIntent>;
  paymentMethods: Maybe<Array<PaymentMethod>>;
  plan: Maybe<BillingPlan>;
  quickSearch: QuickSearchResults;
  search: SearchResults;
  status: Scalars['String']['output'];
  subscription: Maybe<BillingSubscription>;
  updatedAt: Scalars['Time']['output'];
  users: Array<User>;
  vercelIntegration: Maybe<VercelIntegration>;
};


export type AccountQuickSearchArgs = {
  envSlug: Scalars['String']['input'];
  term: Scalars['String']['input'];
};


export type AccountSearchArgs = {
  opts: SearchInput;
};

export type Addon = {
  __typename?: 'Addon';
  available: Scalars['Boolean']['output'];
  baseValue: Maybe<Scalars['Int']['output']>;
  maxValue: Scalars['Int']['output'];
  name: Scalars['String']['output'];
  price: Maybe<Scalars['Int']['output']>;
  purchaseCount: Scalars['Int']['output'];
  quantityPer: Scalars['Int']['output'];
};

export type AddonMulti = {
  __typename?: 'AddonMulti';
  billingPeriod: Scalars['BillingPeriod']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  price: Price;
  quantityPer: Scalars['Int']['output'];
};

export type Addons = {
  __typename?: 'Addons';
  accountID: Maybe<Scalars['UUID']['output']>;
  advancedObservability: AdvancedObservabilityAddon;
  concurrency: Addon;
  connectWorkers: Addon;
  hipaa: Addon;
  planID: Maybe<Scalars['UUID']['output']>;
  slackChannel: Addon;
  userCount: Addon;
};

export type AdvancedObservabilityAddon = {
  __typename?: 'AdvancedObservabilityAddon';
  available: Scalars['Boolean']['output'];
  entitlements: AdvancedObservabilityEntitlements;
  name: Scalars['String']['output'];
  price: Maybe<Scalars['Int']['output']>;
  purchased: Scalars['Boolean']['output'];
};

export type AdvancedObservabilityEntitlements = {
  __typename?: 'AdvancedObservabilityEntitlements';
  history: EntitlementInt;
  metricsExportFreshness: EntitlementInt;
  metricsExportGranularity: EntitlementInt;
};

export type App = {
  __typename?: 'App';
  appVersion: Maybe<Scalars['String']['output']>;
  archivedAt: Maybe<Scalars['Time']['output']>;
  createdAt: Scalars['Time']['output'];
  externalID: Scalars['String']['output'];
  functionCount: Scalars['Int']['output'];
  functions: Array<Workflow>;
  id: Scalars['UUID']['output'];
  isArchived: Scalars['Boolean']['output'];
  isParentArchived: Scalars['Boolean']['output'];
  latestSync: Maybe<Deploy>;
  method: AppMethod;
  name: Scalars['String']['output'];
  signingKeyRotationCheck: SigningKeyRotationCheck;
  syncs: Array<Deploy>;
};


export type AppLatestSyncArgs = {
  status: InputMaybe<SyncStatus>;
};


export type AppSyncsArgs = {
  after: InputMaybe<Scalars['Time']['input']>;
  first?: Scalars['Int']['input'];
};

export type AppCheckFieldBoolean = {
  __typename?: 'AppCheckFieldBoolean';
  value: Maybe<Scalars['Boolean']['output']>;
};

export type AppCheckFieldString = {
  __typename?: 'AppCheckFieldString';
  value: Maybe<Scalars['String']['output']>;
};

export type AppCheckResult = {
  __typename?: 'AppCheckResult';
  apiOrigin: Maybe<AppCheckFieldString>;
  appID: Maybe<AppCheckFieldString>;
  authenticationSucceeded: Maybe<AppCheckFieldBoolean>;
  env: Maybe<AppCheckFieldString>;
  error: Maybe<Scalars['String']['output']>;
  eventAPIOrigin: Maybe<AppCheckFieldString>;
  eventKeyStatus: SecretCheck;
  extra: Maybe<Scalars['Map']['output']>;
  framework: Maybe<AppCheckFieldString>;
  isReachable: Scalars['Boolean']['output'];
  isSDK: Scalars['Boolean']['output'];
  mode: Maybe<SdkMode>;
  respHeaders: Maybe<Scalars['Map']['output']>;
  respStatusCode: Maybe<Scalars['Int']['output']>;
  sdkLanguage: Maybe<AppCheckFieldString>;
  sdkVersion: Maybe<AppCheckFieldString>;
  serveOrigin: Maybe<AppCheckFieldString>;
  servePath: Maybe<AppCheckFieldString>;
  signingKeyFallbackStatus: SecretCheck;
  signingKeyStatus: SecretCheck;
};

export enum AppMethod {
  Api = 'API',
  Connect = 'CONNECT',
  Serve = 'SERVE'
}

export type AppliedAddonMulti = {
  __typename?: 'AppliedAddonMulti';
  addon: AddonMulti;
  quantity: Scalars['Int']['output'];
};

export type AppliedAddons = {
  __typename?: 'AppliedAddons';
  concurrency: Maybe<AppliedAddonMulti>;
  users: Maybe<AppliedAddonMulti>;
};

export type AppsFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  method?: InputMaybe<AppMethod>;
};

export type ArchiveWorkflowInput = {
  archive: Scalars['Boolean']['input'];
  workflowID: Scalars['ID']['input'];
};

export type ArchivedEvent = {
  __typename?: 'ArchivedEvent';
  event: Scalars['Bytes']['output'];
  eventModel: Event;
  eventVersion: EventType;
  functionRuns: Array<FunctionRun>;
  id: Scalars['ULID']['output'];
  ingestSourceID: Maybe<Scalars['ID']['output']>;
  name: Scalars['String']['output'];
  occurredAt: Scalars['Time']['output'];
  receivedAt: Scalars['Time']['output'];
  skippedFunctionRuns: Array<SkippedFunctionRun>;
  source: Maybe<IngestKey>;
  version: Scalars['String']['output'];
};

export type AvailableAddons = {
  __typename?: 'AvailableAddons';
  concurrency: Maybe<AddonMulti>;
  users: Maybe<AddonMulti>;
};

export type BillingPlan = {
  __typename?: 'BillingPlan';
  addons: Addons;
  amount: Scalars['Int']['output'];
  availableAddons: AvailableAddons;
  billingPeriod: Scalars['BillingPeriod']['output'];
  entitlements: Entitlements;
  features: Scalars['Map']['output'];
  id: Scalars['ID']['output'];
  isFree: Scalars['Boolean']['output'];
  isLegacy: Scalars['Boolean']['output'];
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
};

export type BillingSubscription = {
  __typename?: 'BillingSubscription';
  nextInvoiceAmount: Scalars['Int']['output'];
  nextInvoiceDate: Scalars['Time']['output'];
};

export type CdcConnection = {
  __typename?: 'CDCConnection';
  Host: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  description: Maybe<Scalars['String']['output']>;
  engine: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  status: CdcStatus;
  statusDetail: Maybe<Scalars['Map']['output']>;
  updatedAt: Scalars['Time']['output'];
  watermark: Maybe<Scalars['Map']['output']>;
};

export type CdcConnectionInput = {
  adminConn: Scalars['String']['input'];
  engine: Scalars['String']['input'];
  name: Scalars['String']['input'];
  replicaConn?: InputMaybe<Scalars['String']['input']>;
};

export type CdcSetupResponse = {
  __typename?: 'CDCSetupResponse';
  error: Maybe<Scalars['String']['output']>;
  steps: Maybe<Scalars['Map']['output']>;
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
  createdAt: Scalars['Time']['output'];
  environmentID: Scalars['UUID']['output'];
  expression: Maybe<Scalars['String']['output']>;
  functionID: Scalars['UUID']['output'];
  id: Scalars['ULID']['output'];
  name: Maybe<Scalars['String']['output']>;
  queuedAtMax: Scalars['Time']['output'];
  queuedAtMin: Maybe<Scalars['Time']['output']>;
};

export type CancellationConfiguration = {
  __typename?: 'CancellationConfiguration';
  condition: Maybe<Scalars['String']['output']>;
  event: Scalars['String']['output'];
  timeout: Maybe<Scalars['String']['output']>;
};

export type CancellationConnection = {
  __typename?: 'CancellationConnection';
  edges: Array<CancellationEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

export type CancellationEdge = {
  __typename?: 'CancellationEdge';
  cursor: Scalars['String']['output'];
  node: Cancellation;
};

export type CancellationRunCountInput = {
  queuedAtMax: Scalars['Time']['input'];
  queuedAtMin?: InputMaybe<Scalars['Time']['input']>;
};

export type CodedError = {
  __typename?: 'CodedError';
  code: Scalars['String']['output'];
  data: Maybe<Scalars['JSON']['output']>;
  message: Scalars['String']['output'];
};

export type ConcurrencyConfiguration = {
  __typename?: 'ConcurrencyConfiguration';
  key: Maybe<Scalars['String']['output']>;
  limit: ConcurrencyLimitConfiguration;
  scope: ConcurrencyScope;
};

export type ConcurrencyLimitConfiguration = {
  __typename?: 'ConcurrencyLimitConfiguration';
  isPlanLimit: Maybe<Scalars['Boolean']['output']>;
  value: Scalars['Int']['output'];
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
  appID: Maybe<Scalars['UUID']['output']>;
  appName: Maybe<Scalars['String']['output']>;
  appVersion: Maybe<Scalars['String']['output']>;
  buildId: Maybe<Scalars['String']['output']>;
  connectedAt: Scalars['Time']['output'];
  cpuCores: Scalars['Int']['output'];
  /** @deprecated buildId is deprecated. Use appVersion instead. */
  deploy: Maybe<Deploy>;
  disconnectReason: Maybe<Scalars['String']['output']>;
  disconnectedAt: Maybe<Scalars['Time']['output']>;
  functionCount: Scalars['Int']['output'];
  gatewayId: Scalars['ULID']['output'];
  id: Scalars['ULID']['output'];
  instanceId: Scalars['String']['output'];
  lastHeartbeatAt: Maybe<Scalars['Time']['output']>;
  maxWorkerConcurrency: Scalars['Int64']['output'];
  memBytes: Scalars['Int']['output'];
  os: Scalars['String']['output'];
  sdkLang: Scalars['String']['output'];
  sdkPlatform: Scalars['String']['output'];
  sdkVersion: Scalars['String']['output'];
  status: ConnectV1ConnectionStatus;
  workerIp: Scalars['String']['output'];
};

export type ConnectV1WorkerConnectionEdge = {
  __typename?: 'ConnectV1WorkerConnectionEdge';
  cursor: Scalars['String']['output'];
  node: ConnectV1WorkerConnection;
};

export type ConnectV1WorkerConnectionsConnection = {
  __typename?: 'ConnectV1WorkerConnectionsConnection';
  edges: Array<ConnectV1WorkerConnectionEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

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

export type CreateStripeSubscriptionResponse = {
  __typename?: 'CreateStripeSubscriptionResponse';
  clientSecret: Scalars['String']['output'];
  message: Scalars['String']['output'];
};

export type CreateUserPayload = {
  __typename?: 'CreateUserPayload';
  user: Maybe<User>;
};

export type CreateVercelAppInput = {
  originOverride?: InputMaybe<Scalars['String']['input']>;
  path?: InputMaybe<Scalars['String']['input']>;
  projectID: Scalars['String']['input'];
  protectionBypassSecret?: InputMaybe<Scalars['String']['input']>;
  workspaceID: Scalars['ID']['input'];
};

export type CreateVercelAppResponse = {
  __typename?: 'CreateVercelAppResponse';
  success: Scalars['Boolean']['output'];
};

export type DatadogConnectionStatus = {
  __typename?: 'DatadogConnectionStatus';
  envID: Scalars['UUID']['output'];
  envName: Scalars['String']['output'];
  healthy: Scalars['Boolean']['output'];
  id: Scalars['UUID']['output'];
  lastErrorMessage: Maybe<Scalars['String']['output']>;
  lastSentAt: Maybe<Scalars['Time']['output']>;
  orgID: Scalars['UUID']['output'];
  orgName: Scalars['String']['output'];
};

export type DatadogOrganization = {
  __typename?: 'DatadogOrganization';
  createdAt: Scalars['Time']['output'];
  datadogDomain: Scalars['String']['output'];
  datadogOrgID: Maybe<Scalars['String']['output']>;
  datadogOrgName: Maybe<Scalars['String']['output']>;
  datadogSite: Scalars['String']['output'];
  id: Scalars['UUID']['output'];
  updatedAt: Scalars['Time']['output'];
};

export type DebounceConfiguration = {
  __typename?: 'DebounceConfiguration';
  key: Maybe<Scalars['String']['output']>;
  period: Scalars['String']['output'];
};

export type DeleteIngestKey = {
  id: Scalars['ID']['input'];
  workspaceID: Scalars['ID']['input'];
};

export type DeleteResponse = {
  __typename?: 'DeleteResponse';
  ids: Array<Scalars['ID']['output']>;
};

export type DeleteUlidResponse = {
  __typename?: 'DeleteULIDResponse';
  ids: Array<Scalars['ULID']['output']>;
};

export type Deploy = {
  __typename?: 'Deploy';
  appName: Scalars['String']['output'];
  appVersion: Maybe<Scalars['String']['output']>;
  authorID: Maybe<Scalars['UUID']['output']>;
  checksum: Scalars['String']['output'];
  commitAuthor: Maybe<Scalars['String']['output']>;
  commitHash: Maybe<Scalars['String']['output']>;
  commitMessage: Maybe<Scalars['String']['output']>;
  commitRef: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Time']['output'];
  deployedFunctions: Array<Workflow>;
  dupeCount: Scalars['Int']['output'];
  error: Maybe<Scalars['String']['output']>;
  framework: Maybe<Scalars['String']['output']>;
  functionCount: Maybe<Scalars['Int']['output']>;
  id: Scalars['UUID']['output'];
  idempotencyKey: Maybe<Scalars['String']['output']>;
  lastSyncedAt: Scalars['Time']['output'];
  metadata: Scalars['Map']['output'];
  platform: Maybe<Scalars['String']['output']>;
  prevFunctionCount: Maybe<Scalars['Int']['output']>;
  removedFunctions: Array<Workflow>;
  repoURL: Maybe<Scalars['String']['output']>;
  sdkLanguage: Scalars['String']['output'];
  sdkVersion: Scalars['String']['output'];
  status: Scalars['String']['output'];
  syncKind: Maybe<Scalars['String']['output']>;
  trustProbeStatus: Maybe<Scalars['String']['output']>;
  url: Maybe<Scalars['String']['output']>;
  vercelDeploymentID: Maybe<Scalars['String']['output']>;
  vercelDeploymentURL: Maybe<Scalars['String']['output']>;
  vercelProjectID: Maybe<Scalars['String']['output']>;
  vercelProjectURL: Maybe<Scalars['String']['output']>;
  workspaceID: Scalars['UUID']['output'];
};

export type EntitlementBool = {
  __typename?: 'EntitlementBool';
  enabled: Scalars['Boolean']['output'];
};

export type EntitlementConcurrency = {
  __typename?: 'EntitlementConcurrency';
  limit: Scalars['Int']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementConnectAppsPerConnection = {
  __typename?: 'EntitlementConnectAppsPerConnection';
  limit: Maybe<Scalars['Int']['output']>;
};

export type EntitlementConnectWorkerConnections = {
  __typename?: 'EntitlementConnectWorkerConnections';
  limit: Maybe<Scalars['Int']['output']>;
};

export type EntitlementEvents = {
  __typename?: 'EntitlementEvents';
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementExecutions = {
  __typename?: 'EntitlementExecutions';
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementInt = {
  __typename?: 'EntitlementInt';
  limit: Scalars['Int']['output'];
};

export type EntitlementNullableInt = {
  __typename?: 'EntitlementNullableInt';
  limit: Maybe<Scalars['Int']['output']>;
};

export type EntitlementRunCount = {
  __typename?: 'EntitlementRunCount';
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementStepCount = {
  __typename?: 'EntitlementStepCount';
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementUsage = {
  __typename?: 'EntitlementUsage';
  accountConcurrencyLimitHits: Scalars['Int']['output'];
  runCount: EntitlementUsageRunCount;
  stepCount: EntitlementUsageStepCount;
};

export type EntitlementUsageRunCount = {
  __typename?: 'EntitlementUsageRunCount';
  current: Scalars['Int']['output'];
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementUsageStepCount = {
  __typename?: 'EntitlementUsageStepCount';
  current: Scalars['Int']['output'];
  limit: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementUserCount = {
  __typename?: 'EntitlementUserCount';
  limit: Maybe<Scalars['Int']['output']>;
  usage: Scalars['Int']['output'];
};

export type Entitlements = {
  __typename?: 'Entitlements';
  accountID: Maybe<Scalars['UUID']['output']>;
  concurrency: EntitlementConcurrency;
  connect: EntitlementBool;
  connectAppsPerConnection: EntitlementConnectAppsPerConnection;
  connectWorkerConnections: EntitlementConnectWorkerConnections;
  eventBatchCount: EntitlementInt;
  eventBatchTimeout: EntitlementInt;
  eventSize: EntitlementInt;
  events: EntitlementEvents;
  executions: EntitlementExecutions;
  functionBacklogSize: EntitlementNullableInt;
  hipaa: EntitlementBool;
  history: EntitlementInt;
  metricsExport: EntitlementBool;
  metricsExportFreshness: EntitlementInt;
  metricsExportGranularity: EntitlementInt;
  otelTraces: EntitlementBool;
  planID: Maybe<Scalars['UUID']['output']>;
  realtimeConnections: EntitlementInt;
  realtimeMessages: EntitlementInt;
  runCount: EntitlementRunCount;
  runDuration: EntitlementInt;
  slackChannel: EntitlementBool;
  stepCount: EntitlementStepCount;
  tracingCustomSpans: EntitlementInt;
  userCount: EntitlementUserCount;
};

export type EnvEdge = {
  __typename?: 'EnvEdge';
  cursor: Scalars['String']['output'];
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
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  envTypes?: InputMaybe<Array<EnvironmentType>>;
};

export type Event = {
  __typename?: 'Event';
  description: Maybe<Scalars['String']['output']>;
  firstSeen: Maybe<Scalars['Time']['output']>;
  integrationName: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  recent: Array<ArchivedEvent>;
  schemaSource: Maybe<Scalars['SchemaSource']['output']>;
  usage: Usage;
  versionCount: Scalars['Int']['output'];
  versions: Array<Maybe<EventType>>;
  workflows: Array<Workflow>;
  workspaceID: Maybe<Scalars['UUID']['output']>;
};


export type EventRecentArgs = {
  count: InputMaybe<Scalars['Int']['input']>;
};


export type EventUsageArgs = {
  opts: InputMaybe<UsageInput>;
};


export type EventVersionsArgs = {
  versions: InputMaybe<Array<Scalars['String']['input']>>;
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

export type EventSearchConnection = {
  __typename?: 'EventSearchConnection';
  edges: Maybe<Array<Maybe<EventSearchItemEdge>>>;
  pageInfo: PageInfo;
};

export type EventSearchFilter = {
  lowerTime: Scalars['Time']['input'];
  query: Scalars['String']['input'];
  upperTime: Scalars['Time']['input'];
};

export type EventSearchItem = {
  __typename?: 'EventSearchItem';
  id: Scalars['ULID']['output'];
  name: Scalars['String']['output'];
  receivedAt: Scalars['Time']['output'];
};

export type EventSearchItemEdge = {
  __typename?: 'EventSearchItemEdge';
  cursor: Scalars['String']['output'];
  node: EventSearchItem;
};

export type EventSource = {
  __typename?: 'EventSource';
  id: Scalars['ID']['output'];
  name: Maybe<Scalars['String']['output']>;
  sourceKind: Scalars['String']['output'];
};

export type EventType = {
  __typename?: 'EventType';
  createdAt: Maybe<Scalars['Time']['output']>;
  cueType: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  jsonSchema: Scalars['Map']['output'];
  name: Scalars['String']['output'];
  typescript: Scalars['String']['output'];
  updatedAt: Maybe<Scalars['Time']['output']>;
  version: Scalars['String']['output'];
};

export type EventTypeV2 = {
  __typename?: 'EventTypeV2';
  envID: Scalars['UUID']['output'];
  functions: FunctionsConnection;
  latestSchema: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  usage: Usage;
};


export type EventTypeV2FunctionsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  first?: Scalars['Int']['input'];
};


export type EventTypeV2LatestSchemaArgs = {
  format?: EventSchemaFormat;
};


export type EventTypeV2UsageArgs = {
  opts: InputMaybe<UsageInput>;
};

export type EventTypesConnection = {
  __typename?: 'EventTypesConnection';
  edges: Array<EventTypesEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

export type EventTypesEdge = {
  __typename?: 'EventTypesEdge';
  cursor: Scalars['String']['output'];
  node: EventTypeV2;
};

export type EventTypesFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  nameSearch?: InputMaybe<Scalars['String']['input']>;
};

export type EventV2 = {
  __typename?: 'EventV2';
  envID: Scalars['UUID']['output'];
  id: Scalars['ULID']['output'];
  idempotencyKey: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  occurredAt: Scalars['Time']['output'];
  raw: Scalars['String']['output'];
  receivedAt: Scalars['Time']['output'];
  runs: Array<FunctionRunV2>;
  source: Maybe<EventSource>;
  version: Maybe<Scalars['String']['output']>;
};

export type EventsBatchConfiguration = {
  __typename?: 'EventsBatchConfiguration';
  key: Maybe<Scalars['String']['output']>;
  /** The maximum number of events a batch can have. */
  maxSize: Scalars['Int']['output'];
  /** How long to wait before running the function with the batch. */
  timeout: Scalars['String']['output'];
};

export type EventsConnection = {
  __typename?: 'EventsConnection';
  edges: Array<EventsEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

export type EventsEdge = {
  __typename?: 'EventsEdge';
  cursor: Scalars['String']['output'];
  node: EventV2;
};

export type EventsFilter = {
  eventNames?: InputMaybe<Array<Scalars['String']['input']>>;
  from: Scalars['Time']['input'];
  includeInternalEvents?: Scalars['Boolean']['input'];
  query?: InputMaybe<Scalars['String']['input']>;
  until?: InputMaybe<Scalars['Time']['input']>;
};

export type FilterList = {
  __typename?: 'FilterList';
  events: Maybe<Array<Scalars['String']['output']>>;
  ips: Maybe<Array<Scalars['IP']['output']>>;
  type: Maybe<Scalars['FilterType']['output']>;
};

export type FilterListInput = {
  events?: InputMaybe<Array<Scalars['String']['input']>>;
  ips?: InputMaybe<Array<Scalars['IP']['input']>>;
  type?: InputMaybe<Scalars['FilterType']['input']>;
};

export type Function = {
  __typename?: 'Function';
  id: Scalars['UUID']['output'];
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
};

export type FunctionConfiguration = {
  __typename?: 'FunctionConfiguration';
  cancellations: Array<CancellationConfiguration>;
  concurrency: Array<ConcurrencyConfiguration>;
  debounce: Maybe<DebounceConfiguration>;
  eventsBatch: Maybe<EventsBatchConfiguration>;
  priority: Maybe<Scalars['String']['output']>;
  rateLimit: Maybe<RateLimitConfiguration>;
  retries: RetryConfiguration;
  singleton: Maybe<SingletonConfiguration>;
  throttle: Maybe<ThrottleConfiguration>;
};

export type FunctionReplay = {
  __typename?: 'FunctionReplay';
  createdAt: Scalars['Time']['output'];
  endedAt: Scalars['Time']['output'];
  id: Scalars['UUID']['output'];
  name: Maybe<Scalars['String']['output']>;
  scheduledRunCount: Scalars['Int']['output'];
  totalRunCount: Maybe<Scalars['Int']['output']>;
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  accountID: Scalars['UUID']['output'];
  batchID: Maybe<Scalars['ULID']['output']>;
  canRerun: Maybe<Scalars['Boolean']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  event: Maybe<ArchivedEvent>;
  eventID: Maybe<Scalars['ULID']['output']>;
  events: Maybe<Array<ArchivedEvent>>;
  function: Workflow;
  history: Array<RunHistoryItem>;
  historyItemOutput: Maybe<Scalars['String']['output']>;
  id: Scalars['ULID']['output'];
  output: Maybe<Scalars['Bytes']['output']>;
  startedAt: Scalars['Time']['output'];
  status: FunctionRunStatus;
  workflowID: Scalars['UUID']['output'];
  workflowVersion: Maybe<WorkflowVersion>;
  workflowVersionInt: Scalars['Int']['output'];
  workspaceID: Scalars['UUID']['output'];
};


export type FunctionRunHistoryItemOutputArgs = {
  id: Scalars['ULID']['input'];
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

export type FunctionRunV2 = {
  __typename?: 'FunctionRunV2';
  accountID: Scalars['UUID']['output'];
  app: App;
  appID: Scalars['UUID']['output'];
  batchCreatedAt: Maybe<Scalars['Time']['output']>;
  cronSchedule: Maybe<Scalars['String']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  eventName: Maybe<Scalars['String']['output']>;
  function: Workflow;
  functionID: Scalars['UUID']['output'];
  hasAI: Scalars['Boolean']['output'];
  id: Scalars['ULID']['output'];
  isBatch: Scalars['Boolean']['output'];
  output: Maybe<Scalars['Bytes']['output']>;
  queuedAt: Scalars['Time']['output'];
  sourceID: Maybe<Scalars['String']['output']>;
  startedAt: Maybe<Scalars['Time']['output']>;
  status: FunctionRunStatus;
  trace: Maybe<RunTraceSpan>;
  traceID: Scalars['String']['output'];
  triggerIDs: Array<Scalars['ULID']['output']>;
  workspaceID: Scalars['UUID']['output'];
};


export type FunctionRunV2TraceArgs = {
  preview: InputMaybe<Scalars['Boolean']['input']>;
};

export type FunctionRunV2Edge = {
  __typename?: 'FunctionRunV2Edge';
  cursor: Scalars['String']['output'];
  node: FunctionRunV2;
};

export type FunctionTrigger = {
  __typename?: 'FunctionTrigger';
  condition: Maybe<Scalars['String']['output']>;
  type: FunctionTriggerTypes;
  value: Scalars['String']['output'];
};

export enum FunctionTriggerTypes {
  Cron = 'CRON',
  Event = 'EVENT'
}

export type FunctionsConnection = {
  __typename?: 'FunctionsConnection';
  edges: Array<FunctionsEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

export type FunctionsEdge = {
  __typename?: 'FunctionsEdge';
  cursor: Scalars['String']['output'];
  node: Function;
};

export type FunctionsFilter = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  eventName?: InputMaybe<Scalars['String']['input']>;
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
  createdAt: Scalars['Time']['output'];
  filter: FilterList;
  id: Scalars['ID']['output'];
  metadata: Maybe<Scalars['Map']['output']>;
  name: Scalars['NullString']['output'];
  presharedKey: Scalars['String']['output'];
  source: Scalars['IngestSource']['output'];
  url: Maybe<Scalars['String']['output']>;
};

export type IngestKeyFilter = {
  name?: InputMaybe<Scalars['String']['input']>;
  source?: InputMaybe<Scalars['String']['input']>;
};

export type InsightsColumn = {
  __typename?: 'InsightsColumn';
  columnType: InsightsColumnType;
  name: Scalars['String']['output'];
};

export enum InsightsColumnType {
  Date = 'DATE',
  Number = 'NUMBER',
  String = 'STRING',
  Unknown = 'UNKNOWN'
}

export type InsightsQueryStatement = {
  __typename?: 'InsightsQueryStatement';
  createdAt: Scalars['Time']['output'];
  creator: Scalars['String']['output'];
  id: Scalars['ULID']['output'];
  lastEditor: Scalars['String']['output'];
  name: Scalars['String']['output'];
  shared: Scalars['Boolean']['output'];
  sql: Scalars['String']['output'];
  updatedAt: Scalars['Time']['output'];
};

export type InsightsResponse = {
  __typename?: 'InsightsResponse';
  columns: Array<InsightsColumn>;
  rows: Array<InsightsRow>;
};

export type InsightsRow = {
  __typename?: 'InsightsRow';
  values: Array<Scalars['String']['output']>;
};

export type InvokeStepInfo = {
  __typename?: 'InvokeStepInfo';
  functionID: Scalars['String']['output'];
  returnEventID: Maybe<Scalars['ULID']['output']>;
  runID: Maybe<Scalars['ULID']['output']>;
  timedOut: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
  triggeringEventID: Scalars['ULID']['output'];
};

export enum Marketplace {
  Aws = 'AWS',
  DigitalOcean = 'DIGITAL_OCEAN',
  Partner = 'PARTNER',
  Vercel = 'VERCEL'
}

export type MetricsData = {
  __typename?: 'MetricsData';
  bucket: Scalars['Time']['output'];
  value: Scalars['Float']['output'];
};

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

export type MetricsResponse = {
  __typename?: 'MetricsResponse';
  data: Array<MetricsData>;
  from: Scalars['Time']['output'];
  granularity: Scalars['String']['output'];
  to: Scalars['Time']['output'];
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
  createInsightsQuery: InsightsQueryStatement;
  createSigningKey: SigningKey;
  createStripeSubscription: CreateStripeSubscriptionResponse;
  createUser: Maybe<CreateUserPayload>;
  createVercelApp: Maybe<CreateVercelAppResponse>;
  createWorkspace: Array<Maybe<Workspace>>;
  datadogOAuthCompleted: DatadogOrganization;
  datadogOAuthRedirectURL: Scalars['String']['output'];
  deleteCancellation: Scalars['ULID']['output'];
  deleteIngestKey: Maybe<DeleteResponse>;
  deleteSigningKey: SigningKey;
  disableDatadogConnection: Scalars['UUID']['output'];
  disableEnvironmentAutoArchive: Workspace;
  enableDatadogConnection: DatadogConnectionStatus;
  enableEnvironmentAutoArchive: Workspace;
  invokeFunction: Maybe<Scalars['Boolean']['output']>;
  pauseFunction: Workflow;
  removeDatadogOrganization: Scalars['UUID']['output'];
  removeInsightsQuery: Maybe<DeleteUlidResponse>;
  removeVercelApp: Maybe<RemoveVercelAppResponse>;
  rerun: Scalars['ULID']['output'];
  resyncApp: SyncResponse;
  retryWorkflowRun: Maybe<StartWorkflowResponse>;
  rotateSigningKey: SigningKey;
  setAccountEntitlement: Scalars['UUID']['output'];
  setUpAccount: Maybe<SetUpAccountPayload>;
  shareInsightsQuery: InsightsQueryStatement;
  submitChurnSurvey: Scalars['Boolean']['output'];
  syncNewApp: SyncResponse;
  unarchiveApp: App;
  unarchiveEnvironment: Workspace;
  unpauseFunction: Workflow;
  updateAccount: Account;
  updateAccountAddonQuantity: Addon;
  updateIngestKey: IngestKey;
  updateInsightsQuery: InsightsQueryStatement;
  updatePaymentMethod: Maybe<Array<PaymentMethod>>;
  updatePlan: Account;
  updateVercelApp: Maybe<UpdateVercelAppResponse>;
};


export type MutationArchiveAppArgs = {
  id: Scalars['UUID']['input'];
};


export type MutationArchiveEnvironmentArgs = {
  id: Scalars['ID']['input'];
};


export type MutationArchiveEventArgs = {
  name: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
};


export type MutationArchiveWorkflowArgs = {
  input: ArchiveWorkflowInput;
};


export type MutationCancelRunArgs = {
  envID: Scalars['UUID']['input'];
  runID: Scalars['ULID']['input'];
};


export type MutationCdcAutoSetupArgs = {
  envID: Scalars['UUID']['input'];
  input: CdcConnectionInput;
};


export type MutationCdcDeleteArgs = {
  envID: Scalars['UUID']['input'];
  id: Scalars['UUID']['input'];
};


export type MutationCdcManualSetupArgs = {
  envID: Scalars['UUID']['input'];
  input: CdcConnectionInput;
};


export type MutationCdcTestCredentialsArgs = {
  envID: Scalars['UUID']['input'];
  input: CdcConnectionInput;
};


export type MutationCdcTestLogicalReplicationArgs = {
  envID: Scalars['UUID']['input'];
  input: CdcConnectionInput;
};


export type MutationCdcTestSetupArgs = {
  envID: Scalars['UUID']['input'];
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


export type MutationCreateInsightsQueryArgs = {
  input: NewInsightsQuery;
};


export type MutationCreateSigningKeyArgs = {
  envID: Scalars['UUID']['input'];
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


export type MutationDatadogOAuthCompletedArgs = {
  authCode: Scalars['String']['input'];
  ddDomain: Scalars['String']['input'];
  ddSite: Scalars['String']['input'];
  orgID: Scalars['String']['input'];
  orgName: Scalars['String']['input'];
};


export type MutationDatadogOAuthRedirectUrlArgs = {
  ddDomain: Scalars['String']['input'];
  ddSite: Scalars['String']['input'];
};


export type MutationDeleteCancellationArgs = {
  cancellationID: Scalars['ULID']['input'];
  envID: Scalars['UUID']['input'];
};


export type MutationDeleteIngestKeyArgs = {
  input: DeleteIngestKey;
};


export type MutationDeleteSigningKeyArgs = {
  id: Scalars['UUID']['input'];
};


export type MutationDisableDatadogConnectionArgs = {
  connectionID: Scalars['UUID']['input'];
};


export type MutationDisableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID']['input'];
};


export type MutationEnableDatadogConnectionArgs = {
  envID: Scalars['UUID']['input'];
  organizationID: Scalars['UUID']['input'];
};


export type MutationEnableEnvironmentAutoArchiveArgs = {
  id: Scalars['ID']['input'];
};


export type MutationInvokeFunctionArgs = {
  data: InputMaybe<Scalars['Map']['input']>;
  envID: Scalars['UUID']['input'];
  functionSlug: Scalars['String']['input'];
  user: InputMaybe<Scalars['Map']['input']>;
};


export type MutationPauseFunctionArgs = {
  cancelRunning: InputMaybe<Scalars['Boolean']['input']>;
  fnID: Scalars['ID']['input'];
};


export type MutationRemoveDatadogOrganizationArgs = {
  organizationID: Scalars['UUID']['input'];
};


export type MutationRemoveInsightsQueryArgs = {
  id: Scalars['ULID']['input'];
};


export type MutationRemoveVercelAppArgs = {
  input: RemoveVercelAppInput;
};


export type MutationRerunArgs = {
  fromStep: InputMaybe<RerunFromStepInput>;
  runID: Scalars['ULID']['input'];
};


export type MutationResyncAppArgs = {
  appExternalID: Scalars['String']['input'];
  appURL: InputMaybe<Scalars['String']['input']>;
  envID: Scalars['UUID']['input'];
};


export type MutationRetryWorkflowRunArgs = {
  input: StartWorkflowInput;
  workflowRunID: Scalars['ULID']['input'];
};


export type MutationRotateSigningKeyArgs = {
  envID: Scalars['UUID']['input'];
};


export type MutationSetAccountEntitlementArgs = {
  entitlementName: Scalars['String']['input'];
  overrideStrategy: Scalars['String']['input'];
  value: Scalars['Int']['input'];
};


export type MutationShareInsightsQueryArgs = {
  id: Scalars['ULID']['input'];
};


export type MutationSubmitChurnSurveyArgs = {
  accountID: Scalars['UUID']['input'];
  clerkUserID: Scalars['String']['input'];
  email: Scalars['String']['input'];
  feedback: InputMaybe<Scalars['String']['input']>;
  reason: Scalars['String']['input'];
};


export type MutationSyncNewAppArgs = {
  appURL: Scalars['String']['input'];
  envID: Scalars['UUID']['input'];
};


export type MutationUnarchiveAppArgs = {
  id: Scalars['UUID']['input'];
};


export type MutationUnarchiveEnvironmentArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUnpauseFunctionArgs = {
  fnID: Scalars['ID']['input'];
};


export type MutationUpdateAccountArgs = {
  input: UpdateAccount;
};


export type MutationUpdateAccountAddonQuantityArgs = {
  addonName: Scalars['String']['input'];
  quantity: Scalars['Int']['input'];
};


export type MutationUpdateIngestKeyArgs = {
  id: Scalars['ID']['input'];
  input: UpdateIngestKey;
};


export type MutationUpdateInsightsQueryArgs = {
  id: Scalars['ULID']['input'];
  input: UpdateInsightsQuery;
};


export type MutationUpdatePaymentMethodArgs = {
  token: Scalars['String']['input'];
};


export type MutationUpdatePlanArgs = {
  slug: InputMaybe<Scalars['String']['input']>;
  to: InputMaybe<Scalars['ID']['input']>;
};


export type MutationUpdateVercelAppArgs = {
  input: UpdateVercelAppInput;
};

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

/** The pagination information in a connection. */
export type PageInfo = {
  __typename?: 'PageInfo';
  /** When paginating forward, the cursor to query the next page. */
  endCursor: Maybe<Scalars['String']['output']>;
  /** Indicates if there are any pages subsequent to the current page. */
  hasNextPage: Scalars['Boolean']['output'];
  /** Indicates if there are any pages prior to the current page. */
  hasPreviousPage: Scalars['Boolean']['output'];
  /** When paginating backward, the cursor to query the previous page. */
  startCursor: Maybe<Scalars['String']['output']>;
};

export type PageResults = {
  __typename?: 'PageResults';
  cursor: Maybe<Scalars['String']['output']>;
  page: Scalars['Int']['output'];
  perPage: Scalars['Int']['output'];
  totalItems: Maybe<Scalars['Int']['output']>;
  totalPages: Maybe<Scalars['Int']['output']>;
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
  amountLabel: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  description: Scalars['String']['output'];
  invoiceURL: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
};

export type PaymentMethod = {
  __typename?: 'PaymentMethod';
  brand: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  default: Scalars['Boolean']['output'];
  expMonth: Scalars['String']['output'];
  expYear: Scalars['String']['output'];
  last4: Scalars['String']['output'];
};

export type Price = {
  __typename?: 'Price';
  usCents: Scalars['Int']['output'];
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
  executionTimeSeries: Array<TimeSeries>;
  insights: InsightsResponse;
  insightsQuery: InsightsQueryStatement;
  metrics: MetricsResponse;
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
  id: Scalars['ID']['input'];
};


export type QueryDeploysArgs = {
  workspaceID: InputMaybe<Scalars['ID']['input']>;
};


export type QueryEnvBySlugArgs = {
  slug: Scalars['String']['input'];
};


export type QueryEnvsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: InputMaybe<EnvsFilter>;
  first?: Scalars['Int']['input'];
};


export type QueryEventsArgs = {
  query: InputMaybe<EventQuery>;
};


export type QueryExecutionTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryInsightsArgs = {
  query: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
};


export type QueryInsightsQueryArgs = {
  id: Scalars['ULID']['input'];
};


export type QueryMetricsArgs = {
  opts: MetricsOpts;
};


export type QueryRunCountTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryWorkspaceArgs = {
  id: Scalars['ID']['input'];
};

export type QuickSearchApp = {
  __typename?: 'QuickSearchApp';
  name: Scalars['String']['output'];
};

export type QuickSearchEnv = {
  __typename?: 'QuickSearchEnv';
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
};

export type QuickSearchEvent = {
  __typename?: 'QuickSearchEvent';
  envSlug: Scalars['String']['output'];
  id: Scalars['ULID']['output'];
  name: Scalars['String']['output'];
};

export type QuickSearchEventType = {
  __typename?: 'QuickSearchEventType';
  name: Scalars['String']['output'];
};

export type QuickSearchFunction = {
  __typename?: 'QuickSearchFunction';
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
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
  envSlug: Scalars['String']['output'];
  id: Scalars['ULID']['output'];
};

export type RateLimitConfiguration = {
  __typename?: 'RateLimitConfiguration';
  key: Maybe<Scalars['String']['output']>;
  limit: Scalars['Int']['output'];
  period: Scalars['String']['output'];
};

export type RemoveVercelAppInput = {
  projectID: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
};

export type RemoveVercelAppResponse = {
  __typename?: 'RemoveVercelAppResponse';
  success: Scalars['Boolean']['output'];
};

export type Replay = {
  __typename?: 'Replay';
  createdAt: Scalars['Time']['output'];
  endedAt: Maybe<Scalars['Time']['output']>;
  /** Filters applied to the replay, such as specific run statuses. */
  filters: Maybe<Scalars['JSON']['output']>;
  /** Structured filters applied to the replay. */
  filtersV2: Maybe<ReplayFilters>;
  /**
   * The event or function ID that starts the replay range.
   *
   * This is not inclusive.
   *
   * A DateTime can also be used by generating an ULID from it.
   */
  fromRange: Scalars['ULID']['output'];
  /** The number of functions that were processed during the replay. */
  functionRunsProcessedCount: Scalars['Int']['output'];
  /** The number of function runs created scheduled from the replay. */
  functionRunsScheduledCount: Scalars['Int']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  replayType: ReplayType;
  /**
   * The event or function ID that ends the replay range.
   *
   * This is inclusive.
   *
   * A DateTime can also be used by generating an ULID from it.
   */
  toRange: Scalars['ULID']['output'];
  /** The total number of function runs expected to be created from the replay. */
  totalRunCount: Maybe<Scalars['Int']['output']>;
  workflowID: Maybe<Scalars['UUID']['output']>;
  workspaceID: Maybe<Scalars['UUID']['output']>;
};

export type ReplayFilters = {
  __typename?: 'ReplayFilters';
  skipReasons: Array<SkipReason>;
  statuses: Array<FunctionRunStatus>;
};

export type ReplayRunCounts = {
  __typename?: 'ReplayRunCounts';
  cancelledCount: Scalars['Int']['output'];
  completedCount: Scalars['Int']['output'];
  failedCount: Scalars['Int']['output'];
  skippedPausedCount: Scalars['Int']['output'];
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

export type RetryConfiguration = {
  __typename?: 'RetryConfiguration';
  isDefault: Maybe<Scalars['Boolean']['output']>;
  value: Scalars['Int']['output'];
};

export type RunHistoryCancel = {
  __typename?: 'RunHistoryCancel';
  eventID: Maybe<Scalars['ULID']['output']>;
  expression: Maybe<Scalars['String']['output']>;
  userID: Maybe<Scalars['UUID']['output']>;
};

export type RunHistoryInvokeFunction = {
  __typename?: 'RunHistoryInvokeFunction';
  correlationID: Scalars['String']['output'];
  eventID: Scalars['ULID']['output'];
  functionID: Scalars['String']['output'];
  timeout: Scalars['Time']['output'];
};

export type RunHistoryInvokeFunctionResult = {
  __typename?: 'RunHistoryInvokeFunctionResult';
  eventID: Maybe<Scalars['ULID']['output']>;
  runID: Maybe<Scalars['ULID']['output']>;
  timeout: Scalars['Boolean']['output'];
};

export type RunHistoryItem = {
  __typename?: 'RunHistoryItem';
  attempt: Scalars['Int']['output'];
  cancel: Maybe<RunHistoryCancel>;
  createdAt: Scalars['Time']['output'];
  functionVersion: Scalars['Int']['output'];
  groupID: Maybe<Scalars['UUID']['output']>;
  id: Scalars['ULID']['output'];
  invokeFunction: Maybe<RunHistoryInvokeFunction>;
  invokeFunctionResult: Maybe<RunHistoryInvokeFunctionResult>;
  result: Maybe<RunHistoryResult>;
  sleep: Maybe<RunHistorySleep>;
  stepName: Maybe<Scalars['String']['output']>;
  stepType: Maybe<HistoryStepType>;
  type: HistoryType;
  url: Maybe<Scalars['String']['output']>;
  waitForEvent: Maybe<RunHistoryWaitForEvent>;
  waitResult: Maybe<RunHistoryWaitResult>;
};

export type RunHistoryResult = {
  __typename?: 'RunHistoryResult';
  durationMS: Scalars['Int']['output'];
  errorCode: Maybe<Scalars['String']['output']>;
  framework: Maybe<Scalars['String']['output']>;
  platform: Maybe<Scalars['String']['output']>;
  sdkLanguage: Scalars['String']['output'];
  sdkVersion: Scalars['String']['output'];
  sizeBytes: Scalars['Int']['output'];
};

export type RunHistorySleep = {
  __typename?: 'RunHistorySleep';
  until: Scalars['Time']['output'];
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
  eventName: Scalars['String']['output'];
  expression: Maybe<Scalars['String']['output']>;
  timeout: Scalars['Time']['output'];
};

export type RunHistoryWaitResult = {
  __typename?: 'RunHistoryWaitResult';
  eventID: Maybe<Scalars['ULID']['output']>;
  timeout: Scalars['Boolean']['output'];
};

export type RunListConnection = {
  __typename?: 'RunListConnection';
  edges: Maybe<Array<Maybe<RunListItemEdge>>>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

export type RunListItem = {
  __typename?: 'RunListItem';
  endedAt: Maybe<Scalars['Time']['output']>;
  eventID: Scalars['ULID']['output'];
  id: Scalars['ULID']['output'];
  startedAt: Scalars['Time']['output'];
  status: FunctionRunStatus;
};

export type RunListItemEdge = {
  __typename?: 'RunListItemEdge';
  cursor: Scalars['String']['output'];
  node: RunListItem;
};

export type RunTraceSpan = {
  __typename?: 'RunTraceSpan';
  account: Account;
  accountID: Scalars['UUID']['output'];
  appID: Scalars['UUID']['output'];
  attempts: Maybe<Scalars['Int']['output']>;
  childrenSpans: Array<RunTraceSpan>;
  duration: Maybe<Scalars['Int']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  functionID: Scalars['UUID']['output'];
  isPreview: Maybe<Scalars['Boolean']['output']>;
  isRoot: Scalars['Boolean']['output'];
  isUserland: Scalars['Boolean']['output'];
  metadata: Array<SpanMetadata>;
  name: Scalars['String']['output'];
  outputID: Maybe<Scalars['String']['output']>;
  parentSpan: Maybe<RunTraceSpan>;
  parentSpanID: Maybe<Scalars['String']['output']>;
  queuedAt: Scalars['Time']['output'];
  run: FunctionRun;
  runID: Scalars['ULID']['output'];
  spanID: Scalars['String']['output'];
  startedAt: Maybe<Scalars['Time']['output']>;
  status: RunTraceSpanStatus;
  stepID: Maybe<Scalars['String']['output']>;
  stepInfo: Maybe<StepInfo>;
  stepOp: Maybe<StepOp>;
  stepType: Scalars['String']['output'];
  traceID: Scalars['String']['output'];
  userlandSpan: Maybe<UserlandSpan>;
  workspace: Workspace;
  workspaceID: Scalars['UUID']['output'];
};

export type RunTraceSpanOutput = {
  __typename?: 'RunTraceSpanOutput';
  data: Maybe<Scalars['Bytes']['output']>;
  error: Maybe<StepError>;
  input: Maybe<Scalars['Bytes']['output']>;
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
  IDs: Array<Scalars['ULID']['output']>;
  batchID: Maybe<Scalars['ULID']['output']>;
  cron: Maybe<Scalars['String']['output']>;
  eventName: Maybe<Scalars['String']['output']>;
  isBatch: Scalars['Boolean']['output'];
  payloads: Array<Scalars['Bytes']['output']>;
  timestamp: Scalars['Time']['output'];
};

export type RunsConnection = {
  __typename?: 'RunsConnection';
  edges: Array<FunctionRunV2Edge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};

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

export type ScopedFunctionStatusResponse = {
  __typename?: 'ScopedFunctionStatusResponse';
  cancelled: Scalars['Int']['output'];
  completed: Scalars['Int']['output'];
  failed: Scalars['Int']['output'];
  from: Scalars['Time']['output'];
  queued: Scalars['Int']['output'];
  running: Scalars['Int']['output'];
  skipped: Scalars['Int']['output'];
  to: Scalars['Time']['output'];
};

export type ScopedMetric = {
  __typename?: 'ScopedMetric';
  data: Array<MetricsData>;
  id: Scalars['UUID']['output'];
  tagName: Maybe<Scalars['String']['output']>;
  tagValue: Maybe<Scalars['String']['output']>;
};

export type ScopedMetricsFilter = {
  appIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  from: Scalars['Time']['input'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  groupBy?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
  scope: MetricsScope;
  until?: InputMaybe<Scalars['Time']['input']>;
};

export type ScopedMetricsResponse = {
  __typename?: 'ScopedMetricsResponse';
  from: Scalars['Time']['output'];
  granularity: Scalars['String']['output'];
  metrics: Array<ScopedMetric>;
  scope: MetricsScope;
  to: Scalars['Time']['output'];
};

export type SearchInput = {
  term: Scalars['String']['input'];
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
  count: Scalars['Int']['output'];
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
  expires: Maybe<Scalars['Time']['output']>;
  user: User;
};

export type SetUpAccountPayload = {
  __typename?: 'SetUpAccountPayload';
  account: Maybe<Account>;
};

export type SigningKey = {
  __typename?: 'SigningKey';
  createdAt: Scalars['Time']['output'];
  decryptedValue: Scalars['String']['output'];
  id: Scalars['UUID']['output'];
  isActive: Scalars['Boolean']['output'];
  user: Maybe<User>;
};

export type SigningKeyRotationCheck = {
  __typename?: 'SigningKeyRotationCheck';
  sdkSupport: Scalars['Boolean']['output'];
  signingKeyFallbackState: SecretCheck;
  signingKeyState: SecretCheck;
};

export type SingletonConfiguration = {
  __typename?: 'SingletonConfiguration';
  key: Maybe<Scalars['String']['output']>;
  mode: SingletonMode;
};

export enum SingletonMode {
  Cancel = 'CANCEL',
  Skip = 'SKIP'
}

export enum SkipReason {
  FunctionPaused = 'FUNCTION_PAUSED',
  None = 'NONE'
}

export type SkippedFunctionRun = {
  __typename?: 'SkippedFunctionRun';
  accountID: Scalars['UUID']['output'];
  batchID: Maybe<Scalars['ULID']['output']>;
  eventID: Maybe<Scalars['ULID']['output']>;
  id: Scalars['ULID']['output'];
  skipReason: SkipReason;
  skippedAt: Scalars['Time']['output'];
  workflowID: Scalars['UUID']['output'];
  workspaceID: Scalars['UUID']['output'];
};

export type SleepStepInfo = {
  __typename?: 'SleepStepInfo';
  sleepUntil: Scalars['Time']['output'];
};

export type SpanMetadata = {
  __typename?: 'SpanMetadata';
  kind: Scalars['SpanMetadataKind']['output'];
  scope: Scalars['SpanMetadataScope']['output'];
  updatedAt: Scalars['Time']['output'];
  values: Scalars['SpanMetadataValues']['output'];
};

export type StartWorkflowInput = {
  workflowID: Scalars['ID']['input'];
  workflowVersion?: InputMaybe<Scalars['Int']['input']>;
  workspaceID: Scalars['ID']['input'];
};

export type StartWorkflowResponse = {
  __typename?: 'StartWorkflowResponse';
  id: Scalars['ULID']['output'];
};

export type StepError = {
  __typename?: 'StepError';
  cause: Maybe<Scalars['String']['output']>;
  message: Scalars['String']['output'];
  name: Maybe<Scalars['String']['output']>;
  stack: Maybe<Scalars['String']['output']>;
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
  amount: Scalars['Int']['input'];
  planID?: InputMaybe<Scalars['ID']['input']>;
  planSlug?: InputMaybe<Scalars['String']['input']>;
  quantity: Scalars['Int']['input'];
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
  burst: Scalars['Int']['output'];
  key: Maybe<Scalars['String']['output']>;
  limit: Scalars['Int']['output'];
  period: Scalars['String']['output'];
};

export type TimeSeries = {
  __typename?: 'TimeSeries';
  data: Array<TimeSeriesPoint>;
  name: Scalars['String']['output'];
};

export type TimeSeriesOptions = {
  interval?: InputMaybe<Scalars['String']['input']>;
  month: Scalars['Int']['input'];
  year: Scalars['Int']['input'];
};

export type TimeSeriesPoint = {
  __typename?: 'TimeSeriesPoint';
  time: Scalars['Time']['output'];
  value: Maybe<Scalars['Float']['output']>;
};

export type UpdateAccount = {
  billingEmail?: InputMaybe<Scalars['String']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
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

export type UpdateVercelAppResponse = {
  __typename?: 'UpdateVercelAppResponse';
  success: Scalars['Boolean']['output'];
  vercelApp: Maybe<VercelApp>;
};

export type Usage = {
  __typename?: 'Usage';
  asOf: Scalars['Time']['output'];
  data: Array<UsageSlot>;
  period: Scalars['Period']['output'];
  range: Scalars['Timerange']['output'];
  total: Scalars['Int']['output'];
};

export type UsageInput = {
  from?: InputMaybe<Scalars['Time']['input']>;
  period?: InputMaybe<Scalars['Period']['input']>;
  range?: InputMaybe<Scalars['Timerange']['input']>;
  to?: InputMaybe<Scalars['Time']['input']>;
};

export type UsageSlot = {
  __typename?: 'UsageSlot';
  count: Scalars['Int']['output'];
  slot: Scalars['Time']['output'];
};

export type User = {
  __typename?: 'User';
  account: Maybe<Account>;
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  lastLoginAt: Maybe<Scalars['Time']['output']>;
  name: Maybe<Scalars['NullString']['output']>;
  passwordChangedAt: Maybe<Scalars['Time']['output']>;
  roles: Maybe<Array<Maybe<Scalars['Role']['output']>>>;
  updatedAt: Scalars['Time']['output'];
};

export type UserlandSpan = {
  __typename?: 'UserlandSpan';
  resourceAttrs: Maybe<Scalars['Bytes']['output']>;
  scopeName: Maybe<Scalars['String']['output']>;
  scopeVersion: Maybe<Scalars['String']['output']>;
  serviceName: Maybe<Scalars['String']['output']>;
  spanAttrs: Maybe<Scalars['Bytes']['output']>;
  spanKind: Maybe<Scalars['String']['output']>;
  spanName: Maybe<Scalars['String']['output']>;
};

export type VercelApp = {
  __typename?: 'VercelApp';
  id: Scalars['UUID']['output'];
  originOverride: Maybe<Scalars['String']['output']>;
  path: Maybe<Scalars['String']['output']>;
  projectID: Scalars['String']['output'];
  protectionBypassSecret: Maybe<Scalars['String']['output']>;
  workspaceID: Scalars['UUID']['output'];
};

export enum VercelDeploymentProtection {
  All = 'ALL',
  AllExceptCustomDomains = 'ALL_EXCEPT_CUSTOM_DOMAINS',
  Disabled = 'DISABLED',
  Preview = 'PREVIEW',
  ProdDeploymentUrlsAndAllPreviews = 'PROD_DEPLOYMENT_URLS_AND_ALL_PREVIEWS',
  Unknown = 'UNKNOWN'
}

export type VercelIntegration = {
  __typename?: 'VercelIntegration';
  isMarketplace: Scalars['Boolean']['output'];
  projects: Array<VercelProject>;
};

export type VercelProject = {
  __typename?: 'VercelProject';
  canChangeEnabled: Scalars['Boolean']['output'];
  deploymentProtection: VercelDeploymentProtection;
  isEnabled: Scalars['Boolean']['output'];
  name: Scalars['String']['output'];
  originOverride: Maybe<Scalars['String']['output']>;
  projectID: Scalars['String']['output'];
  protectionBypassSecret: Maybe<Scalars['String']['output']>;
  servePath: Scalars['String']['output'];
};

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String']['output'];
  expression: Maybe<Scalars['String']['output']>;
  foundEventID: Maybe<Scalars['ULID']['output']>;
  timedOut: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
};

export type Workflow = {
  __typename?: 'Workflow';
  app: App;
  archivedAt: Maybe<Scalars['Time']['output']>;
  cancellationRunCount: Scalars['Int']['output'];
  cancellations: CancellationConnection;
  configuration: Maybe<FunctionConfiguration>;
  current: Maybe<WorkflowVersion>;
  failureHandler: Maybe<Workflow>;
  id: Scalars['ID']['output'];
  isArchived: Scalars['Boolean']['output'];
  isParentArchived: Scalars['Boolean']['output'];
  isPaused: Scalars['Boolean']['output'];
  latestVersion: Maybe<WorkflowVersion>;
  metrics: MetricsResponse;
  name: Scalars['String']['output'];
  previous: Array<Maybe<WorkflowVersion>>;
  replayCounts: ReplayRunCounts;
  /**
   * A list of all the function's replays.
   *
   * This doesn't include environment-level replays.
   */
  replays: Array<Replay>;
  run: FunctionRun;
  runs: Maybe<RunListConnection>;
  runsV2: Maybe<RunListConnection>;
  slug: Scalars['String']['output'];
  triggers: Array<FunctionTrigger>;
  url: Scalars['String']['output'];
  usage: Usage;
};


export type WorkflowCancellationRunCountArgs = {
  input: CancellationRunCountInput;
};


export type WorkflowCancellationsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  first?: Scalars['Int']['input'];
};


export type WorkflowMetricsArgs = {
  opts: MetricsRequest;
};


export type WorkflowReplayCountsArgs = {
  from: Scalars['Time']['input'];
  to: Scalars['Time']['input'];
};


export type WorkflowRunArgs = {
  id: Scalars['ULID']['input'];
};


export type WorkflowRunsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: RunsFilter;
  first?: Scalars['Int']['input'];
};


export type WorkflowRunsV2Args = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: RunsFilter;
  first?: Scalars['Int']['input'];
};


export type WorkflowUsageArgs = {
  event: InputMaybe<Scalars['String']['input']>;
  opts: InputMaybe<UsageInput>;
};

export type WorkflowResponse = {
  __typename?: 'WorkflowResponse';
  workflow: Workflow;
};

export type WorkflowVersion = {
  __typename?: 'WorkflowVersion';
  createdAt: Scalars['Time']['output'];
  deploy: Maybe<Deploy>;
  description: Maybe<Scalars['NullString']['output']>;
  retries: Scalars['Int']['output'];
  throttleCount: Scalars['Int']['output'];
  throttlePeriod: Scalars['String']['output'];
  triggers: Array<FunctionTrigger>;
  updatedAt: Scalars['Time']['output'];
  url: Scalars['String']['output'];
  validFrom: Maybe<Scalars['Time']['output']>;
  validTo: Maybe<Scalars['Time']['output']>;
  version: Scalars['Int']['output'];
  workflowID: Scalars['ID']['output'];
  workflowType: Scalars['String']['output'];
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
  connectWorkerMetrics: ScopedMetricsResponse;
  createdAt: Scalars['Time']['output'];
  event: Maybe<Event>;
  eventByNames: Array<EventType>;
  eventSearch: EventSearchConnection;
  eventType: EventTypeV2;
  eventTypes: PaginatedEventTypes;
  eventTypesV2: EventTypesConnection;
  eventV2: EventV2;
  events: PaginatedEvents;
  eventsV2: EventsConnection;
  functionCount: Scalars['Int']['output'];
  id: Scalars['ID']['output'];
  ingestKey: IngestKey;
  ingestKeys: Array<IngestKey>;
  isArchived: Scalars['Boolean']['output'];
  isAutoArchiveEnabled: Scalars['Boolean']['output'];
  lastDeployedAt: Maybe<Scalars['Time']['output']>;
  name: Scalars['String']['output'];
  parentID: Maybe<Scalars['ID']['output']>;
  replay: Replay;
  run: Maybe<FunctionRunV2>;
  runTraceSpanOutputByID: RunTraceSpanOutput;
  runTrigger: RunTraceTrigger;
  runs: RunsConnection;
  scopedFunctionStatus: ScopedFunctionStatusResponse;
  scopedMetrics: ScopedMetricsResponse;
  signingKeys: Array<SigningKey>;
  slug: Scalars['String']['output'];
  test: Scalars['Boolean']['output'];
  traceOutput: Scalars['Bytes']['output'];
  type: EnvironmentType;
  unattachedSyncs: Array<Deploy>;
  vercelApps: Array<VercelApp>;
  webhookSigningKey: Scalars['String']['output'];
  workerConnection: Maybe<ConnectV1WorkerConnection>;
  workerConnections: ConnectV1WorkerConnectionsConnection;
  workflow: Maybe<Workflow>;
  workflowBySlug: Maybe<Workflow>;
  workflows: PaginatedWorkflows;
};


export type WorkspaceAppByExternalIdArgs = {
  externalID: Scalars['String']['input'];
};


export type WorkspaceAppCheckArgs = {
  url: Scalars['String']['input'];
};


export type WorkspaceAppsArgs = {
  filter: InputMaybe<AppsFilter>;
};


export type WorkspaceArchivedEventArgs = {
  id: Scalars['ULID']['input'];
};


export type WorkspaceConnectWorkerMetricsArgs = {
  filter: ConnectV1WorkerMetricsFilter;
};


export type WorkspaceEventArgs = {
  name: Scalars['String']['input'];
};


export type WorkspaceEventByNamesArgs = {
  names: Array<Scalars['String']['input']>;
};


export type WorkspaceEventSearchArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: EventSearchFilter;
  first?: Scalars['Int']['input'];
};


export type WorkspaceEventTypeArgs = {
  name: Scalars['String']['input'];
};


export type WorkspaceEventTypesV2Args = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: EventTypesFilter;
  first?: Scalars['Int']['input'];
};


export type WorkspaceEventV2Args = {
  id: Scalars['ULID']['input'];
};


export type WorkspaceEventsArgs = {
  prefix: InputMaybe<Scalars['String']['input']>;
};


export type WorkspaceEventsV2Args = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: EventsFilter;
  first?: Scalars['Int']['input'];
};


export type WorkspaceIngestKeyArgs = {
  id: Scalars['ID']['input'];
};


export type WorkspaceIngestKeysArgs = {
  filter: InputMaybe<IngestKeyFilter>;
};


export type WorkspaceReplayArgs = {
  id: Scalars['ID']['input'];
};


export type WorkspaceRunArgs = {
  runID: Scalars['String']['input'];
};


export type WorkspaceRunTraceSpanOutputByIdArgs = {
  outputID: Scalars['String']['input'];
};


export type WorkspaceRunTriggerArgs = {
  runID: Scalars['String']['input'];
};


export type WorkspaceRunsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: RunsFilterV2;
  first?: Scalars['Int']['input'];
  orderBy: Array<RunsOrderBy>;
  preview: InputMaybe<Scalars['Boolean']['input']>;
};


export type WorkspaceScopedFunctionStatusArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceScopedMetricsArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceTraceOutputArgs = {
  outputID: Scalars['String']['input'];
};


export type WorkspaceUnattachedSyncsArgs = {
  after: InputMaybe<Scalars['Time']['input']>;
  first?: Scalars['Int']['input'];
};


export type WorkspaceWorkerConnectionArgs = {
  connectionId: Scalars['ULID']['input'];
};


export type WorkspaceWorkerConnectionsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: ConnectV1WorkerConnectionsFilter;
  first?: Scalars['Int']['input'];
  orderBy: Array<ConnectV1WorkerConnectionsOrderBy>;
};


export type WorkspaceWorkflowArgs = {
  id: Scalars['ID']['input'];
};


export type WorkspaceWorkflowBySlugArgs = {
  slug: Scalars['String']['input'];
};


export type WorkspaceWorkflowsArgs = {
  archived?: InputMaybe<Scalars['Boolean']['input']>;
  search: InputMaybe<Scalars['String']['input']>;
};

export type AchiveAppMutationVariables = Exact<{
  appID: Scalars['UUID']['input'];
}>;


export type AchiveAppMutation = { __typename?: 'Mutation', archiveApp: { __typename?: 'App', id: string } };

export type UnachiveAppMutationVariables = Exact<{
  appID: Scalars['UUID']['input'];
}>;


export type UnachiveAppMutation = { __typename?: 'Mutation', unarchiveApp: { __typename?: 'App', id: string } };

export type GetArchivedAppBannerDataQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  externalAppID: Scalars['String']['input'];
}>;


export type GetArchivedAppBannerDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', isArchived: boolean } } };

export type ResyncAppMutationVariables = Exact<{
  appExternalID: Scalars['String']['input'];
  appURL: InputMaybe<Scalars['String']['input']>;
  envID: Scalars['UUID']['input'];
}>;


export type ResyncAppMutation = { __typename?: 'Mutation', resyncApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type SyncNewAppMutationVariables = Exact<{
  appURL: Scalars['String']['input'];
  envID: Scalars['UUID']['input'];
}>;


export type SyncNewAppMutation = { __typename?: 'Mutation', syncNewApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', externalID: string, id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type SyncQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  externalAppID: Scalars['String']['input'];
  syncID: Scalars['ID']['input'];
}>;


export type SyncQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, name: string, method: AppMethod } }, sync: { __typename?: 'Deploy', appVersion: string | null, commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> } };

export type CheckAppQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  url: Scalars['String']['input'];
}>;


export type CheckAppQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', appCheck: { __typename?: 'AppCheckResult', error: string | null, eventKeyStatus: SecretCheck, extra: Record<string, unknown> | null, isReachable: boolean, isSDK: boolean, mode: SdkMode | null, respHeaders: Record<string, unknown> | null, respStatusCode: number | null, signingKeyStatus: SecretCheck, signingKeyFallbackStatus: SecretCheck, apiOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, appID: { __typename?: 'AppCheckFieldString', value: string | null } | null, authenticationSucceeded: { __typename?: 'AppCheckFieldBoolean', value: boolean | null } | null, env: { __typename?: 'AppCheckFieldString', value: string | null } | null, eventAPIOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, framework: { __typename?: 'AppCheckFieldString', value: string | null } | null, sdkLanguage: { __typename?: 'AppCheckFieldString', value: string | null } | null, sdkVersion: { __typename?: 'AppCheckFieldString', value: string | null } | null, serveOrigin: { __typename?: 'AppCheckFieldString', value: string | null } | null, servePath: { __typename?: 'AppCheckFieldString', value: string | null } | null } } };

export type AppQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  externalAppID: Scalars['String']['input'];
}>;


export type AppQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, externalID: string, appVersion: string | null, name: string, method: AppMethod, functions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> }>, latestSync: { __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, appVersion: string | null } | null } } };

export type AppsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  filter: AppsFilter;
}>;


export type AppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', id: string, externalID: string, functionCount: number, isArchived: boolean, name: string, method: AppMethod, isParentArchived: boolean, latestSync: { __typename?: 'Deploy', error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null } | null, functions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> }> }> } };

export type AppNavDataQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  externalAppID: Scalars['String']['input'];
}>;


export type AppNavDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, isArchived: boolean, isParentArchived: boolean, method: AppMethod, name: string, latestSync: { __typename?: 'Deploy', platform: string | null, url: string | null } | null } } };

export type AppSyncsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  externalAppID: Scalars['String']['input'];
}>;


export type AppSyncsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', app: { __typename?: 'App', id: string, syncs: Array<{ __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> }> } } };

export type LatestUnattachedSyncQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type LatestUnattachedSyncQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string }> } };

export type UpdateAccountAddonQuantityMutationVariables = Exact<{
  addonName: Scalars['String']['input'];
  quantity: Scalars['Int']['input'];
}>;


export type UpdateAccountAddonQuantityMutation = { __typename?: 'Mutation', updateAccountAddonQuantity: { __typename?: 'Addon', purchaseCount: number } };

export type UpdateAccountMutationVariables = Exact<{
  input: UpdateAccount;
}>;


export type UpdateAccountMutation = { __typename?: 'Mutation', account: { __typename?: 'Account', billingEmail: string, name: null | string | null } };

export type UpdatePaymentMethodMutationVariables = Exact<{
  token: Scalars['String']['input'];
}>;


export type UpdatePaymentMethodMutation = { __typename?: 'Mutation', updatePaymentMethod: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null };

export type GetPaymentIntentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPaymentIntentsQuery = { __typename?: 'Query', account: { __typename?: 'Account', paymentIntents: Array<{ __typename?: 'PaymentIntent', status: string, createdAt: string, amountLabel: string, description: string, invoiceURL: string | null }> } };

export type CreateStripeSubscriptionMutationVariables = Exact<{
  input: StripeSubscriptionInput;
}>;


export type CreateStripeSubscriptionMutation = { __typename?: 'Mutation', createStripeSubscription: { __typename?: 'CreateStripeSubscriptionResponse', clientSecret: string, message: string } };

export type UpdatePlanMutationVariables = Exact<{
  planSlug: Scalars['String']['input'];
}>;


export type UpdatePlanMutation = { __typename?: 'Mutation', updatePlan: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, name: string } | null } };

export type SubmitChurnSurveyMutationVariables = Exact<{
  reason: Scalars['String']['input'];
  feedback: InputMaybe<Scalars['String']['input']>;
  email: Scalars['String']['input'];
  accountID: Scalars['UUID']['input'];
  clerkUserID: Scalars['String']['input'];
}>;


export type SubmitChurnSurveyMutation = { __typename?: 'Mutation', submitChurnSurvey: boolean };

export type GetBillableStepsQueryVariables = Exact<{
  month: Scalars['Int']['input'];
  year: Scalars['Int']['input'];
}>;


export type GetBillableStepsQuery = { __typename?: 'Query', usage: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type GetBillableRunsQueryVariables = Exact<{
  month: Scalars['Int']['input'];
  year: Scalars['Int']['input'];
}>;


export type GetBillableRunsQuery = { __typename?: 'Query', usage: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type GetBillableExecutionsQueryVariables = Exact<{
  month: Scalars['Int']['input'];
  year: Scalars['Int']['input'];
}>;


export type GetBillableExecutionsQuery = { __typename?: 'Query', usage: Array<{ __typename?: 'TimeSeries', data: Array<{ __typename?: 'TimeSeriesPoint', time: string, value: number | null }> }> };

export type GetBillingInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingInfoQuery = { __typename?: 'Query', account: { __typename?: 'Account', entitlements: { __typename?: 'Entitlements', executions: { __typename?: 'EntitlementExecutions', limit: number | null }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null }, runCount: { __typename?: 'EntitlementRunCount', limit: number | null } }, plan: { __typename?: 'BillingPlan', slug: string } | null } };

export type CreateEnvironmentMutationVariables = Exact<{
  name: Scalars['String']['input'];
}>;


export type CreateEnvironmentMutation = { __typename?: 'Mutation', createWorkspace: Array<{ __typename?: 'Workspace', id: string } | null> };

export type EnableDatadogConnectionMutationVariables = Exact<{
  organizationID: Scalars['UUID']['input'];
  envID: Scalars['UUID']['input'];
}>;


export type EnableDatadogConnectionMutation = { __typename?: 'Mutation', enableDatadogConnection: { __typename?: 'DatadogConnectionStatus', id: string } };

export type FinishDatadogIntegrationDocumentMutationVariables = Exact<{
  orgName: Scalars['String']['input'];
  orgID: Scalars['String']['input'];
  authCode: Scalars['String']['input'];
  ddSite: Scalars['String']['input'];
  ddDomain: Scalars['String']['input'];
}>;


export type FinishDatadogIntegrationDocumentMutation = { __typename?: 'Mutation', datadogOAuthCompleted: { __typename?: 'DatadogOrganization', id: string } };

export type GetDatadogSetupDataQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDatadogSetupDataQuery = { __typename?: 'Query', account: { __typename?: 'Account', datadogConnections: Array<{ __typename?: 'DatadogConnectionStatus', id: string, orgID: string, orgName: string, envID: string, envName: string, healthy: boolean, lastErrorMessage: string | null, lastSentAt: string | null }>, datadogOrganizations: Array<{ __typename?: 'DatadogOrganization', id: string, datadogDomain: string, datadogOrgName: string | null }> } };

export type DisableDatadogConnectionMutationVariables = Exact<{
  connectionID: Scalars['UUID']['input'];
}>;


export type DisableDatadogConnectionMutation = { __typename?: 'Mutation', disableDatadogConnection: string };

export type RemoveDatadogOrganizationMutationVariables = Exact<{
  organizationID: Scalars['UUID']['input'];
}>;


export type RemoveDatadogOrganizationMutation = { __typename?: 'Mutation', removeDatadogOrganization: string };

export type StartDatadogIntegrationMutationVariables = Exact<{
  ddSite: Scalars['String']['input'];
  ddDomain: Scalars['String']['input'];
}>;


export type StartDatadogIntegrationMutation = { __typename?: 'Mutation', datadogOAuthRedirectURL: string };

export type DisableEnvironmentAutoArchiveDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DisableEnvironmentAutoArchiveDocumentMutation = { __typename?: 'Mutation', disableEnvironmentAutoArchive: { __typename?: 'Workspace', id: string } };

export type EnableEnvironmentAutoArchiveMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type EnableEnvironmentAutoArchiveMutation = { __typename?: 'Mutation', enableEnvironmentAutoArchive: { __typename?: 'Workspace', id: string } };

export type ArchiveEnvironmentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type ArchiveEnvironmentMutation = { __typename?: 'Mutation', archiveEnvironment: { __typename?: 'Workspace', id: string } };

export type UnarchiveEnvironmentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type UnarchiveEnvironmentMutation = { __typename?: 'Mutation', unarchiveEnvironment: { __typename?: 'Workspace', id: string } };

export type GetEventTypesV2QueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  cursor: InputMaybe<Scalars['String']['input']>;
  archived: InputMaybe<Scalars['Boolean']['input']>;
  nameSearch: InputMaybe<Scalars['String']['input']>;
}>;


export type GetEventTypesV2Query = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventTypesV2: { __typename?: 'EventTypesConnection', edges: Array<{ __typename?: 'EventTypesEdge', node: { __typename?: 'EventTypeV2', name: string, functions: { __typename?: 'FunctionsConnection', edges: Array<{ __typename?: 'FunctionsEdge', node: { __typename?: 'Function', id: string, slug: string, name: string } }> } } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor: string | null, hasPreviousPage: boolean, startCursor: string | null } } } };

export type GetEventTypeVolumeV2QueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  eventName: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetEventTypeVolumeV2Query = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventType: { __typename?: 'EventTypeV2', name: string, usage: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number, slot: string }> } } } };

export type GetEventTypeQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  eventName: Scalars['String']['input'];
}>;


export type GetEventTypeQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventType: { __typename?: 'EventTypeV2', name: string, functions: { __typename?: 'FunctionsConnection', edges: Array<{ __typename?: 'FunctionsEdge', node: { __typename?: 'Function', id: string, slug: string, name: string } }> } } } };

export type GetAllEventNamesQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type GetAllEventNamesQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventTypesV2: { __typename?: 'EventTypesConnection', edges: Array<{ __typename?: 'EventTypesEdge', node: { __typename?: 'EventTypeV2', name: string } }> } } };

export type ArchiveEventMutationVariables = Exact<{
  environmentId: Scalars['ID']['input'];
  name: Scalars['String']['input'];
}>;


export type ArchiveEventMutation = { __typename?: 'Mutation', archiveEvent: { __typename?: 'Event', name: string } | null };

export type GetLatestEventLogsQueryVariables = Exact<{
  name: InputMaybe<Scalars['String']['input']>;
  environmentID: Scalars['ID']['input'];
}>;


export type GetLatestEventLogsQuery = { __typename?: 'Query', events: { __typename?: 'PaginatedEvents', data: Array<{ __typename?: 'Event', recent: Array<{ __typename?: 'ArchivedEvent', id: string, receivedAt: string, event: string, source: { __typename?: 'IngestKey', name: null | string } | null }> }> } | null };

export type GetEventKeysQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
}>;


export type GetEventKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventKeys: Array<{ __typename?: 'IngestKey', name: null | string, value: string }> } };

export type GetEventsV2QueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  cursor: InputMaybe<Scalars['String']['input']>;
  startTime: Scalars['Time']['input'];
  endTime: InputMaybe<Scalars['Time']['input']>;
  celQuery?: InputMaybe<Scalars['String']['input']>;
  eventNames?: InputMaybe<Array<Scalars['String']['input']> | Scalars['String']['input']>;
  includeInternalEvents?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type GetEventsV2Query = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventsV2: { __typename?: 'EventsConnection', totalCount: number, edges: Array<{ __typename?: 'EventsEdge', node: { __typename?: 'EventV2', name: string, id: string, receivedAt: string, runs: Array<{ __typename?: 'FunctionRunV2', status: FunctionRunStatus, id: string, startedAt: string | null, endedAt: string | null, function: { __typename?: 'Workflow', name: string, slug: string } }> } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor: string | null, hasPreviousPage: boolean, startCursor: string | null } } } };

export type GetEventV2QueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  eventID: Scalars['ULID']['input'];
}>;


export type GetEventV2Query = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventV2: { __typename?: 'EventV2', name: string, id: string, receivedAt: string, idempotencyKey: string | null, occurredAt: string, version: string | null, source: { __typename?: 'EventSource', name: string | null } | null } } };

export type GetEventPayloadQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  eventID: Scalars['ULID']['input'];
}>;


export type GetEventPayloadQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventV2: { __typename?: 'EventV2', raw: string } } };

export type GetEventV2RunsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  eventID: Scalars['ULID']['input'];
}>;


export type GetEventV2RunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventV2: { __typename?: 'EventV2', name: string, runs: Array<{ __typename?: 'FunctionRunV2', status: FunctionRunStatus, id: string, startedAt: string | null, endedAt: string | null, function: { __typename?: 'Workflow', name: string, slug: string } }> } } };

export type GetArchivedFuncBannerDataQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  funcID: Scalars['ID']['input'];
}>;


export type GetArchivedFuncBannerDataQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, archivedAt: string | null } | null } };

export type CreateCancellationMutationVariables = Exact<{
  input: CreateCancellationInput;
}>;


export type CreateCancellationMutation = { __typename?: 'Mutation', createCancellation: { __typename?: 'Cancellation', id: string } };

export type GetCancellationRunCountQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
  queuedAtMin: InputMaybe<Scalars['Time']['input']>;
  queuedAtMax: Scalars['Time']['input'];
}>;


export type GetCancellationRunCountQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', cancellationRunCount: number } | null } };

export type DeleteCancellationMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
  cancellationID: Scalars['ULID']['input'];
}>;


export type DeleteCancellationMutation = { __typename?: 'Mutation', deleteCancellation: string };

export type GetFunctionRateLimitDocumentQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  fnSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetFunctionRateLimitDocumentQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', ratelimit: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFunctionRunsMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetFunctionRunsMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', completed: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, canceled: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, failed: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type GetFnMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  fnSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetFnMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFailedFunctionRunsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
  from: Scalars['Time']['input'];
  until: Scalars['Time']['input'];
}>;


export type GetFailedFunctionRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', failedRuns: { __typename?: 'RunsConnection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', id: string, endedAt: string | null } }> } } };

export type PauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID']['input'];
  cancelRunning: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type PauseFunctionMutation = { __typename?: 'Mutation', pauseFunction: { __typename?: 'Workflow', id: string } };

export type UnpauseFunctionMutationVariables = Exact<{
  fnID: Scalars['ID']['input'];
}>;


export type UnpauseFunctionMutation = { __typename?: 'Mutation', unpauseFunction: { __typename?: 'Workflow', id: string } };

export type GetSdkRequestMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  fnSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetSdkRequestMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', queued: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, started: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, ended: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetStepBacklogMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  fnSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetStepBacklogMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', scheduled: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, sleeping: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetStepsRunningMetricsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  fnSlug: Scalars['String']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetStepsRunningMetricsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', running: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, concurrencyLimit: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> } } | null } };

export type GetFnCancellationsQueryVariables = Exact<{
  after: InputMaybe<Scalars['String']['input']>;
  envSlug: Scalars['String']['input'];
  fnSlug: Scalars['String']['input'];
}>;


export type GetFnCancellationsQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', fn: { __typename?: 'Workflow', cancellations: { __typename?: 'CancellationConnection', edges: Array<{ __typename?: 'CancellationEdge', cursor: string, node: { __typename?: 'Cancellation', createdAt: string, id: string, name: string | null, queuedAtMax: string, queuedAtMin: string | null, envID: string } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean } } } | null } | null };

export type InsightsResultsQueryVariables = Exact<{
  query: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
}>;


export type InsightsResultsQuery = { __typename?: 'Query', insights: { __typename?: 'InsightsResponse', columns: Array<{ __typename?: 'InsightsColumn', name: string, columnType: InsightsColumnType }>, rows: Array<{ __typename?: 'InsightsRow', values: Array<string> }> } };

export type GetEventTypeSchemasQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  cursor: InputMaybe<Scalars['String']['input']>;
  nameSearch: InputMaybe<Scalars['String']['input']>;
  archived: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type GetEventTypeSchemasQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', eventTypesV2: { __typename?: 'EventTypesConnection', edges: Array<{ __typename?: 'EventTypesEdge', node: { __typename?: 'EventTypeV2', name: string, latestSchema: string | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor: string | null, hasPreviousPage: boolean, startCursor: string | null } } } };

export type InsightsSavedQueriesQueryVariables = Exact<{ [key: string]: never; }>;


export type InsightsSavedQueriesQuery = { __typename?: 'Query', account: { __typename?: 'Account', insightsQueries: Array<{ __typename?: 'InsightsQueryStatement', id: string, creator: string, lastEditor: string, name: string, shared: boolean, sql: string, createdAt: string, updatedAt: string }> } };

export type CreateInsightsQueryMutationVariables = Exact<{
  input: NewInsightsQuery;
}>;


export type CreateInsightsQueryMutation = { __typename?: 'Mutation', createInsightsQuery: { __typename?: 'InsightsQueryStatement', id: string, createdAt: string, creator: string, lastEditor: string, name: string, shared: boolean, sql: string, updatedAt: string } };

export type RemoveInsightsQueryMutationVariables = Exact<{
  id: Scalars['ULID']['input'];
}>;


export type RemoveInsightsQueryMutation = { __typename?: 'Mutation', removeInsightsQuery: { __typename?: 'DeleteULIDResponse', ids: Array<string> } | null };

export type ShareInsightsQueryMutationVariables = Exact<{
  id: Scalars['ULID']['input'];
}>;


export type ShareInsightsQueryMutation = { __typename?: 'Mutation', shareInsightsQuery: { __typename?: 'InsightsQueryStatement', id: string, createdAt: string, creator: string, lastEditor: string, name: string, shared: boolean, sql: string, updatedAt: string } };

export type UpdateInsightsQueryMutationVariables = Exact<{
  id: Scalars['ULID']['input'];
  input: UpdateInsightsQuery;
}>;


export type UpdateInsightsQueryMutation = { __typename?: 'Mutation', updateInsightsQuery: { __typename?: 'InsightsQueryStatement', id: string, createdAt: string, creator: string, lastEditor: string, name: string, shared: boolean, sql: string, updatedAt: string } };

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

export type UpdateIngestKeyMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateIngestKey;
}>;


export type UpdateIngestKeyMutation = { __typename?: 'Mutation', updateIngestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url: string | null, metadata: Record<string, unknown> | null, filter: { __typename?: 'FilterList', type: string | null, ips: Array<string> | null, events: Array<string> | null } } };

export type NewIngestKeyMutationVariables = Exact<{
  input: NewIngestKey;
}>;


export type NewIngestKeyMutation = { __typename?: 'Mutation', key: { __typename?: 'IngestKey', id: string } };

export type DeleteEventKeyMutationVariables = Exact<{
  input: DeleteIngestKey;
}>;


export type DeleteEventKeyMutation = { __typename?: 'Mutation', deleteIngestKey: { __typename?: 'DeleteResponse', ids: Array<string> } | null };

export type GetIngestKeysQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
}>;


export type GetIngestKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, source: string }> } };

export type MetricsLookupsQueryVariables = Exact<{
  envSlug: Scalars['String']['input'];
  page: InputMaybe<Scalars['Int']['input']>;
  pageSize: InputMaybe<Scalars['Int']['input']>;
}>;


export type MetricsLookupsQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', externalID: string, id: string, name: string, isArchived: boolean }>, workflows: { __typename?: 'PaginatedWorkflows', data: Array<{ __typename?: 'Workflow', name: string, id: string, slug: string }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null, perPage: number } } } | null };

export type AccountConcurrencyLookupQueryVariables = Exact<{ [key: string]: never; }>;


export type AccountConcurrencyLookupQuery = { __typename?: 'Query', account: { __typename?: 'Account', marketplace: Marketplace | null, entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number } } } };

export type FunctionStatusMetricsQueryVariables = Exact<{
  workspaceId: Scalars['ID']['input'];
  from: Scalars['Time']['input'];
  functionIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  appIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  until: InputMaybe<Scalars['Time']['input']>;
  scope: MetricsScope;
}>;


export type FunctionStatusMetricsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', scheduled: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, started: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, completed: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, completedByFunction: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, totals: { __typename?: 'ScopedFunctionStatusResponse', queued: number, running: number, completed: number, failed: number, cancelled: number, skipped: number } } };

export type VolumeMetricsQueryVariables = Exact<{
  workspaceId: Scalars['ID']['input'];
  from: Scalars['Time']['input'];
  functionIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  appIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  until: InputMaybe<Scalars['Time']['input']>;
  scope: MetricsScope;
}>;


export type VolumeMetricsQuery = { __typename?: 'Query', accountConcurrency: { __typename?: 'MetricsResponse', from: string, to: string, granularity: string, data: Array<{ __typename?: 'MetricsData', bucket: string, value: number }> }, workspace: { __typename?: 'Workspace', runsThroughput: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputEnded: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputStarted: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, sdkThroughputScheduled: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, stepThroughput: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, backlog: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, stepRunning: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, concurrency: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, workerPercentageUsed: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> }, workerTotalCapacity: { __typename?: 'ScopedMetricsResponse', metrics: Array<{ __typename?: 'ScopedMetric', id: string, tagName: string | null, tagValue: string | null, data: Array<{ __typename?: 'MetricsData', value: number, bucket: string }> }> } } };

export type QuickSearchQueryVariables = Exact<{
  term: Scalars['String']['input'];
  envSlug: Scalars['String']['input'];
}>;


export type QuickSearchQuery = { __typename?: 'Query', account: { __typename?: 'Account', quickSearch: { __typename?: 'QuickSearchResults', apps: Array<{ __typename?: 'QuickSearchApp', name: string }>, event: { __typename?: 'QuickSearchEvent', envSlug: string, id: string, name: string } | null, eventTypes: Array<{ __typename?: 'QuickSearchEventType', name: string }>, functions: Array<{ __typename?: 'QuickSearchFunction', name: string, slug: string }>, run: { __typename?: 'QuickSearchRun', envSlug: string, id: string } | null } } };

export type SyncOnboardingAppMutationVariables = Exact<{
  appURL: Scalars['String']['input'];
  envID: Scalars['UUID']['input'];
}>;


export type SyncOnboardingAppMutation = { __typename?: 'Mutation', syncNewApp: { __typename?: 'SyncResponse', app: { __typename?: 'App', externalID: string, id: string } | null, error: { __typename?: 'CodedError', code: string, data: null | boolean | number | string | Record<string, unknown> | unknown[] | null, message: string } | null } };

export type InvokeFunctionOnboardingMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
  data: InputMaybe<Scalars['Map']['input']>;
  functionSlug: Scalars['String']['input'];
  user: InputMaybe<Scalars['Map']['input']>;
}>;


export type InvokeFunctionOnboardingMutation = { __typename?: 'Mutation', invokeFunction: boolean | null };

export type InvokeFunctionLookupQueryVariables = Exact<{
  envSlug: Scalars['String']['input'];
  page: InputMaybe<Scalars['Int']['input']>;
  pageSize: InputMaybe<Scalars['Int']['input']>;
}>;


export type InvokeFunctionLookupQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', data: Array<{ __typename?: 'Workflow', name: string, id: string, slug: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> }>, page: { __typename?: 'PageResults', page: number, totalPages: number | null, perPage: number } } } | null };

export type GetVercelAppsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type GetVercelAppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string, error: string | null, url: string | null, vercelDeploymentURL: string | null }>, apps: Array<{ __typename?: 'App', id: string, name: string, externalID: string, isArchived: boolean, latestSync: { __typename?: 'Deploy', error: string | null, id: string, platform: string | null, vercelDeploymentID: string | null, vercelProjectID: string | null, status: string } | null }> } };

export type ProductionAppsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type ProductionAppsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', id: string }>, unattachedSyncs: Array<{ __typename?: 'Deploy', lastSyncedAt: string }> } };

export type GetAccountEntitlementsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAccountEntitlementsQuery = { __typename?: 'Query', account: { __typename?: 'Account', entitlements: { __typename?: 'Entitlements', history: { __typename?: 'EntitlementInt', limit: number } } } };

export type GetReplayRunCountsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
  from: Scalars['Time']['input'];
  to: Scalars['Time']['input'];
}>;


export type GetReplayRunCountsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, replayCounts: { __typename?: 'ReplayRunCounts', completedCount: number, failedCount: number, cancelledCount: number, skippedPausedCount: number } } | null } };

export type CreateFunctionReplayMutationVariables = Exact<{
  environmentID: Scalars['UUID']['input'];
  functionID: Scalars['UUID']['input'];
  name: Scalars['String']['input'];
  fromRange: Scalars['ULID']['input'];
  toRange: Scalars['ULID']['input'];
  statuses: InputMaybe<Array<ReplayRunStatus> | ReplayRunStatus>;
}>;


export type CreateFunctionReplayMutation = { __typename?: 'Mutation', createFunctionReplay: { __typename?: 'Replay', id: string } };

export type GetReplayQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  replayID: Scalars['ID']['input'];
}>;


export type GetReplayQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', replay: { __typename?: 'Replay', id: string, name: string, createdAt: string, endedAt: string | null, functionRunsScheduledCount: number, fromRange: string, toRange: string, functionRunsProcessedCount: number, filtersV2: { __typename?: 'ReplayFilters', statuses: Array<FunctionRunStatus> } | null } } };

export type GetReplaysQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
}>;


export type GetReplaysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', id: string, function: { __typename?: 'Workflow', id: string, replays: Array<{ __typename?: 'Replay', id: string, name: string, createdAt: string, endedAt: string | null, functionRunsScheduledCount: number, functionRunsProcessedCount: number }> } | null } };

export type GetRunTraceTriggerQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  runID: Scalars['String']['input'];
}>;


export type GetRunTraceTriggerQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runTrigger: { __typename?: 'RunTraceTrigger', IDs: Array<string>, payloads: Array<string>, timestamp: string, eventName: string | null, isBatch: boolean, batchID: string | null, cron: string | null } } };

export type GetRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  environmentID: Scalars['ID']['input'];
  startTime: Scalars['Time']['input'];
  endTime: InputMaybe<Scalars['Time']['input']>;
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsOrderByField;
  functionSlug: InputMaybe<Scalars['String']['input']>;
  functionRunCursor?: InputMaybe<Scalars['String']['input']>;
  celQuery?: InputMaybe<Scalars['String']['input']>;
  preview?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type GetRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', runs: { __typename?: 'RunsConnection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', cronSchedule: string | null, eventName: string | null, id: string, isBatch: boolean, queuedAt: string, endedAt: string | null, startedAt: string | null, status: FunctionRunStatus, hasAI: boolean, app: { __typename?: 'App', externalID: string, name: string }, function: { __typename?: 'Workflow', name: string, slug: string } } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

export type CountRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']['input']> | Scalars['UUID']['input']>;
  environmentID: Scalars['ID']['input'];
  startTime: Scalars['Time']['input'];
  endTime: InputMaybe<Scalars['Time']['input']>;
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsOrderByField;
  functionSlug: InputMaybe<Scalars['String']['input']>;
  celQuery?: InputMaybe<Scalars['String']['input']>;
}>;


export type CountRunsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', runs: { __typename?: 'RunsConnection', totalCount: number } } };

export type AppFilterQueryVariables = Exact<{
  envSlug: Scalars['String']['input'];
}>;


export type AppFilterQuery = { __typename?: 'Query', env: { __typename?: 'Workspace', apps: Array<{ __typename?: 'App', externalID: string, id: string, name: string }> } | null };

export type SeatOverageCheckQueryVariables = Exact<{ [key: string]: never; }>;


export type SeatOverageCheckQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, entitlements: { __typename?: 'Entitlements', userCount: { __typename?: 'EntitlementUserCount', usage: number, limit: number | null } } } };

export type CreateSigningKeyMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
}>;


export type CreateSigningKeyMutation = { __typename?: 'Mutation', createSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type DeleteSigningKeyMutationVariables = Exact<{
  signingKeyID: Scalars['UUID']['input'];
}>;


export type DeleteSigningKeyMutation = { __typename?: 'Mutation', deleteSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type RotateSigningKeyMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
}>;


export type RotateSigningKeyMutation = { __typename?: 'Mutation', rotateSigningKey: { __typename?: 'SigningKey', createdAt: string } };

export type GetSigningKeysQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type GetSigningKeysQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', signingKeys: Array<{ __typename?: 'SigningKey', createdAt: string, decryptedValue: string, id: string, isActive: boolean, user: { __typename?: 'User', email: string, name: null | string | null } | null }> } };

export type UnattachedSyncQueryVariables = Exact<{
  syncID: Scalars['ID']['input'];
}>;


export type UnattachedSyncQuery = { __typename?: 'Query', sync: { __typename?: 'Deploy', appVersion: string | null, commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, error: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }>, syncedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string, slug: string }> } };

export type UnattachedSyncsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type UnattachedSyncsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', syncs: Array<{ __typename?: 'Deploy', commitAuthor: string | null, commitHash: string | null, commitMessage: string | null, commitRef: string | null, framework: string | null, id: string, lastSyncedAt: string, platform: string | null, repoURL: string | null, sdkLanguage: string, sdkVersion: string, status: string, url: string | null, vercelDeploymentID: string | null, vercelDeploymentURL: string | null, vercelProjectID: string | null, vercelProjectURL: string | null }> } };

export type GetWorkerConnectionsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  appID: Scalars['UUID']['input'];
  startTime: InputMaybe<Scalars['Time']['input']>;
  status: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
  timeField: ConnectV1WorkerConnectionsOrderByField;
  cursor?: InputMaybe<Scalars['String']['input']>;
  orderBy?: InputMaybe<Array<ConnectV1WorkerConnectionsOrderBy> | ConnectV1WorkerConnectionsOrderBy>;
  first: Scalars['Int']['input'];
}>;


export type GetWorkerConnectionsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number, edges: Array<{ __typename?: 'ConnectV1WorkerConnectionEdge', node: { __typename?: 'ConnectV1WorkerConnection', id: string, gatewayId: string, workerIp: string, maxWorkerConcurrency: number, connectedAt: string, lastHeartbeatAt: string | null, disconnectedAt: string | null, disconnectReason: string | null, status: ConnectV1ConnectionStatus, sdkLang: string, sdkVersion: string, sdkPlatform: string, appVersion: string | null, functionCount: number, cpuCores: number, memBytes: number, os: string, instanceID: string, app: { __typename?: 'App', id: string } | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } } };

export type GetWorkerCountConnectionsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  appID: Scalars['UUID']['input'];
  startTime: InputMaybe<Scalars['Time']['input']>;
  status?: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
  timeField: ConnectV1WorkerConnectionsOrderByField;
}>;


export type GetWorkerCountConnectionsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number } } };

export type GetDeployssQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
}>;


export type GetDeployssQuery = { __typename?: 'Query', deploys: Array<{ __typename?: 'Deploy', id: string, appName: string, authorID: string | null, checksum: string, createdAt: string, error: string | null, framework: string | null, metadata: Record<string, unknown>, sdkLanguage: string, sdkVersion: string, status: string, deployedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }>, removedFunctions: Array<{ __typename?: 'Workflow', id: string, name: string }> }> | null };

export type GetEnvironmentsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetEnvironmentsQuery = { __typename?: 'Query', workspaces: Array<{ __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, webhookSigningKey: string, createdAt: string, isArchived: boolean, isAutoArchiveEnabled: boolean, lastDeployedAt: string | null }> | null };

export type GetEnvironmentBySlugQueryVariables = Exact<{
  slug: Scalars['String']['input'];
}>;


export type GetEnvironmentBySlugQuery = { __typename?: 'Query', envBySlug: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } | null };

export type GetDefaultEnvironmentQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDefaultEnvironmentQuery = { __typename?: 'Query', defaultEnv: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean } };

export type GetFunctionsUsageQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  page: InputMaybe<Scalars['Int']['input']>;
  archived: InputMaybe<Scalars['Boolean']['input']>;
  pageSize: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetFunctionsUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, dailyStarts: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyCompleted: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyCancelled: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> }, dailyFailures: { __typename?: 'Usage', total: number, data: Array<{ __typename?: 'UsageSlot', count: number }> } }> } } };

export type GetFunctionsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  page: InputMaybe<Scalars['Int']['input']>;
  archived: InputMaybe<Scalars['Boolean']['input']>;
  search: InputMaybe<Scalars['String']['input']>;
  pageSize: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetFunctionsQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflows: { __typename?: 'PaginatedWorkflows', page: { __typename?: 'PageResults', page: number, perPage: number, totalItems: number | null, totalPages: number | null }, data: Array<{ __typename?: 'Workflow', id: string, slug: string, name: string, isPaused: boolean, isArchived: boolean, app: { __typename?: 'App', name: string, externalID: string }, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> }> } } };

export type GetFunctionQueryVariables = Exact<{
  slug: Scalars['String']['input'];
  environmentID: Scalars['ID']['input'];
}>;


export type GetFunctionQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', id: string, workflow: { __typename?: 'Workflow', id: string, name: string, slug: string, isPaused: boolean, isArchived: boolean, app: { __typename?: 'App', externalID: string, name: string, latestSync: { __typename?: 'Deploy', lastSyncedAt: string } | null }, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string, condition: string | null }>, failureHandler: { __typename?: 'Workflow', slug: string, name: string } | null, configuration: { __typename?: 'FunctionConfiguration', priority: string | null, cancellations: Array<{ __typename?: 'CancellationConfiguration', event: string, timeout: string | null, condition: string | null }>, retries: { __typename?: 'RetryConfiguration', value: number, isDefault: boolean | null }, eventsBatch: { __typename?: 'EventsBatchConfiguration', maxSize: number, timeout: string, key: string | null } | null, concurrency: Array<{ __typename?: 'ConcurrencyConfiguration', scope: ConcurrencyScope, key: string | null, limit: { __typename?: 'ConcurrencyLimitConfiguration', value: number, isPlanLimit: boolean | null } }>, rateLimit: { __typename?: 'RateLimitConfiguration', limit: number, period: string, key: string | null } | null, debounce: { __typename?: 'DebounceConfiguration', period: string, key: string | null } | null, throttle: { __typename?: 'ThrottleConfiguration', burst: number, key: string | null, limit: number, period: string } | null, singleton: { __typename?: 'SingletonConfiguration', key: string | null, mode: SingletonMode } | null } | null } | null } };

export type GetFunctionUsageQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  environmentID: Scalars['ID']['input'];
  startTime: Scalars['Time']['input'];
  endTime: Scalars['Time']['input'];
}>;


export type GetFunctionUsageQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', workflow: { __typename?: 'Workflow', dailyStarts: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyCancelled: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyCompleted: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> }, dailyFailures: { __typename?: 'Usage', period: unknown, total: number, data: Array<{ __typename?: 'UsageSlot', slot: string, count: number }> } } | null } };

export type EntitlementUsageQueryVariables = Exact<{ [key: string]: never; }>;


export type EntitlementUsageQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, addons: { __typename?: 'Addons', concurrency: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, userCount: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, advancedObservability: { __typename?: 'AdvancedObservabilityAddon', available: boolean, name: string, price: number | null, purchased: boolean, entitlements: { __typename?: 'AdvancedObservabilityEntitlements', history: { __typename?: 'EntitlementInt', limit: number }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number } } }, slackChannel: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, connectWorkers: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number } }, entitlements: { __typename?: 'Entitlements', runCount: { __typename?: 'EntitlementRunCount', limit: number | null, overageAllowed: boolean }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null, overageAllowed: boolean }, concurrency: { __typename?: 'EntitlementConcurrency', limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, userCount: { __typename?: 'EntitlementUserCount', usage: number, limit: number | null }, hipaa: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExport: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number }, slackChannel: { __typename?: 'EntitlementBool', enabled: boolean }, connectWorkerConnections: { __typename?: 'EntitlementConnectWorkerConnections', limit: number | null } }, plan: { __typename?: 'BillingPlan', name: string } | null } };

export type EntitlementUsageWithMetricsQueryVariables = Exact<{ [key: string]: never; }>;


export type EntitlementUsageWithMetricsQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, addons: { __typename?: 'Addons', concurrency: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, userCount: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, advancedObservability: { __typename?: 'AdvancedObservabilityAddon', available: boolean, name: string, price: number | null, purchased: boolean, entitlements: { __typename?: 'AdvancedObservabilityEntitlements', history: { __typename?: 'EntitlementInt', limit: number }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number } } }, slackChannel: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number }, connectWorkers: { __typename?: 'Addon', available: boolean, baseValue: number | null, maxValue: number, name: string, price: number | null, purchaseCount: number, quantityPer: number } }, entitlements: { __typename?: 'Entitlements', executions: { __typename?: 'EntitlementExecutions', usage: number, limit: number | null, overageAllowed: boolean }, runCount: { __typename?: 'EntitlementRunCount', usage: number, limit: number | null, overageAllowed: boolean }, stepCount: { __typename?: 'EntitlementStepCount', usage: number, limit: number | null, overageAllowed: boolean }, concurrency: { __typename?: 'EntitlementConcurrency', usage: number, limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, userCount: { __typename?: 'EntitlementUserCount', usage: number, limit: number | null }, hipaa: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExport: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number }, slackChannel: { __typename?: 'EntitlementBool', enabled: boolean }, connectWorkerConnections: { __typename?: 'EntitlementConnectWorkerConnections', limit: number | null } }, plan: { __typename?: 'BillingPlan', name: string } | null } };

export type GetCurrentPlanQueryVariables = Exact<{ [key: string]: never; }>;


export type GetCurrentPlanQuery = { __typename?: 'Query', account: { __typename?: 'Account', plan: { __typename?: 'BillingPlan', id: string, slug: string, isLegacy: boolean, isFree: boolean, name: string, amount: number, billingPeriod: unknown, entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, runCount: { __typename?: 'EntitlementRunCount', limit: number | null }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null }, userCount: { __typename?: 'EntitlementUserCount', limit: number | null } }, addons: { __typename?: 'Addons', concurrency: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number }, userCount: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number }, slackChannel: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number }, connectWorkers: { __typename?: 'Addon', available: boolean, price: number | null, purchaseCount: number, quantityPer: number } } } | null, subscription: { __typename?: 'BillingSubscription', nextInvoiceDate: string } | null } };

export type GetBillingDetailsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetBillingDetailsQuery = { __typename?: 'Query', account: { __typename?: 'Account', billingEmail: string, name: null | string | null, paymentMethods: Array<{ __typename?: 'PaymentMethod', brand: string, last4: string, expMonth: string, expYear: string, createdAt: string, default: boolean }> | null } };

export type GetPlansQueryVariables = Exact<{ [key: string]: never; }>;


export type GetPlansQuery = { __typename?: 'Query', plans: Array<{ __typename?: 'BillingPlan', id: string, isLegacy: boolean, isFree: boolean, name: string, amount: number, billingPeriod: unknown, entitlements: { __typename?: 'Entitlements', concurrency: { __typename?: 'EntitlementConcurrency', limit: number }, eventSize: { __typename?: 'EntitlementInt', limit: number }, history: { __typename?: 'EntitlementInt', limit: number }, runCount: { __typename?: 'EntitlementRunCount', limit: number | null }, stepCount: { __typename?: 'EntitlementStepCount', limit: number | null } } } | null> };

export type MetricsEntitlementsQueryVariables = Exact<{ [key: string]: never; }>;


export type MetricsEntitlementsQuery = { __typename?: 'Query', account: { __typename?: 'Account', id: string, entitlements: { __typename?: 'Entitlements', metricsExport: { __typename?: 'EntitlementBool', enabled: boolean }, metricsExportFreshness: { __typename?: 'EntitlementInt', limit: number }, metricsExportGranularity: { __typename?: 'EntitlementInt', limit: number } } } };

export type GetProductionWorkspaceQueryVariables = Exact<{ [key: string]: never; }>;


export type GetProductionWorkspaceQuery = { __typename?: 'Query', defaultEnv: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } };

export type GetPostgresIntegrationsQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
}>;


export type GetPostgresIntegrationsQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', cdcConnections: Array<{ __typename?: 'CDCConnection', id: string, name: string, status: CdcStatus, statusDetail: Record<string, unknown> | null, description: string | null }> } };

export type TestCredentialsMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID']['input'];
}>;


export type TestCredentialsMutation = { __typename?: 'Mutation', cdcTestCredentials: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type TestReplicationMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID']['input'];
}>;


export type TestReplicationMutation = { __typename?: 'Mutation', cdcTestLogicalReplication: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type TestAutoSetupMutationVariables = Exact<{
  input: CdcConnectionInput;
  envID: Scalars['UUID']['input'];
}>;


export type TestAutoSetupMutation = { __typename?: 'Mutation', cdcAutoSetup: { __typename?: 'CDCSetupResponse', steps: Record<string, unknown> | null, error: string | null } };

export type CdcDeleteMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
  id: Scalars['UUID']['input'];
}>;


export type CdcDeleteMutation = { __typename?: 'Mutation', cdcDelete: { __typename?: 'DeleteResponse', ids: Array<string> } };

export type GetSavedVercelProjectsQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
}>;


export type GetSavedVercelProjectsQuery = { __typename?: 'Query', account: { __typename?: 'Account', marketplace: Marketplace | null }, environment: { __typename?: 'Workspace', savedVercelProjects: Array<{ __typename?: 'VercelApp', id: string, originOverride: string | null, projectID: string, protectionBypassSecret: string | null, path: string | null, workspaceID: string }> } };

export type VercelIntegrationQueryVariables = Exact<{ [key: string]: never; }>;


export type VercelIntegrationQuery = { __typename?: 'Query', account: { __typename?: 'Account', vercelIntegration: { __typename?: 'VercelIntegration', isMarketplace: boolean, projects: Array<{ __typename?: 'VercelProject', canChangeEnabled: boolean, deploymentProtection: VercelDeploymentProtection, isEnabled: boolean, name: string, originOverride: string | null, projectID: string, protectionBypassSecret: string | null, servePath: string }> } | null } };

export type ProfileQueryVariables = Exact<{ [key: string]: never; }>;


export type ProfileQuery = { __typename?: 'Query', account: { __typename?: 'Account', name: null | string | null, marketplace: Marketplace | null } };

export type CancelRunMutationVariables = Exact<{
  envID: Scalars['UUID']['input'];
  runID: Scalars['ULID']['input'];
}>;


export type CancelRunMutation = { __typename?: 'Mutation', cancelRun: { __typename?: 'FunctionRun', id: string } };

export type GetEventKeysForBlankSlateQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
}>;


export type GetEventKeysForBlankSlateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKeys: Array<{ __typename?: 'IngestKey', name: null | string, presharedKey: string, createdAt: string }> } };

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, stepID: string | null, spanID: string, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename?: 'SpanMetadata', kind: SpanMetadataKind, scope: SpanMetadataScope, values: Record<string, any>, updatedAt: string }>, stepInfo:
    | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
    | { __typename: 'SleepStepInfo', sleepUntil: string }
    | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
   | null } & { ' $fragmentName'?: 'TraceDetailsFragment' };

export type GetRunTraceQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  runID: Scalars['String']['input'];
  preview: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type GetRunTraceQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', run: { __typename?: 'FunctionRunV2', status: FunctionRunStatus, hasAI: boolean, function: { __typename?: 'Workflow', id: string, name: string, slug: string, app: { __typename?: 'App', name: string, externalID: string } }, trace: (
        { __typename?: 'RunTraceSpan', childrenSpans: Array<(
          { __typename?: 'RunTraceSpan', childrenSpans: Array<(
            { __typename?: 'RunTraceSpan', childrenSpans: Array<(
              { __typename?: 'RunTraceSpan', childrenSpans: Array<(
                { __typename?: 'RunTraceSpan' }
                & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
              )> }
              & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
            )> }
            & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
          )> }
          & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
        )> }
        & { ' $fragmentRefs'?: { 'TraceDetailsFragment': TraceDetailsFragment } }
      ) | null } | null } };

export type TraceResultQueryVariables = Exact<{
  envID: Scalars['ID']['input'];
  traceID: Scalars['String']['input'];
}>;


export type TraceResultQuery = { __typename?: 'Query', workspace: { __typename?: 'Workspace', runTraceSpanOutputByID: { __typename?: 'RunTraceSpanOutput', data: string | null, input: string | null, error: { __typename?: 'StepError', message: string, name: string | null, stack: string | null, cause: string | null } | null } } };

export type RerunFunctionRunMutationVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionID: Scalars['ID']['input'];
  functionRunID: Scalars['ULID']['input'];
}>;


export type RerunFunctionRunMutation = { __typename?: 'Mutation', retryWorkflowRun: { __typename?: 'StartWorkflowResponse', id: string } | null };

export type RerunMutationVariables = Exact<{
  runID: Scalars['ULID']['input'];
  fromStep: InputMaybe<RerunFromStepInput>;
}>;


export type RerunMutation = { __typename?: 'Mutation', rerun: string };

export type SetUpAccountMutationVariables = Exact<{ [key: string]: never; }>;


export type SetUpAccountMutation = { __typename?: 'Mutation', setUpAccount: { __typename?: 'SetUpAccountPayload', account: { __typename?: 'Account', id: string } | null } | null };

export type CreateUserMutationVariables = Exact<{ [key: string]: never; }>;


export type CreateUserMutation = { __typename?: 'Mutation', createUser: { __typename?: 'CreateUserPayload', user: { __typename?: 'User', id: string } | null } | null };

export type GetFunctionPauseStateQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  functionSlug: Scalars['String']['input'];
}>;


export type GetFunctionPauseStateQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', function: { __typename?: 'Workflow', id: string, isPaused: boolean } | null } };

export type GetIngestKeyQueryVariables = Exact<{
  environmentID: Scalars['ID']['input'];
  keyID: Scalars['ID']['input'];
}>;


export type GetIngestKeyQuery = { __typename?: 'Query', environment: { __typename?: 'Workspace', ingestKey: { __typename?: 'IngestKey', id: string, name: null | string, createdAt: string, presharedKey: string, url: string | null, metadata: Record<string, unknown> | null, source: string, filter: { __typename?: 'FilterList', type: string | null, ips: Array<string> | null, events: Array<string> | null } } } };

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

export const TraceDetailsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"isUserland"}},{"kind":"Field","name":{"kind":"Name","value":"userlandSpan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"spanName"}},{"kind":"Field","name":{"kind":"Name","value":"spanKind"}},{"kind":"Field","name":{"kind":"Name","value":"serviceName"}},{"kind":"Field","name":{"kind":"Name","value":"scopeName"}},{"kind":"Field","name":{"kind":"Name","value":"scopeVersion"}},{"kind":"Field","name":{"kind":"Name","value":"spanAttrs"}},{"kind":"Field","name":{"kind":"Name","value":"resourceAttrs"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"values"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"stepID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepType"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<TraceDetailsFragment, unknown>;
export const AchiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AchiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<AchiveAppMutation, AchiveAppMutationVariables>;
export const UnachiveAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnachiveApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnachiveAppMutation, UnachiveAppMutationVariables>;
export const GetArchivedAppBannerDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetArchivedAppBannerData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"isArchived"}}]}}]}}]}}]} as unknown as DocumentNode<GetArchivedAppBannerDataQuery, GetArchivedAppBannerDataQueryVariables>;
export const ResyncAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResyncApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resyncApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appExternalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appExternalID"}}},{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<ResyncAppMutation, ResyncAppMutationVariables>;
export const SyncNewAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncNewApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncNewApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<SyncNewAppMutation, SyncNewAppMutationVariables>;
export const SyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Sync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appVersion"}},{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<SyncQuery, SyncQueryVariables>;
export const CheckAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CheckApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"url"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appCheck"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"url"},"value":{"kind":"Variable","name":{"kind":"Name","value":"url"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apiOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"appID"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"authenticationSucceeded"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"env"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"eventAPIOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventKeyStatus"}},{"kind":"Field","name":{"kind":"Name","value":"extra"}},{"kind":"Field","name":{"kind":"Name","value":"framework"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"isReachable"}},{"kind":"Field","name":{"kind":"Name","value":"isSDK"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"respHeaders"}},{"kind":"Field","name":{"kind":"Name","value":"respStatusCode"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"serveOrigin"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"servePath"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"signingKeyStatus"}},{"kind":"Field","name":{"kind":"Name","value":"signingKeyFallbackStatus"}}]}}]}}]}}]} as unknown as DocumentNode<CheckAppQuery, CheckAppQueryVariables>;
export const AppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"App"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"appVersion"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}},{"kind":"Field","name":{"kind":"Name","value":"appVersion"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppQuery, AppQueryVariables>;
export const AppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Apps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AppsFilter"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"method"}},{"kind":"Field","name":{"kind":"Name","value":"isParentArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppsQuery, AppsQueryVariables>;
export const AppNavDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppNavData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isParentArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"method"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<AppNavDataQuery, AppNavDataQueryVariables>;
export const AppSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"app"},"name":{"kind":"Name","value":"appByExternalID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"externalID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"externalAppID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"syncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AppSyncsQuery, AppSyncsQueryVariables>;
export const LatestUnattachedSyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LatestUnattachedSync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}}]}}]} as unknown as DocumentNode<LatestUnattachedSyncQuery, LatestUnattachedSyncQueryVariables>;
export const UpdateAccountAddonQuantityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccountAddonQuantity"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"addonName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"quantity"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateAccountAddonQuantity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"addonName"},"value":{"kind":"Variable","name":{"kind":"Name","value":"addonName"}}},{"kind":"Argument","name":{"kind":"Name","value":"quantity"},"value":{"kind":"Variable","name":{"kind":"Name","value":"quantity"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountAddonQuantityMutation, UpdateAccountAddonQuantityMutationVariables>;
export const UpdateAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAccount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAccount"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"account"},"name":{"kind":"Name","value":"updateAccount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<UpdateAccountMutation, UpdateAccountMutationVariables>;
export const UpdatePaymentMethodDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePaymentMethod"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePaymentMethod"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]} as unknown as DocumentNode<UpdatePaymentMethodMutation, UpdatePaymentMethodMutationVariables>;
export const GetPaymentIntentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPaymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"paymentIntents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"amountLabel"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"invoiceURL"}}]}}]}}]}}]} as unknown as DocumentNode<GetPaymentIntentsQuery, GetPaymentIntentsQueryVariables>;
export const CreateStripeSubscriptionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripeSubscription"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StripeSubscriptionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createStripeSubscription"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clientSecret"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CreateStripeSubscriptionMutation, CreateStripeSubscriptionMutationVariables>;
export const UpdatePlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdatePlan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"planSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updatePlan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"planSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<UpdatePlanMutation, UpdatePlanMutationVariables>;
export const SubmitChurnSurveyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubmitChurnSurvey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"reason"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"feedback"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"email"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"accountID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"clerkUserID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"submitChurnSurvey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"reason"},"value":{"kind":"Variable","name":{"kind":"Name","value":"reason"}}},{"kind":"Argument","name":{"kind":"Name","value":"feedback"},"value":{"kind":"Variable","name":{"kind":"Name","value":"feedback"}}},{"kind":"Argument","name":{"kind":"Name","value":"email"},"value":{"kind":"Variable","name":{"kind":"Name","value":"email"}}},{"kind":"Argument","name":{"kind":"Name","value":"accountID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"accountID"}}},{"kind":"Argument","name":{"kind":"Name","value":"clerkUserID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"clerkUserID"}}}]}]}}]} as unknown as DocumentNode<SubmitChurnSurveyMutation, SubmitChurnSurveyMutationVariables>;
export const GetBillableStepsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableSteps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"usage"},"name":{"kind":"Name","value":"billableStepTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableStepsQuery, GetBillableStepsQueryVariables>;
export const GetBillableRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"usage"},"name":{"kind":"Name","value":"runCountTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableRunsQuery, GetBillableRunsQueryVariables>;
export const GetBillableExecutionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillableExecutions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"month"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"year"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"usage"},"name":{"kind":"Name","value":"executionTimeSeries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"timeOptions"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"month"},"value":{"kind":"Variable","name":{"kind":"Name","value":"month"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"year"},"value":{"kind":"Variable","name":{"kind":"Name","value":"year"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillableExecutionsQuery, GetBillableExecutionsQueryVariables>;
export const GetBillingInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"executions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillingInfoQuery, GetBillingInfoQueryVariables>;
export const CreateEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createWorkspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateEnvironmentMutation, CreateEnvironmentMutationVariables>;
export const EnableDatadogConnectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EnableDatadogConnection"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"organizationID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enableDatadogConnection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"organizationID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"organizationID"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<EnableDatadogConnectionMutation, EnableDatadogConnectionMutationVariables>;
export const FinishDatadogIntegrationDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"FinishDatadogIntegrationDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"orgName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"orgID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"authCode"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ddSite"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ddDomain"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"datadogOAuthCompleted"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"orgName"},"value":{"kind":"Variable","name":{"kind":"Name","value":"orgName"}}},{"kind":"Argument","name":{"kind":"Name","value":"orgID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"orgID"}}},{"kind":"Argument","name":{"kind":"Name","value":"authCode"},"value":{"kind":"Variable","name":{"kind":"Name","value":"authCode"}}},{"kind":"Argument","name":{"kind":"Name","value":"ddSite"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ddSite"}}},{"kind":"Argument","name":{"kind":"Name","value":"ddDomain"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ddDomain"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<FinishDatadogIntegrationDocumentMutation, FinishDatadogIntegrationDocumentMutationVariables>;
export const GetDatadogSetupDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDatadogSetupData"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"datadogConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"orgID"}},{"kind":"Field","name":{"kind":"Name","value":"orgName"}},{"kind":"Field","name":{"kind":"Name","value":"envID"}},{"kind":"Field","name":{"kind":"Name","value":"envName"}},{"kind":"Field","name":{"kind":"Name","value":"healthy"}},{"kind":"Field","name":{"kind":"Name","value":"lastErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"lastSentAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"datadogOrganizations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"datadogDomain"}},{"kind":"Field","name":{"kind":"Name","value":"datadogOrgName"}}]}}]}}]}}]} as unknown as DocumentNode<GetDatadogSetupDataQuery, GetDatadogSetupDataQueryVariables>;
export const DisableDatadogConnectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisableDatadogConnection"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"connectionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disableDatadogConnection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"connectionID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"connectionID"}}}]}]}}]} as unknown as DocumentNode<DisableDatadogConnectionMutation, DisableDatadogConnectionMutationVariables>;
export const RemoveDatadogOrganizationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveDatadogOrganization"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"organizationID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeDatadogOrganization"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"organizationID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"organizationID"}}}]}]}}]} as unknown as DocumentNode<RemoveDatadogOrganizationMutation, RemoveDatadogOrganizationMutationVariables>;
export const StartDatadogIntegrationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"StartDatadogIntegration"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ddSite"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ddDomain"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"datadogOAuthRedirectURL"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ddSite"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ddSite"}}},{"kind":"Argument","name":{"kind":"Name","value":"ddDomain"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ddDomain"}}}]}]}}]} as unknown as DocumentNode<StartDatadogIntegrationMutation, StartDatadogIntegrationMutationVariables>;
export const DisableEnvironmentAutoArchiveDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DisableEnvironmentAutoArchiveDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"disableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<DisableEnvironmentAutoArchiveDocumentMutation, DisableEnvironmentAutoArchiveDocumentMutationVariables>;
export const EnableEnvironmentAutoArchiveDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"EnableEnvironmentAutoArchive"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enableEnvironmentAutoArchive"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<EnableEnvironmentAutoArchiveMutation, EnableEnvironmentAutoArchiveMutationVariables>;
export const ArchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<ArchiveEnvironmentMutation, ArchiveEnvironmentMutationVariables>;
export const UnarchiveEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnarchiveEnvironment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveEnvironment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnarchiveEnvironmentMutation, UnarchiveEnvironmentMutationVariables>;
export const GetEventTypesV2Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypesV2"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"nameSearch"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventTypesV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"nameSearch"},"value":{"kind":"Variable","name":{"kind":"Name","value":"nameSearch"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypesV2Query, GetEventTypesV2QueryVariables>;
export const GetEventTypeVolumeV2Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypeVolumeV2"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventType"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"EnumValue","value":"hour"}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}},{"kind":"Field","name":{"kind":"Name","value":"slot"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeVolumeV2Query, GetEventTypeVolumeV2QueryVariables>;
export const GetEventTypeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventType"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventType"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventName"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeQuery, GetEventTypeQueryVariables>;
export const GetAllEventNamesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAllEventNames"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventTypesV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetAllEventNamesQuery, GetAllEventNamesQueryVariables>;
export const ArchiveEventDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveEvent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveEvent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentId"}}},{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<ArchiveEventMutation, ArchiveEventMutationVariables>;
export const GetLatestEventLogsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLatestEventLogs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"recent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"count"},"value":{"kind":"IntValue","value":"5"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetLatestEventLogsQuery, GetLatestEventLogsQueryVariables>;
export const GetEventKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"eventKeys"},"name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"value"},"name":{"kind":"Name","value":"presharedKey"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysQuery, GetEventKeysQueryVariables>;
export const GetEventsV2Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventsV2"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventNames"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"includeInternalEvents"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}},"defaultValue":{"kind":"BooleanValue","value":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventsV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"50"}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"eventNames"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventNames"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"includeInternalEvents"},"value":{"kind":"Variable","name":{"kind":"Name","value":"includeInternalEvents"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"runs"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventsV2Query, GetEventsV2QueryVariables>;
export const GetEventV2Document = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventV2"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"receivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"idempotencyKey"}},{"kind":"Field","name":{"kind":"Name","value":"occurredAt"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"source"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventV2Query, GetEventV2QueryVariables>;
export const GetEventPayloadDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventPayload"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"raw"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventPayloadQuery, GetEventPayloadQueryVariables>;
export const GetEventV2RunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventV2Runs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"eventID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"runs"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventV2RunsQuery, GetEventV2RunsQueryVariables>;
export const GetArchivedFuncBannerDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetArchivedFuncBannerData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"funcID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"funcID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetArchivedFuncBannerDataQuery, GetArchivedFuncBannerDataQueryVariables>;
export const CreateCancellationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateCancellation"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateCancellationInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createCancellation"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateCancellationMutation, CreateCancellationMutationVariables>;
export const GetCancellationRunCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCancellationRunCount"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMin"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMax"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellationRunCount"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"queuedAtMin"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMin"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"queuedAtMax"},"value":{"kind":"Variable","name":{"kind":"Name","value":"queuedAtMax"}}}]}}]}]}}]}}]}}]} as unknown as DocumentNode<GetCancellationRunCountQuery, GetCancellationRunCountQueryVariables>;
export const DeleteCancellationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteCancellation"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cancellationID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteCancellation"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"cancellationID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cancellationID"}}}]}]}}]} as unknown as DocumentNode<DeleteCancellationMutation, DeleteCancellationMutationVariables>;
export const GetFunctionRateLimitDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRateLimitDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"ratelimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_rate_limited_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRateLimitDocumentQuery, GetFunctionRateLimitDocumentQueryVariables>;
export const GetFunctionRunsMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionRunsMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"canceled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"failed"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionRunsMetricsQuery, GetFunctionRunsMetricsQueryVariables>;
export const GetFnMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnMetricsQuery, GetFnMetricsQueryVariables>;
export const GetFailedFunctionRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFailedFunctionRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"until"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"failedRuns"},"name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"ListValue","values":[{"kind":"EnumValue","value":"FAILED"}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"EnumValue","value":"ENDED_AT"}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"EnumValue","value":"ENDED_AT"}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFailedFunctionRunsQuery, GetFailedFunctionRunsQueryVariables>;
export const PauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cancelRunning"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}},{"kind":"Argument","name":{"kind":"Name","value":"cancelRunning"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cancelRunning"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<PauseFunctionMutation, PauseFunctionMutationVariables>;
export const UnpauseFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnpauseFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unpauseFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"fnID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<UnpauseFunctionMutation, UnpauseFunctionMutationVariables>;
export const GetSdkRequestMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSDKRequestMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"queued"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"ended"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSdkRequestMetricsQuery, GetSdkRequestMetricsQueryVariables>;
export const GetStepBacklogMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepBacklogMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"scheduled"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_scheduled","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"sleeping"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_sleeping","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepBacklogMetricsQuery, GetStepBacklogMetricsQueryVariables>;
export const GetStepsRunningMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStepsRunningMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"running"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"concurrencyLimit"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetStepsRunningMetricsQuery, GetStepsRunningMetricsQueryVariables>;
export const GetFnCancellationsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFnCancellations"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"fn"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fnSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellations"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cursor"}},{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","alias":{"kind":"Name","value":"envID"},"name":{"kind":"Name","value":"environmentID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAtMax"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAtMin"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFnCancellationsQuery, GetFnCancellationsQueryVariables>;
export const InsightsResultsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"InsightsResults"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"workspaceID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"insights"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"columnType"}}]}},{"kind":"Field","name":{"kind":"Name","value":"rows"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"values"}}]}}]}}]}}]} as unknown as DocumentNode<InsightsResultsQuery, InsightsResultsQueryVariables>;
export const GetEventTypeSchemasDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventTypeSchemas"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"nameSearch"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventTypesV2"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"nameSearch"},"value":{"kind":"Variable","name":{"kind":"Name","value":"nameSearch"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"latestSchema"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetEventTypeSchemasQuery, GetEventTypeSchemasQueryVariables>;
export const InsightsSavedQueriesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"InsightsSavedQueries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"insightsQueries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"creator"}},{"kind":"Field","name":{"kind":"Name","value":"lastEditor"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shared"}},{"kind":"Field","name":{"kind":"Name","value":"sql"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode<InsightsSavedQueriesQuery, InsightsSavedQueriesQueryVariables>;
export const CreateInsightsQueryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateInsightsQuery"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewInsightsQuery"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createInsightsQuery"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"creator"}},{"kind":"Field","name":{"kind":"Name","value":"lastEditor"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shared"}},{"kind":"Field","name":{"kind":"Name","value":"sql"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<CreateInsightsQueryMutation, CreateInsightsQueryMutationVariables>;
export const RemoveInsightsQueryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveInsightsQuery"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeInsightsQuery"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<RemoveInsightsQueryMutation, RemoveInsightsQueryMutationVariables>;
export const ShareInsightsQueryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ShareInsightsQuery"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"shareInsightsQuery"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"creator"}},{"kind":"Field","name":{"kind":"Name","value":"lastEditor"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shared"}},{"kind":"Field","name":{"kind":"Name","value":"sql"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<ShareInsightsQueryMutation, ShareInsightsQueryMutationVariables>;
export const UpdateInsightsQueryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateInsightsQuery"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateInsightsQuery"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateInsightsQuery"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"creator"}},{"kind":"Field","name":{"kind":"Name","value":"lastEditor"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shared"}},{"kind":"Field","name":{"kind":"Name","value":"sql"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<UpdateInsightsQueryMutation, UpdateInsightsQueryMutationVariables>;
export const CreateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<CreateVercelAppMutation, CreateVercelAppMutationVariables>;
export const UpdateVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<UpdateVercelAppMutation, UpdateVercelAppMutationVariables>;
export const RemoveVercelAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveVercelApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RemoveVercelAppInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeVercelApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"success"}}]}}]}}]} as unknown as DocumentNode<RemoveVercelAppMutation, RemoveVercelAppMutationVariables>;
export const UpdateIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}}]}}]}}]} as unknown as DocumentNode<UpdateIngestKeyMutation, UpdateIngestKeyMutationVariables>;
export const NewIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"NewIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<NewIngestKeyMutation, NewIngestKeyMutationVariables>;
export const DeleteEventKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteEventKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DeleteIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<DeleteEventKeyMutation, DeleteEventKeyMutationVariables>;
export const GetIngestKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeysQuery, GetIngestKeysQueryVariables>;
export const MetricsLookupsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"MetricsLookups"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}}]}},{"kind":"Field","name":{"kind":"Name","value":"workflows"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}}]}}]}}]}}]}}]} as unknown as DocumentNode<MetricsLookupsQuery, MetricsLookupsQueryVariables>;
export const AccountConcurrencyLookupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AccountConcurrencyLookup"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"marketplace"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AccountConcurrencyLookupQuery, AccountConcurrencyLookupQueryVariables>;
export const FunctionStatusMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"FunctionStatusMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"until"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"scope"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"MetricsScope"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"scheduled"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"started"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completed"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"groupBy"},"value":{"kind":"StringValue","value":"status","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"completedByFunction"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"EnumValue","value":"FN"}},{"kind":"ObjectField","name":{"kind":"Name","value":"groupBy"},"value":{"kind":"StringValue","value":"status","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"totals"},"name":{"kind":"Name","value":"scopedFunctionStatus"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"EnumValue","value":"FN"}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queued"}},{"kind":"Field","name":{"kind":"Name","value":"running"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}},{"kind":"Field","name":{"kind":"Name","value":"cancelled"}},{"kind":"Field","name":{"kind":"Name","value":"cancelled"}},{"kind":"Field","name":{"kind":"Name","value":"skipped"}}]}}]}}]}}]} as unknown as DocumentNode<FunctionStatusMetricsQuery, FunctionStatusMetricsQueryVariables>;
export const VolumeMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"VolumeMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"until"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"scope"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"MetricsScope"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"accountConcurrency"},"name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"from"}},{"kind":"Field","name":{"kind":"Name","value":"to"}},{"kind":"Field","name":{"kind":"Name","value":"granularity"}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"runsThroughput"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"function_run_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputEnded"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_ended_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputStarted"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_started_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sdkThroughputScheduled"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"sdk_req_scheduled_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"stepThroughput"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"backlog"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_scheduled","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"stepRunning"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"steps_running","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"functionIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"concurrency"},"name":{"kind":"Name","value":"scopedMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"concurrency_limit_reached_total","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"scope"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scope"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workerPercentageUsed"},"name":{"kind":"Name","value":"connectWorkerMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"worker_percentage_used","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"workspaceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"workerTotalCapacity"},"name":{"kind":"Name","value":"connectWorkerMetrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"StringValue","value":"worker_total_capacity","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"until"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"tagName"}},{"kind":"Field","name":{"kind":"Name","value":"tagValue"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"bucket"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<VolumeMetricsQuery, VolumeMetricsQueryVariables>;
export const QuickSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"QuickSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"term"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"quickSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"term"},"value":{"kind":"Variable","name":{"kind":"Name","value":"term"}}},{"kind":"Argument","name":{"kind":"Name","value":"envSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"event"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envSlug"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"functions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"run"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envSlug"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]}}]} as unknown as DocumentNode<QuickSearchQuery, QuickSearchQueryVariables>;
export const SyncOnboardingAppDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncOnboardingApp"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncNewApp"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"appURL"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appURL"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<SyncOnboardingAppMutation, SyncOnboardingAppMutationVariables>;
export const InvokeFunctionOnboardingDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"InvokeFunctionOnboarding"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"data"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"user"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Map"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"invokeFunction"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"data"},"value":{"kind":"Variable","name":{"kind":"Name","value":"data"}}},{"kind":"Argument","name":{"kind":"Name","value":"functionSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"user"},"value":{"kind":"Variable","name":{"kind":"Name","value":"user"}}}]}]}}]} as unknown as DocumentNode<InvokeFunctionOnboardingMutation, InvokeFunctionOnboardingMutationVariables>;
export const InvokeFunctionLookupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"InvokeFunctionLookup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}}]}}]}}]}}]}}]} as unknown as DocumentNode<InvokeFunctionLookupQuery, InvokeFunctionLookupQueryVariables>;
export const GetVercelAppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetVercelApps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}}]}},{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetVercelAppsQuery, GetVercelAppsQueryVariables>;
export const ProductionAppsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ProductionApps"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}}]}}]} as unknown as DocumentNode<ProductionAppsQuery, ProductionAppsQueryVariables>;
export const GetAccountEntitlementsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountEntitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountEntitlementsQuery, GetAccountEntitlementsQueryVariables>;
export const GetReplayRunCountsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplayRunCounts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"from"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"to"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"replayCounts"},"name":{"kind":"Name","value":"replayCounts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"from"}}},{"kind":"Argument","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"to"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"cancelledCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedPausedCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplayRunCountsQuery, GetReplayRunCountsQueryVariables>;
export const CreateFunctionReplayDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateFunctionReplay"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ReplayRunStatus"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createFunctionReplay"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fromRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fromRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"toRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"toRange"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"statusesV2"},"value":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CreateFunctionReplayMutation, CreateFunctionReplayMutationVariables>;
export const GetReplayDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplay"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"replayID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"replay"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"replayID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsScheduledCount"}},{"kind":"Field","name":{"kind":"Name","value":"fromRange"}},{"kind":"Field","name":{"kind":"Name","value":"toRange"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsProcessedCount"}},{"kind":"Field","name":{"kind":"Name","value":"filtersV2"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"statuses"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplayQuery, GetReplayQueryVariables>;
export const GetReplaysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReplays"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"replays"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsScheduledCount"}},{"kind":"Field","name":{"kind":"Name","value":"functionRunsProcessedCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetReplaysQuery, GetReplaysQueryVariables>;
export const GetRunTraceTriggerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTraceTrigger"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runTrigger"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"IDs"}},{"kind":"Field","name":{"kind":"Name","value":"payloads"}},{"kind":"Field","name":{"kind":"Name","value":"timestamp"}},{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"isBatch"}},{"kind":"Field","name":{"kind":"Name","value":"batchID"}},{"kind":"Field","name":{"kind":"Name","value":"cron"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceTriggerQuery, GetRunTraceTriggerQueryVariables>;
export const GetRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RunsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"preview"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}},"defaultValue":{"kind":"BooleanValue","value":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunCursor"}}},{"kind":"Argument","name":{"kind":"Name","value":"preview"},"value":{"kind":"Variable","name":{"kind":"Name","value":"preview"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"cronSchedule"}},{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isBatch"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"hasAI"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetRunsQuery, GetRunsQueryVariables>;
export const CountRunsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CountRuns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"FunctionRunStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RunsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"appIDs"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"until"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"fnSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"celQuery"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<CountRunsQuery, CountRunsQueryVariables>;
export const AppFilterDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AppFilter"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"env"},"name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"apps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<AppFilterQuery, AppFilterQueryVariables>;
export const SeatOverageCheckDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SeatOverageCheck"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<SeatOverageCheckQuery, SeatOverageCheckQueryVariables>;
export const CreateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<CreateSigningKeyMutation, CreateSigningKeyMutationVariables>;
export const DeleteSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"signingKeyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<DeleteSigningKeyMutation, DeleteSigningKeyMutationVariables>;
export const RotateSigningKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RotateSigningKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rotateSigningKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<RotateSigningKeyMutation, RotateSigningKeyMutationVariables>;
export const GetSigningKeysDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSigningKeys"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"signingKeys"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"decryptedValue"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isActive"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetSigningKeysQuery, GetSigningKeysQueryVariables>;
export const UnattachedSyncDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSync"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sync"},"name":{"kind":"Name","value":"deploy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"syncID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appVersion"}},{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","alias":{"kind":"Name","value":"removedFunctions"},"name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"syncedFunctions"},"name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncQuery, UnattachedSyncQueryVariables>;
export const UnattachedSyncsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UnattachedSyncs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"syncs"},"name":{"kind":"Name","value":"unattachedSyncs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"40"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"commitAuthor"}},{"kind":"Field","name":{"kind":"Name","value":"commitHash"}},{"kind":"Field","name":{"kind":"Name","value":"commitMessage"}},{"kind":"Field","name":{"kind":"Name","value":"commitRef"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}},{"kind":"Field","name":{"kind":"Name","value":"platform"}},{"kind":"Field","name":{"kind":"Name","value":"repoURL"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelDeploymentURL"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"vercelProjectURL"}}]}}]}}]}}]} as unknown as DocumentNode<UnattachedSyncsQuery, UnattachedSyncsQueryVariables>;
export const GetWorkerConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetWorkerConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1ConnectionStatus"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderByField"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}},"defaultValue":{"kind":"NullValue"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"orderBy"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderBy"}}}},"defaultValue":{"kind":"ListValue","values":[]}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"first"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workerConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"Variable","name":{"kind":"Name","value":"first"}}},{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"ListValue","values":[{"kind":"Variable","name":{"kind":"Name","value":"appID"}}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"Variable","name":{"kind":"Name","value":"orderBy"}}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"cursor"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"edges"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"node"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"gatewayId"}},{"kind":"Field","alias":{"kind":"Name","value":"instanceID"},"name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"workerIp"}},{"kind":"Field","name":{"kind":"Name","value":"maxWorkerConcurrency"}},{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastHeartbeatAt"}},{"kind":"Field","name":{"kind":"Name","value":"disconnectedAt"}},{"kind":"Field","name":{"kind":"Name","value":"disconnectReason"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLang"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"sdkPlatform"}},{"kind":"Field","name":{"kind":"Name","value":"appVersion"}},{"kind":"Field","name":{"kind":"Name","value":"functionCount"}},{"kind":"Field","name":{"kind":"Name","value":"cpuCores"}},{"kind":"Field","name":{"kind":"Name","value":"memBytes"}},{"kind":"Field","name":{"kind":"Name","value":"os"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasPreviousPage"}},{"kind":"Field","name":{"kind":"Name","value":"startCursor"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<GetWorkerConnectionsQuery, GetWorkerConnectionsQueryVariables>;
export const GetWorkerCountConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetWorkerCountConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"appID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1ConnectionStatus"}}}},"defaultValue":{"kind":"ListValue","values":[]}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConnectV1WorkerConnectionsOrderByField"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workerConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"appIDs"},"value":{"kind":"ListValue","values":[{"kind":"Variable","name":{"kind":"Name","value":"appID"}}]}},{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"timeField"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"orderBy"},"value":{"kind":"ListValue","values":[{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"field"},"value":{"kind":"Variable","name":{"kind":"Name","value":"timeField"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"direction"},"value":{"kind":"EnumValue","value":"DESC"}}]}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<GetWorkerCountConnectionsQuery, GetWorkerCountConnectionsQueryVariables>;
export const GetDeployssDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDeployss"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deploys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"appName"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"checksum"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"framework"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"sdkLanguage"}},{"kind":"Field","name":{"kind":"Name","value":"sdkVersion"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"deployedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removedFunctions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<GetDeployssQuery, GetDeployssQueryVariables>;
export const GetEnvironmentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspaces"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentsQuery, GetEnvironmentsQueryVariables>;
export const GetEnvironmentBySlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironmentBySlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentBySlugQuery, GetEnvironmentBySlugQueryVariables>;
export const GetDefaultEnvironmentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDefaultEnvironment"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"defaultEnv"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}}]}}]}}]} as unknown as DocumentNode<GetDefaultEnvironmentQuery, GetDefaultEnvironmentQueryVariables>;
export const GetFunctionsUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionsUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCompleted"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCancelled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"period"},"value":{"kind":"StringValue","value":"hour","block":false}},{"kind":"ObjectField","name":{"kind":"Name","value":"range"},"value":{"kind":"StringValue","value":"day","block":false}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsUsageQuery, GetFunctionsUsageQueryVariables>;
export const GetFunctionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"archived"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"archived"},"value":{"kind":"Variable","name":{"kind":"Name","value":"archived"}}},{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}}],"directives":[{"kind":"Directive","name":{"kind":"Name","value":"paginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"perPage"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}}]}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"perPage"}},{"kind":"Field","name":{"kind":"Name","value":"totalItems"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}}]}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionsQuery, GetFunctionsQueryVariables>;
export const GetFunctionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunction"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","alias":{"kind":"Name","value":"workflow"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalID"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"latestSync"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastSyncedAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"triggers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"failureHandler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"configuration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancellations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"event"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"condition"}}]}},{"kind":"Field","name":{"kind":"Name","value":"retries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isDefault"}}]}},{"kind":"Field","name":{"kind":"Name","value":"priority"}},{"kind":"Field","name":{"kind":"Name","value":"eventsBatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"maxSize"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"limit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"isPlanLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"rateLimit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"debounce"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"throttle"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"burst"}},{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"period"}}]}},{"kind":"Field","name":{"kind":"Name","value":"singleton"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionQuery, GetFunctionQueryVariables>;
export const GetFunctionUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionUsage"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Time"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workflow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"dailyStarts"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"started","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCancelled"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"cancelled","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyCompleted"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"completed","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"dailyFailures"},"name":{"kind":"Name","value":"usage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"opts"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"from"},"value":{"kind":"Variable","name":{"kind":"Name","value":"startTime"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"to"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endTime"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"event"},"value":{"kind":"StringValue","value":"errored","block":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"period"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"data"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slot"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionUsageQuery, GetFunctionUsageQueryVariables>;
export const EntitlementUsageDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EntitlementUsage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"addons"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"advancedObservability"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchased"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackChannel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectWorkers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"hipaa"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExport"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackChannel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectWorkerConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<EntitlementUsageQuery, EntitlementUsageQueryVariables>;
export const EntitlementUsageWithMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"EntitlementUsageWithMetrics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"addons"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"advancedObservability"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchased"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackChannel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectWorkers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"baseValue"}},{"kind":"Field","name":{"kind":"Name","value":"maxValue"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"executions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"overageAllowed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usage"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"hipaa"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExport"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackChannel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectWorkerConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<EntitlementUsageWithMetricsQuery, EntitlementUsageWithMetricsQueryVariables>;
export const GetCurrentPlanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCurrentPlan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"isLegacy"}},{"kind":"Field","name":{"kind":"Name","value":"isFree"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"addons"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"userCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackChannel"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectWorkers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"price"}},{"kind":"Field","name":{"kind":"Name","value":"purchaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"quantityPer"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nextInvoiceDate"}}]}}]}}]}}]} as unknown as DocumentNode<GetCurrentPlanQuery, GetCurrentPlanQueryVariables>;
export const GetBillingDetailsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetBillingDetails"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"billingEmail"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"paymentMethods"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brand"}},{"kind":"Field","name":{"kind":"Name","value":"last4"}},{"kind":"Field","name":{"kind":"Name","value":"expMonth"}},{"kind":"Field","name":{"kind":"Name","value":"expYear"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"default"}}]}}]}}]}}]} as unknown as DocumentNode<GetBillingDetailsQuery, GetBillingDetailsQueryVariables>;
export const GetPlansDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetPlans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"plans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isLegacy"}},{"kind":"Field","name":{"kind":"Name","value":"isFree"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"billingPeriod"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"eventSize"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"history"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"runCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stepCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetPlansQuery, GetPlansQueryVariables>;
export const MetricsEntitlementsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"MetricsEntitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"entitlements"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metricsExport"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportFreshness"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metricsExportGranularity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"limit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<MetricsEntitlementsQuery, MetricsEntitlementsQueryVariables>;
export const GetProductionWorkspaceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetProductionWorkspace"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"defaultEnv"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetProductionWorkspaceQuery, GetProductionWorkspaceQueryVariables>;
export const GetPostgresIntegrationsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"getPostgresIntegrations"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"statusDetail"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode<GetPostgresIntegrationsQuery, GetPostgresIntegrationsQueryVariables>;
export const TestCredentialsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testCredentials"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcTestCredentials"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestCredentialsMutation, TestCredentialsMutationVariables>;
export const TestReplicationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testReplication"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcTestLogicalReplication"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestReplicationMutation, TestReplicationMutationVariables>;
export const TestAutoSetupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"testAutoSetup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CDCConnectionInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcAutoSetup"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"steps"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<TestAutoSetupMutation, TestAutoSetupMutationVariables>;
export const CdcDeleteDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"cdcDelete"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cdcDelete"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ids"}}]}}]}}]} as unknown as DocumentNode<CdcDeleteMutation, CdcDeleteMutationVariables>;
export const GetSavedVercelProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSavedVercelProjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"marketplace"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"savedVercelProjects"},"name":{"kind":"Name","value":"vercelApps"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"workspaceID"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}}]}}]}}]}}]} as unknown as DocumentNode<GetSavedVercelProjectsQuery, GetSavedVercelProjectsQueryVariables>;
export const VercelIntegrationDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"VercelIntegration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"vercelIntegration"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"isMarketplace"}},{"kind":"Field","name":{"kind":"Name","value":"projects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"canChangeEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"deploymentProtection"}},{"kind":"Field","name":{"kind":"Name","value":"isEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"originOverride"}},{"kind":"Field","name":{"kind":"Name","value":"projectID"}},{"kind":"Field","name":{"kind":"Name","value":"protectionBypassSecret"}},{"kind":"Field","name":{"kind":"Name","value":"servePath"}}]}}]}}]}}]}}]} as unknown as DocumentNode<VercelIntegrationQuery, VercelIntegrationQueryVariables>;
export const ProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Profile"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"marketplace"}}]}}]}}]} as unknown as DocumentNode<ProfileQuery, ProfileQueryVariables>;
export const CancelRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UUID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"envID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}},{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<CancelRunMutation, CancelRunMutationVariables>;
export const GetEventKeysForBlankSlateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventKeysForBlankSlate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKeys"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"source"},"value":{"kind":"StringValue","value":"key","block":false}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode<GetEventKeysForBlankSlateQuery, GetEventKeysForBlankSlateQueryVariables>;
export const GetRunTraceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetRunTrace"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"preview"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"run"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"function"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"app"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"externalID"}}]}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}}]}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"trace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"preview"},"value":{"kind":"Variable","name":{"kind":"Name","value":"preview"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}},{"kind":"Field","name":{"kind":"Name","value":"childrenSpans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TraceDetails"}}]}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"hasAI"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TraceDetails"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"RunTraceSpan"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"attempts"}},{"kind":"Field","name":{"kind":"Name","value":"queuedAt"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"endedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isRoot"}},{"kind":"Field","name":{"kind":"Name","value":"isUserland"}},{"kind":"Field","name":{"kind":"Name","value":"userlandSpan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"spanName"}},{"kind":"Field","name":{"kind":"Name","value":"spanKind"}},{"kind":"Field","name":{"kind":"Name","value":"serviceName"}},{"kind":"Field","name":{"kind":"Name","value":"scopeName"}},{"kind":"Field","name":{"kind":"Name","value":"scopeVersion"}},{"kind":"Field","name":{"kind":"Name","value":"spanAttrs"}},{"kind":"Field","name":{"kind":"Name","value":"resourceAttrs"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"values"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"outputID"}},{"kind":"Field","name":{"kind":"Name","value":"stepID"}},{"kind":"Field","name":{"kind":"Name","value":"spanID"}},{"kind":"Field","name":{"kind":"Name","value":"stepOp"}},{"kind":"Field","name":{"kind":"Name","value":"stepType"}},{"kind":"Field","name":{"kind":"Name","value":"stepInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"InvokeStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggeringEventID"}},{"kind":"Field","name":{"kind":"Name","value":"functionID"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"returnEventID"}},{"kind":"Field","name":{"kind":"Name","value":"runID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SleepStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sleepUntil"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"WaitForEventStepInfo"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"eventName"}},{"kind":"Field","name":{"kind":"Name","value":"expression"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"foundEventID"}},{"kind":"Field","name":{"kind":"Name","value":"timedOut"}}]}}]}}]}}]} as unknown as DocumentNode<GetRunTraceQuery, GetRunTraceQueryVariables>;
export const TraceResultDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TraceResult"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"envID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"traceID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"envID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"runTraceSpanOutputByID"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"outputID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"traceID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"input"}},{"kind":"Field","name":{"kind":"Name","value":"error"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"stack"}},{"kind":"Field","name":{"kind":"Name","value":"cause"}}]}}]}}]}}]}}]} as unknown as DocumentNode<TraceResultQuery, TraceResultQueryVariables>;
export const RerunFunctionRunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RerunFunctionRun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryWorkflowRun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"workspaceID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"workflowID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionID"}}}]}},{"kind":"Argument","name":{"kind":"Name","value":"workflowRunID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionRunID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<RerunFunctionRunMutation, RerunFunctionRunMutationVariables>;
export const RerunDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"Rerun"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"runID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ULID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fromStep"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"RerunFromStepInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rerun"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"runID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"runID"}}},{"kind":"Argument","name":{"kind":"Name","value":"fromStep"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fromStep"}}}]}]}}]} as unknown as DocumentNode<RerunMutation, RerunMutationVariables>;
export const SetUpAccountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SetUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"setUpAccount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<SetUpAccountMutation, SetUpAccountMutationVariables>;
export const CreateUserDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createUser"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<CreateUserMutation, CreateUserMutationVariables>;
export const GetFunctionPauseStateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetFunctionPauseState"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"function"},"name":{"kind":"Name","value":"workflowBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"functionSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"isPaused"}}]}}]}}]}}]} as unknown as DocumentNode<GetFunctionPauseStateQuery, GetFunctionPauseStateQueryVariables>;
export const GetIngestKeyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetIngestKey"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"environment"},"name":{"kind":"Name","value":"workspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"environmentID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"presharedKey"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"filter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"ips"}},{"kind":"Field","name":{"kind":"Name","value":"events"}}]}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"source"}}]}}]}}]}}]} as unknown as DocumentNode<GetIngestKeyQuery, GetIngestKeyQueryVariables>;
export const CreateWebhookDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateWebhook"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NewIngestKey"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"key"},"name":{"kind":"Name","value":"createIngestKey"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]} as unknown as DocumentNode<CreateWebhookMutation, CreateWebhookMutationVariables>;
export const CompleteAwsMarketplaceSetupDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CompleteAWSMarketplaceSetup"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSMarketplaceSetupInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"completeAWSMarketplaceSetup"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<CompleteAwsMarketplaceSetupMutation, CompleteAwsMarketplaceSetupMutationVariables>;
export const GetAccountSupportInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAccountSupportInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"account"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"plan"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"amount"}},{"kind":"Field","name":{"kind":"Name","value":"features"}}]}}]}}]}}]} as unknown as DocumentNode<GetAccountSupportInfoQuery, GetAccountSupportInfoQueryVariables>;