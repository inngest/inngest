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

export type ApiKey = {
  __typename?: 'APIKey';
  createdAt: Scalars['Time']['output'];
  env?: Maybe<Workspace>;
  id: Scalars['UUID']['output'];
  maskedKey: Scalars['String']['output'];
  name: Scalars['String']['output'];
  scopes: Array<ApiKeyScope>;
};

export type ApiKeyCreateResult = {
  __typename?: 'APIKeyCreateResult';
  apiKey: ApiKey;
  plaintextKey: Scalars['String']['output'];
};

export type ApiKeyScope = {
  __typename?: 'APIKeyScope';
  allow: Array<Scalars['String']['output']>;
  deny: Array<Scalars['String']['output']>;
  name: Scalars['String']['output'];
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
  activeBanners: Array<Banner>;
  addons: Addons;
  apiKeys: Array<ApiKey>;
  appliedAddons: AppliedAddons;
  billingEmail: Scalars['String']['output'];
  constraintAPIEnrolled: Scalars['Boolean']['output'];
  createdAt: Scalars['Time']['output'];
  datadogConnections: Array<DatadogConnectionStatus>;
  datadogOrganizations: Array<DatadogOrganization>;
  entitlementUsage: EntitlementUsage;
  entitlements: Entitlements;
  id: Scalars['ID']['output'];
  insightsQueries: Array<InsightsQueryStatement>;
  marketplace?: Maybe<Marketplace>;
  name?: Maybe<Scalars['NullString']['output']>;
  paymentIntents: Array<PaymentIntent>;
  paymentMethods?: Maybe<Array<PaymentMethod>>;
  /**
   * Collections / payment status for the account. Null when the account is in
   * good standing (no overdue invoices and no failed payment). Powers the in-app
   * overdue-invoice banner.
   */
  paymentStatus?: Maybe<AccountPaymentStatus>;
  plan?: Maybe<BillingPlan>;
  quickSearch: QuickSearchResults;
  search: SearchResults;
  securityEmail?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
  subscription?: Maybe<BillingSubscription>;
  updatedAt: Scalars['Time']['output'];
  users: Array<User>;
  vercelIntegration?: Maybe<VercelIntegration>;
};


export type AccountApiKeysArgs = {
  workspaceID?: InputMaybe<Scalars['UUID']['input']>;
};


export type AccountConstraintApiEnrolledArgs = {
  inEffect?: InputMaybe<Scalars['Boolean']['input']>;
};


export type AccountQuickSearchArgs = {
  envSlug: Scalars['String']['input'];
  term: Scalars['String']['input'];
};


export type AccountSearchArgs = {
  opts: SearchInput;
};

export type AccountPaymentStatus = {
  __typename?: 'AccountPaymentStatus';
  /** When the pending action takes effect (downgrade/suspension), if scheduled. Null otherwise. */
  actionDate?: Maybe<Scalars['Time']['output']>;
  /** Total past-due amount across all overdue invoices, in cents. */
  amountDueCents: Scalars['Int']['output'];
  /** Pre-formatted amount for display, e.g. "$240.00". */
  amountDueLabel: Scalars['String']['output'];
  /** ISO 4217 currency code, e.g. "usd". */
  currency: Scalars['String']['output'];
  /** Days the oldest overdue invoice is past due. 0 if a payment failed but nothing is overdue yet. */
  daysPastDue: Scalars['Int']['output'];
  /** Whether the most recent payment attempt failed (card declined, etc.). */
  hasFailedPayment: Scalars['Boolean']['output'];
  /** Per-invoice detail for the /billing page banner. */
  overdueInvoices: Array<OverdueInvoice>;
  /** What happens at actionDate. Null when nothing is scheduled. */
  pendingAction?: Maybe<PaymentPendingAction>;
  /** Most direct link to resolve payment (hosted invoice URL or billing portal). Must be https. */
  resolveURL: Scalars['String']['output'];
  /** Highest severity across all open/overdue invoices. Drives banner color. */
  severity: PaymentStatusSeverity;
  /**
   * Machine-readable collections stage, computed server-side from invoice age and
   * payment state. Drives banner copy and /billing detail messaging.
   */
  stage: PaymentCollectionStage;
};

export type Addon = {
  __typename?: 'Addon';
  available: Scalars['Boolean']['output'];
  baseValue?: Maybe<Scalars['Int']['output']>;
  maxValue: Scalars['Int']['output'];
  name: Scalars['String']['output'];
  price?: Maybe<Scalars['Int']['output']>;
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
  accountID?: Maybe<Scalars['UUID']['output']>;
  advancedObservability: AdvancedObservabilityAddon;
  concurrency: Addon;
  connectWorkers: Addon;
  hipaa: Addon;
  planID?: Maybe<Scalars['UUID']['output']>;
  slackChannel: Addon;
  userCount: Addon;
};

export type AdvancedObservabilityAddon = {
  __typename?: 'AdvancedObservabilityAddon';
  available: Scalars['Boolean']['output'];
  entitlements: AdvancedObservabilityEntitlements;
  name: Scalars['String']['output'];
  price?: Maybe<Scalars['Int']['output']>;
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
  appVersion?: Maybe<Scalars['String']['output']>;
  archivedAt?: Maybe<Scalars['Time']['output']>;
  createdAt: Scalars['Time']['output'];
  externalID: Scalars['String']['output'];
  functionCount: Scalars['Int']['output'];
  functions: Array<Workflow>;
  id: Scalars['UUID']['output'];
  isArchived: Scalars['Boolean']['output'];
  isParentArchived: Scalars['Boolean']['output'];
  latestSync?: Maybe<Deploy>;
  method: AppMethod;
  name: Scalars['String']['output'];
  signingKeyRotationCheck: SigningKeyRotationCheck;
  syncs: Array<Deploy>;
};


export type AppLatestSyncArgs = {
  status?: InputMaybe<SyncStatus>;
};


export type AppSyncsArgs = {
  after?: InputMaybe<Scalars['Time']['input']>;
  first?: Scalars['Int']['input'];
};

export type AppCheckFieldBoolean = {
  __typename?: 'AppCheckFieldBoolean';
  value?: Maybe<Scalars['Boolean']['output']>;
};

export type AppCheckFieldString = {
  __typename?: 'AppCheckFieldString';
  value?: Maybe<Scalars['String']['output']>;
};

export type AppCheckResult = {
  __typename?: 'AppCheckResult';
  apiOrigin?: Maybe<AppCheckFieldString>;
  appID?: Maybe<AppCheckFieldString>;
  authenticationSucceeded?: Maybe<AppCheckFieldBoolean>;
  env?: Maybe<AppCheckFieldString>;
  error?: Maybe<Scalars['String']['output']>;
  eventAPIOrigin?: Maybe<AppCheckFieldString>;
  eventKeyStatus: SecretCheck;
  extra?: Maybe<Scalars['Map']['output']>;
  framework?: Maybe<AppCheckFieldString>;
  isReachable: Scalars['Boolean']['output'];
  isSDK: Scalars['Boolean']['output'];
  mode?: Maybe<SdkMode>;
  respHeaders?: Maybe<Scalars['Map']['output']>;
  respStatusCode?: Maybe<Scalars['Int']['output']>;
  sdkLanguage?: Maybe<AppCheckFieldString>;
  sdkVersion?: Maybe<AppCheckFieldString>;
  serveOrigin?: Maybe<AppCheckFieldString>;
  servePath?: Maybe<AppCheckFieldString>;
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
  concurrency?: Maybe<AppliedAddonMulti>;
  users?: Maybe<AppliedAddonMulti>;
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
  ingestSourceID?: Maybe<Scalars['ID']['output']>;
  name: Scalars['String']['output'];
  occurredAt: Scalars['Time']['output'];
  receivedAt: Scalars['Time']['output'];
  skippedFunctionRuns: Array<SkippedFunctionRun>;
  source?: Maybe<IngestKey>;
  version: Scalars['String']['output'];
};

export type AvailableAddons = {
  __typename?: 'AvailableAddons';
  concurrency?: Maybe<AddonMulti>;
  users?: Maybe<AddonMulti>;
};

export type Banner = {
  __typename?: 'Banner';
  body: Scalars['String']['output'];
  cta?: Maybe<BannerCta>;
  dismissible: Scalars['Boolean']['output'];
  id: Scalars['ID']['output'];
  severity: BannerSeverity;
  title?: Maybe<Scalars['String']['output']>;
};

export type BannerCta = {
  __typename?: 'BannerCTA';
  label: Scalars['String']['output'];
  url: Scalars['String']['output'];
};

export enum BannerSeverity {
  Error = 'ERROR',
  Info = 'INFO',
  Success = 'SUCCESS',
  Warning = 'WARNING'
}

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
  description?: Maybe<Scalars['String']['output']>;
  engine: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  status: CdcStatus;
  statusDetail?: Maybe<Scalars['Map']['output']>;
  updatedAt: Scalars['Time']['output'];
  watermark?: Maybe<Scalars['Map']['output']>;
};

export type CdcConnectionInput = {
  adminConn: Scalars['String']['input'];
  engine: Scalars['String']['input'];
  name: Scalars['String']['input'];
  replicaConn?: InputMaybe<Scalars['String']['input']>;
};

export type CdcSetupResponse = {
  __typename?: 'CDCSetupResponse';
  error?: Maybe<Scalars['String']['output']>;
  steps?: Maybe<Scalars['Map']['output']>;
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
  expression?: Maybe<Scalars['String']['output']>;
  functionID: Scalars['UUID']['output'];
  id: Scalars['ULID']['output'];
  name?: Maybe<Scalars['String']['output']>;
  queuedAtMax: Scalars['Time']['output'];
  queuedAtMin?: Maybe<Scalars['Time']['output']>;
};

export type CancellationConfiguration = {
  __typename?: 'CancellationConfiguration';
  condition?: Maybe<Scalars['String']['output']>;
  event: Scalars['String']['output'];
  timeout?: Maybe<Scalars['String']['output']>;
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
  data?: Maybe<Scalars['JSON']['output']>;
  message: Scalars['String']['output'];
};

export type ConcurrencyConfiguration = {
  __typename?: 'ConcurrencyConfiguration';
  key?: Maybe<Scalars['String']['output']>;
  limit: ConcurrencyLimitConfiguration;
  scope: ConcurrencyScope;
};

export type ConcurrencyLimitConfiguration = {
  __typename?: 'ConcurrencyLimitConfiguration';
  isPlanLimit?: Maybe<Scalars['Boolean']['output']>;
  value: Scalars['Int']['output'];
};

export enum ConcurrencyScope {
  Account = 'ACCOUNT',
  Environment = 'ENVIRONMENT',
  Function = 'FUNCTION'
}

export type ConfirmSubscriptionUpgradeResponse = {
  __typename?: 'ConfirmSubscriptionUpgradeResponse';
  account?: Maybe<Account>;
  message: Scalars['String']['output'];
  success: Scalars['Boolean']['output'];
};

export enum ConnectV1ConnectionStatus {
  Connected = 'CONNECTED',
  Disconnected = 'DISCONNECTED',
  Disconnecting = 'DISCONNECTING',
  Draining = 'DRAINING',
  Ready = 'READY'
}

export type ConnectV1WorkerConnection = {
  __typename?: 'ConnectV1WorkerConnection';
  app?: Maybe<App>;
  appID?: Maybe<Scalars['UUID']['output']>;
  appName?: Maybe<Scalars['String']['output']>;
  appVersion?: Maybe<Scalars['String']['output']>;
  buildId?: Maybe<Scalars['String']['output']>;
  connectedAt: Scalars['Time']['output'];
  cpuCores: Scalars['Int']['output'];
  /** @deprecated buildId is deprecated. Use appVersion instead. */
  deploy?: Maybe<Deploy>;
  disconnectReason?: Maybe<Scalars['String']['output']>;
  disconnectedAt?: Maybe<Scalars['Time']['output']>;
  functionCount: Scalars['Int']['output'];
  gatewayId: Scalars['ULID']['output'];
  id: Scalars['ULID']['output'];
  instanceId: Scalars['String']['output'];
  lastHeartbeatAt?: Maybe<Scalars['Time']['output']>;
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

export type CreateStripeSubscriptionResponse = {
  __typename?: 'CreateStripeSubscriptionResponse';
  clientSecret?: Maybe<Scalars['String']['output']>;
  message: Scalars['String']['output'];
  subscriptionId?: Maybe<Scalars['String']['output']>;
};

export type CreateUserPayload = {
  __typename?: 'CreateUserPayload';
  user?: Maybe<User>;
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
  lastErrorMessage?: Maybe<Scalars['String']['output']>;
  lastSentAt?: Maybe<Scalars['Time']['output']>;
  orgID: Scalars['UUID']['output'];
  orgName: Scalars['String']['output'];
};

export type DatadogOrganization = {
  __typename?: 'DatadogOrganization';
  createdAt: Scalars['Time']['output'];
  datadogDomain: Scalars['String']['output'];
  datadogOrgID?: Maybe<Scalars['String']['output']>;
  datadogOrgName?: Maybe<Scalars['String']['output']>;
  datadogSite: Scalars['String']['output'];
  id: Scalars['UUID']['output'];
  updatedAt: Scalars['Time']['output'];
};

export type DebounceConfiguration = {
  __typename?: 'DebounceConfiguration';
  key?: Maybe<Scalars['String']['output']>;
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
  appVersion?: Maybe<Scalars['String']['output']>;
  authorID?: Maybe<Scalars['UUID']['output']>;
  checksum: Scalars['String']['output'];
  commitAuthor?: Maybe<Scalars['String']['output']>;
  commitHash?: Maybe<Scalars['String']['output']>;
  commitMessage?: Maybe<Scalars['String']['output']>;
  commitRef?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Time']['output'];
  deployedFunctions: Array<Workflow>;
  dupeCount: Scalars['Int']['output'];
  error?: Maybe<Scalars['String']['output']>;
  framework?: Maybe<Scalars['String']['output']>;
  functionCount?: Maybe<Scalars['Int']['output']>;
  id: Scalars['UUID']['output'];
  idempotencyKey?: Maybe<Scalars['String']['output']>;
  lastSyncedAt: Scalars['Time']['output'];
  metadata: Scalars['Map']['output'];
  platform?: Maybe<Scalars['String']['output']>;
  prevFunctionCount?: Maybe<Scalars['Int']['output']>;
  removedFunctions: Array<Workflow>;
  repoURL?: Maybe<Scalars['String']['output']>;
  sdkLanguage: Scalars['String']['output'];
  sdkVersion: Scalars['String']['output'];
  status: Scalars['String']['output'];
  syncKind?: Maybe<Scalars['String']['output']>;
  trustProbeStatus?: Maybe<Scalars['String']['output']>;
  url?: Maybe<Scalars['String']['output']>;
  vercelDeploymentID?: Maybe<Scalars['String']['output']>;
  vercelDeploymentURL?: Maybe<Scalars['String']['output']>;
  vercelProjectID?: Maybe<Scalars['String']['output']>;
  vercelProjectURL?: Maybe<Scalars['String']['output']>;
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
  limit?: Maybe<Scalars['Int']['output']>;
};

export type EntitlementConnectWorkerConnections = {
  __typename?: 'EntitlementConnectWorkerConnections';
  limit?: Maybe<Scalars['Int']['output']>;
};

export type EntitlementEvents = {
  __typename?: 'EntitlementEvents';
  limit?: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementExecutions = {
  __typename?: 'EntitlementExecutions';
  limit?: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementInt = {
  __typename?: 'EntitlementInt';
  limit: Scalars['Int']['output'];
};

export type EntitlementNullableInt = {
  __typename?: 'EntitlementNullableInt';
  limit?: Maybe<Scalars['Int']['output']>;
};

export type EntitlementRunCount = {
  __typename?: 'EntitlementRunCount';
  limit?: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
  usage: Scalars['Int']['output'];
};

export type EntitlementStepCount = {
  __typename?: 'EntitlementStepCount';
  limit?: Maybe<Scalars['Int']['output']>;
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
  limit?: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementUsageStepCount = {
  __typename?: 'EntitlementUsageStepCount';
  current: Scalars['Int']['output'];
  limit?: Maybe<Scalars['Int']['output']>;
  overageAllowed: Scalars['Boolean']['output'];
};

export type EntitlementUserCount = {
  __typename?: 'EntitlementUserCount';
  limit?: Maybe<Scalars['Int']['output']>;
  usage: Scalars['Int']['output'];
};

export type Entitlements = {
  __typename?: 'Entitlements';
  accountID?: Maybe<Scalars['UUID']['output']>;
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
  planID?: Maybe<Scalars['UUID']['output']>;
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
  description?: Maybe<Scalars['String']['output']>;
  firstSeen?: Maybe<Scalars['Time']['output']>;
  integrationName?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  schemaSource?: Maybe<Scalars['SchemaSource']['output']>;
  usage: Usage;
  versionCount: Scalars['Int']['output'];
  versions: Array<Maybe<EventType>>;
  workflows: Array<Workflow>;
  workspaceID?: Maybe<Scalars['UUID']['output']>;
};


export type EventUsageArgs = {
  opts?: InputMaybe<UsageInput>;
};


export type EventVersionsArgs = {
  versions?: InputMaybe<Array<Scalars['String']['input']>>;
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

export type EventSource = {
  __typename?: 'EventSource';
  id: Scalars['ID']['output'];
  name?: Maybe<Scalars['String']['output']>;
  sourceKind: Scalars['String']['output'];
};

export type EventType = {
  __typename?: 'EventType';
  createdAt?: Maybe<Scalars['Time']['output']>;
  cueType: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  jsonSchema: Scalars['Map']['output'];
  name: Scalars['String']['output'];
  typescript: Scalars['String']['output'];
  updatedAt?: Maybe<Scalars['Time']['output']>;
  version: Scalars['String']['output'];
};

export type EventTypeV2 = {
  __typename?: 'EventTypeV2';
  envID: Scalars['UUID']['output'];
  functions: FunctionsConnection;
  latestSchema?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  usage: Usage;
};


export type EventTypeV2FunctionsArgs = {
  after?: InputMaybe<Scalars['String']['input']>;
  first?: Scalars['Int']['input'];
};


export type EventTypeV2LatestSchemaArgs = {
  format?: EventSchemaFormat;
};


export type EventTypeV2UsageArgs = {
  opts?: InputMaybe<UsageInput>;
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
  idempotencyKey?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  occurredAt: Scalars['Time']['output'];
  raw: Scalars['String']['output'];
  receivedAt: Scalars['Time']['output'];
  runs: Array<FunctionRunV2>;
  source?: Maybe<EventSource>;
  version?: Maybe<Scalars['String']['output']>;
};

export type EventsBatchConfiguration = {
  __typename?: 'EventsBatchConfiguration';
  key?: Maybe<Scalars['String']['output']>;
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

export type Experiment = {
  __typename?: 'Experiment';
  firstSeen: Scalars['Time']['output'];
  functionID: Scalars['ID']['output'];
  functionSlug: Scalars['String']['output'];
  lastSeen: Scalars['Time']['output'];
  name: Scalars['String']['output'];
  selectionStrategy: Scalars['String']['output'];
  totalRuns: Scalars['Int']['output'];
  variantCount: Scalars['Int']['output'];
  variants: Array<Scalars['String']['output']>;
};

export type ExperimentDetail = {
  __typename?: 'ExperimentDetail';
  firstSeen: Scalars['Time']['output'];
  lastSeen: Scalars['Time']['output'];
  name: Scalars['String']['output'];
  selectionStrategy: Scalars['String']['output'];
  variantWeights: Array<VariantWeight>;
  variants: Array<ExperimentVariantMetrics>;
};

export type ExperimentScoringConfig = {
  __typename?: 'ExperimentScoringConfig';
  experimentName: Scalars['String']['output'];
  metrics: Array<ExperimentScoringMetric>;
  updatedAt: Scalars['Time']['output'];
};

export type ExperimentScoringMetric = {
  __typename?: 'ExperimentScoringMetric';
  displayName: Scalars['String']['output'];
  enabled: Scalars['Boolean']['output'];
  invert: Scalars['Boolean']['output'];
  key: Scalars['String']['output'];
  kind: ScoreKind;
  labelBest: Scalars['String']['output'];
  labelWorst: Scalars['String']['output'];
  maxValue: Scalars['Float']['output'];
  minValue: Scalars['Float']['output'];
  points: Scalars['Int']['output'];
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

export type ExperimentVariantMetrics = {
  __typename?: 'ExperimentVariantMetrics';
  metrics: Array<VariantMetric>;
  runCount: Scalars['Int']['output'];
  variantName: Scalars['String']['output'];
};

export type FilterList = {
  __typename?: 'FilterList';
  events?: Maybe<Array<Scalars['String']['output']>>;
  ips?: Maybe<Array<Scalars['IP']['output']>>;
  type?: Maybe<Scalars['FilterType']['output']>;
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
  debounce?: Maybe<DebounceConfiguration>;
  eventsBatch?: Maybe<EventsBatchConfiguration>;
  priority?: Maybe<Scalars['String']['output']>;
  rateLimit?: Maybe<RateLimitConfiguration>;
  retries: RetryConfiguration;
  singleton?: Maybe<SingletonConfiguration>;
  throttle?: Maybe<ThrottleConfiguration>;
};

export type FunctionReplay = {
  __typename?: 'FunctionReplay';
  createdAt: Scalars['Time']['output'];
  endedAt: Scalars['Time']['output'];
  id: Scalars['UUID']['output'];
  name?: Maybe<Scalars['String']['output']>;
  scheduledRunCount: Scalars['Int']['output'];
  totalRunCount?: Maybe<Scalars['Int']['output']>;
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  accountID: Scalars['UUID']['output'];
  batchID?: Maybe<Scalars['ULID']['output']>;
  canRerun?: Maybe<Scalars['Boolean']['output']>;
  endedAt?: Maybe<Scalars['Time']['output']>;
  event?: Maybe<ArchivedEvent>;
  eventID?: Maybe<Scalars['ULID']['output']>;
  events?: Maybe<Array<ArchivedEvent>>;
  function: Workflow;
  id: Scalars['ULID']['output'];
  output?: Maybe<Scalars['Bytes']['output']>;
  startedAt: Scalars['Time']['output'];
  status: FunctionRunStatus;
  workflowID: Scalars['UUID']['output'];
  workflowVersion?: Maybe<WorkflowVersion>;
  workflowVersionInt: Scalars['Int']['output'];
  workspaceID: Scalars['UUID']['output'];
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
  batchCreatedAt?: Maybe<Scalars['Time']['output']>;
  cronSchedule?: Maybe<Scalars['String']['output']>;
  deferredFrom: Array<RunDeferredFrom>;
  defers: Array<RunDefer>;
  endedAt?: Maybe<Scalars['Time']['output']>;
  eventName?: Maybe<Scalars['String']['output']>;
  function: Workflow;
  functionID: Scalars['UUID']['output'];
  hasAI: Scalars['Boolean']['output'];
  id: Scalars['ULID']['output'];
  isBatch: Scalars['Boolean']['output'];
  isDeferred: Scalars['Boolean']['output'];
  output?: Maybe<Scalars['Bytes']['output']>;
  queuedAt: Scalars['Time']['output'];
  siblingDefers: Array<RunDefer>;
  sourceID?: Maybe<Scalars['String']['output']>;
  startedAt?: Maybe<Scalars['Time']['output']>;
  status: FunctionRunStatus;
  trace?: Maybe<RunTraceSpan>;
  traceID: Scalars['String']['output'];
  triggerIDs: Array<Scalars['ULID']['output']>;
  workspaceID: Scalars['UUID']['output'];
};


export type FunctionRunV2TraceArgs = {
  preview?: InputMaybe<Scalars['Boolean']['input']>;
};

export type FunctionRunV2Edge = {
  __typename?: 'FunctionRunV2Edge';
  cursor: Scalars['String']['output'];
  node: FunctionRunV2;
};

export type FunctionTrigger = {
  __typename?: 'FunctionTrigger';
  condition?: Maybe<Scalars['String']['output']>;
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

export type IngestKey = {
  __typename?: 'IngestKey';
  createdAt: Scalars['Time']['output'];
  filter: FilterList;
  id: Scalars['ID']['output'];
  metadata?: Maybe<Scalars['Map']['output']>;
  name: Scalars['NullString']['output'];
  presharedKey: Scalars['String']['output'];
  source: Scalars['IngestSource']['output'];
  url?: Maybe<Scalars['String']['output']>;
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
  Dynamic = 'DYNAMIC',
  Number = 'NUMBER',
  String = 'STRING',
  Unknown = 'UNKNOWN'
}

export type InsightsDiagnostic = {
  __typename?: 'InsightsDiagnostic';
  code: Scalars['InsightsDiagnosticCode']['output'];
  message: Scalars['String']['output'];
  position?: Maybe<InsightsDiagnosticPosition>;
  severity: Scalars['InsightsDiagnosticSeverity']['output'];
};

export type InsightsDiagnosticPosition = {
  __typename?: 'InsightsDiagnosticPosition';
  context: Scalars['String']['output'];
  end: Scalars['Int']['output'];
  start: Scalars['Int']['output'];
};

export type InsightsMetricOrderBy = {
  column: Scalars['String']['input'];
  direction: InsightsMetricOrderByDirection;
};

export enum InsightsMetricOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
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
  diagnostics?: Maybe<Array<InsightsDiagnostic>>;
  query: Scalars['String']['output'];
  rows: Array<InsightsRow>;
};

export type InsightsRow = {
  __typename?: 'InsightsRow';
  values: Array<Scalars['String']['output']>;
};

export type InvokeStepInfo = {
  __typename?: 'InvokeStepInfo';
  functionID: Scalars['String']['output'];
  returnEventID?: Maybe<Scalars['ULID']['output']>;
  runID?: Maybe<Scalars['ULID']['output']>;
  timedOut?: Maybe<Scalars['Boolean']['output']>;
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
  archiveEvent?: Maybe<Event>;
  archiveWorkflow?: Maybe<WorkflowResponse>;
  cancelRun: FunctionRun;
  cdcAutoSetup: CdcSetupResponse;
  cdcDelete: DeleteResponse;
  cdcManualSetup: CdcSetupResponse;
  cdcTestCredentials: CdcSetupResponse;
  cdcTestLogicalReplication: CdcSetupResponse;
  cdcTestSetup: CdcSetupResponse;
  completeAWSMarketplaceSetup?: Maybe<AwsMarketplaceSetupResponse>;
  confirmSubscriptionUpgrade: ConfirmSubscriptionUpgradeResponse;
  createAPIKey: ApiKeyCreateResult;
  createCancellation: Cancellation;
  createFunctionReplay: Replay;
  createIngestKey: IngestKey;
  createInsightsQuery: InsightsQueryStatement;
  createSigningKey: SigningKey;
  createStripeSubscription: CreateStripeSubscriptionResponse;
  createUser?: Maybe<CreateUserPayload>;
  createVercelApp?: Maybe<CreateVercelAppResponse>;
  createWorkspace: Array<Maybe<Workspace>>;
  datadogOAuthCompleted: DatadogOrganization;
  datadogOAuthRedirectURL: Scalars['String']['output'];
  deleteAPIKey: Scalars['Boolean']['output'];
  deleteCancellation: Scalars['ULID']['output'];
  deleteIngestKey?: Maybe<DeleteResponse>;
  deleteSigningKey: SigningKey;
  disableDatadogConnection: Scalars['UUID']['output'];
  disableEnvironmentAutoArchive: Workspace;
  enableDatadogConnection: DatadogConnectionStatus;
  enableEnvironmentAutoArchive: Workspace;
  enrollToConstraintAPI: Scalars['Boolean']['output'];
  invokeFunction?: Maybe<Scalars['Boolean']['output']>;
  pauseFunction: Workflow;
  removeDatadogOrganization: Scalars['UUID']['output'];
  removeInsightsQuery?: Maybe<DeleteUlidResponse>;
  removeVercelApp?: Maybe<RemoveVercelAppResponse>;
  rerun: Scalars['ULID']['output'];
  resyncApp: SyncResponse;
  retryWorkflowRun?: Maybe<StartWorkflowResponse>;
  rotateSigningKey: SigningKey;
  setAccountEntitlement: Scalars['UUID']['output'];
  setUpAccount?: Maybe<SetUpAccountPayload>;
  shareInsightsQuery: InsightsQueryStatement;
  submitChurnSurvey: Scalars['Boolean']['output'];
  syncNewApp: SyncResponse;
  unarchiveApp: App;
  unarchiveEnvironment: Workspace;
  unpauseFunction: Workflow;
  updateAPIKey: ApiKey;
  updateAccount: Account;
  updateAccountAddonQuantity: Addon;
  updateExperimentScoringConfig: ExperimentScoringConfig;
  updateIngestKey: IngestKey;
  updateInsightsQuery: InsightsQueryStatement;
  updatePaymentMethod?: Maybe<Array<PaymentMethod>>;
  updatePlan: Account;
  updateVercelApp?: Maybe<UpdateVercelAppResponse>;
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


export type MutationConfirmSubscriptionUpgradeArgs = {
  subscriptionId: Scalars['String']['input'];
};


export type MutationCreateApiKeyArgs = {
  input: CreateApiKeyInput;
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


export type MutationDeleteApiKeyArgs = {
  id: Scalars['UUID']['input'];
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
  data?: InputMaybe<Scalars['Map']['input']>;
  envID: Scalars['UUID']['input'];
  functionSlug: Scalars['String']['input'];
  meta?: InputMaybe<Scalars['Map']['input']>;
  user?: InputMaybe<Scalars['Map']['input']>;
};


export type MutationPauseFunctionArgs = {
  cancelRunning?: InputMaybe<Scalars['Boolean']['input']>;
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
  envID?: InputMaybe<Scalars['UUID']['input']>;
  fromStep?: InputMaybe<RerunFromStepInput>;
  runID: Scalars['ULID']['input'];
};


export type MutationResyncAppArgs = {
  appExternalID: Scalars['String']['input'];
  appURL?: InputMaybe<Scalars['String']['input']>;
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
  feedback?: InputMaybe<Scalars['String']['input']>;
  reason: Scalars['String']['input'];
  renderedOrder?: InputMaybe<Array<Scalars['String']['input']>>;
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


export type MutationUpdateApiKeyArgs = {
  input: UpdateApiKeyInput;
};


export type MutationUpdateAccountArgs = {
  input: UpdateAccount;
};


export type MutationUpdateAccountAddonQuantityArgs = {
  addonName: Scalars['String']['input'];
  quantity: Scalars['Int']['input'];
};


export type MutationUpdateExperimentScoringConfigArgs = {
  experimentName: Scalars['String']['input'];
  functionID: Scalars['ID']['input'];
  metrics: Array<ExperimentScoringMetricInput>;
  workspaceID: Scalars['ID']['input'];
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
  slug?: InputMaybe<Scalars['String']['input']>;
  to?: InputMaybe<Scalars['ID']['input']>;
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

export type OverdueInvoice = {
  __typename?: 'OverdueInvoice';
  amountCents: Scalars['Int']['output'];
  /** e.g. "$120.00" */
  amountLabel: Scalars['String']['output'];
  /** last payment attempt, null if none */
  attemptedAt?: Maybe<Scalars['Time']['output']>;
  currency: Scalars['String']['output'];
  daysPastDue: Scalars['Int']['output'];
  dueAt: Scalars['Time']['output'];
  /** e.g. "card_declined", null if none */
  failureReason?: Maybe<Scalars['String']['output']>;
  /** Stripe invoice ID (e.g. "in_123"). Typed as String because IDs in this schema are UUIDs; invoice IDs are not. */
  id: Scalars['String']['output'];
  /** hosted invoice / pay link (https), null if none */
  invoiceURL?: Maybe<Scalars['String']['output']>;
  /** underlying invoice status (open, uncollectible, …) */
  status: Scalars['String']['output'];
};

/** The pagination information in a connection. */
export type PageInfo = {
  __typename?: 'PageInfo';
  /** When paginating forward, the cursor to query the next page. */
  endCursor?: Maybe<Scalars['String']['output']>;
  /** Indicates if there are any pages subsequent to the current page. */
  hasNextPage: Scalars['Boolean']['output'];
  /** Indicates if there are any pages prior to the current page. */
  hasPreviousPage: Scalars['Boolean']['output'];
  /** When paginating backward, the cursor to query the previous page. */
  startCursor?: Maybe<Scalars['String']['output']>;
};

export type PageResults = {
  __typename?: 'PageResults';
  cursor?: Maybe<Scalars['String']['output']>;
  page: Scalars['Int']['output'];
  perPage: Scalars['Int']['output'];
  totalItems?: Maybe<Scalars['Int']['output']>;
  totalPages?: Maybe<Scalars['Int']['output']>;
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

export type PaymentIntent = {
  __typename?: 'PaymentIntent';
  amountLabel: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  description: Scalars['String']['output'];
  invoiceURL?: Maybe<Scalars['String']['output']>;
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
  deploys?: Maybe<Array<Deploy>>;
  envBySlug?: Maybe<Workspace>;
  envs: EnvsConnection;
  events?: Maybe<PaginatedEvents>;
  executionTimeSeries: Array<TimeSeries>;
  experimentDetail: ExperimentDetail;
  experimentInsightsQuery: Scalars['String']['output'];
  experimentScoringConfig: ExperimentScoringConfig;
  experiments: Array<Experiment>;
  insights: InsightsResponse;
  insightsMetric?: Maybe<InsightsResponse>;
  insightsQuery: InsightsQueryStatement;
  metrics: MetricsResponse;
  plans: Array<Maybe<BillingPlan>>;
  runCountTimeSeries: Array<TimeSeries>;
  scoreNames: Array<Score>;
  scoreTimeSeries: Array<ScoreSeries>;
  session?: Maybe<Session>;
  sessionRuns: Array<SessionRun>;
  sessions: Array<SessionGroup>;
  workspace: Workspace;
  workspaces?: Maybe<Array<Workspace>>;
};


export type QueryBillableStepTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryDeployArgs = {
  id: Scalars['ID']['input'];
};


export type QueryDeploysArgs = {
  workspaceID?: InputMaybe<Scalars['ID']['input']>;
};


export type QueryEnvBySlugArgs = {
  slug: Scalars['String']['input'];
};


export type QueryEnvsArgs = {
  after?: InputMaybe<Scalars['String']['input']>;
  filter?: InputMaybe<EnvsFilter>;
  first?: Scalars['Int']['input'];
};


export type QueryEventsArgs = {
  query?: InputMaybe<EventQuery>;
};


export type QueryExecutionTimeSeriesArgs = {
  timeOptions: TimeSeriesOptions;
};


export type QueryExperimentDetailArgs = {
  experimentName: Scalars['String']['input'];
  functionID: Scalars['ID']['input'];
  timeRange?: InputMaybe<TimeRangeInput>;
  variantFilter?: InputMaybe<Scalars['String']['input']>;
  workspaceID: Scalars['ID']['input'];
};


export type QueryExperimentInsightsQueryArgs = {
  experimentName: Scalars['String']['input'];
  functionID: Scalars['ID']['input'];
  timeRange?: InputMaybe<TimeRangeInput>;
  workspaceID: Scalars['ID']['input'];
};


export type QueryExperimentScoringConfigArgs = {
  experimentName: Scalars['String']['input'];
  functionID: Scalars['ID']['input'];
  workspaceID: Scalars['ID']['input'];
};


export type QueryExperimentsArgs = {
  timeRange?: InputMaybe<TimeRangeInput>;
  workspaceID: Scalars['ID']['input'];
};


export type QueryInsightsArgs = {
  query: Scalars['String']['input'];
  workspaceID: Scalars['ID']['input'];
};


export type QueryInsightsMetricArgs = {
  functionIDs?: InputMaybe<Array<Scalars['ID']['input']>>;
  key: Scalars['String']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<InsightsMetricOrderBy>;
  range: TimeRangeInput;
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


export type QueryScoreNamesArgs = {
  filter: ScoreFilter;
  functionIDs?: InputMaybe<Array<Scalars['ID']['input']>>;
  workspaceID: Scalars['ID']['input'];
};


export type QueryScoreTimeSeriesArgs = {
  bucketSeconds?: InputMaybe<Scalars['Int']['input']>;
  filter: ScoreFilter;
  functionIDs?: InputMaybe<Array<Scalars['ID']['input']>>;
  scoreNames?: InputMaybe<Array<Scalars['String']['input']>>;
  workspaceID: Scalars['ID']['input'];
};


export type QuerySessionRunsArgs = {
  sessionId: Scalars['String']['input'];
  sessionKey: Scalars['String']['input'];
  timeRange?: InputMaybe<TimeRangeInput>;
  workspaceID: Scalars['ID']['input'];
};


export type QuerySessionsArgs = {
  sessionIdSearch?: InputMaybe<Scalars['String']['input']>;
  sessionKey: Scalars['String']['input'];
  timeRange?: InputMaybe<TimeRangeInput>;
  workspaceID: Scalars['ID']['input'];
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
  event?: Maybe<QuickSearchEvent>;
  eventTypes: Array<QuickSearchEventType>;
  functions: Array<QuickSearchFunction>;
  run?: Maybe<QuickSearchRun>;
};

export type QuickSearchRun = {
  __typename?: 'QuickSearchRun';
  envSlug: Scalars['String']['output'];
  id: Scalars['ULID']['output'];
};

export type RateLimitConfiguration = {
  __typename?: 'RateLimitConfiguration';
  key?: Maybe<Scalars['String']['output']>;
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
  endedAt?: Maybe<Scalars['Time']['output']>;
  /** Filters applied to the replay, such as specific run statuses. */
  filters?: Maybe<Scalars['JSON']['output']>;
  /** Structured filters applied to the replay. */
  filtersV2?: Maybe<ReplayFilters>;
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
  totalRunCount?: Maybe<Scalars['Int']['output']>;
  workflowID?: Maybe<Scalars['UUID']['output']>;
  workspaceID?: Maybe<Scalars['UUID']['output']>;
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
  isDefault?: Maybe<Scalars['Boolean']['output']>;
  value: Scalars['Int']['output'];
};

export type RunDefer = {
  __typename?: 'RunDefer';
  fnSlug: Scalars['String']['output'];
  function?: Maybe<Workflow>;
  hashedDeferID: Scalars['String']['output'];
  run?: Maybe<FunctionRunV2>;
  runID?: Maybe<Scalars['ULID']['output']>;
  status: RunDeferStatus;
  userlandDeferID: Scalars['String']['output'];
};

export enum RunDeferStatus {
  Aborted = 'ABORTED',
  Rejected = 'REJECTED',
  Scheduled = 'SCHEDULED'
}

export type RunDeferredFrom = {
  __typename?: 'RunDeferredFrom';
  function: Workflow;
  run?: Maybe<FunctionRunV2>;
  runID: Scalars['ULID']['output'];
};

export type RunTraceSpan = {
  __typename?: 'RunTraceSpan';
  account: Account;
  accountID: Scalars['UUID']['output'];
  appID: Scalars['UUID']['output'];
  attempts?: Maybe<Scalars['Int']['output']>;
  childrenSpans: Array<RunTraceSpan>;
  duration?: Maybe<Scalars['Int']['output']>;
  endedAt?: Maybe<Scalars['Time']['output']>;
  functionID: Scalars['UUID']['output'];
  groupID?: Maybe<Scalars['String']['output']>;
  isPreview?: Maybe<Scalars['Boolean']['output']>;
  isRoot: Scalars['Boolean']['output'];
  isUserland: Scalars['Boolean']['output'];
  metadata: Array<SpanMetadata>;
  name: Scalars['String']['output'];
  outputID?: Maybe<Scalars['String']['output']>;
  parentSpan?: Maybe<RunTraceSpan>;
  parentSpanID?: Maybe<Scalars['String']['output']>;
  queuedAt: Scalars['Time']['output'];
  response?: Maybe<RunTraceSpanResponseInfo>;
  run: FunctionRun;
  runID: Scalars['ULID']['output'];
  scheduledAt?: Maybe<Scalars['Time']['output']>;
  skipExistingRunID?: Maybe<Scalars['String']['output']>;
  skipReason?: Maybe<Scalars['String']['output']>;
  spanID: Scalars['String']['output'];
  startedAt?: Maybe<Scalars['Time']['output']>;
  status: RunTraceSpanStatus;
  stepID?: Maybe<Scalars['String']['output']>;
  stepInfo?: Maybe<StepInfo>;
  stepOp?: Maybe<StepOp>;
  stepType: Scalars['String']['output'];
  traceID: Scalars['String']['output'];
  userlandSpan?: Maybe<UserlandSpan>;
  workspace: Workspace;
  workspaceID: Scalars['UUID']['output'];
};

export type RunTraceSpanOutput = {
  __typename?: 'RunTraceSpanOutput';
  data?: Maybe<Scalars['Bytes']['output']>;
  error?: Maybe<StepError>;
  input?: Maybe<Scalars['Bytes']['output']>;
};

export type RunTraceSpanResponseInfo = {
  __typename?: 'RunTraceSpanResponseInfo';
  headers: Scalars['HTTPHeaders']['output'];
  statusCode: Scalars['Int']['output'];
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
  batchID?: Maybe<Scalars['ULID']['output']>;
  cron?: Maybe<Scalars['String']['output']>;
  eventName?: Maybe<Scalars['String']['output']>;
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
  tagName?: Maybe<Scalars['String']['output']>;
  tagValue?: Maybe<Scalars['String']['output']>;
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

export type Score = {
  __typename?: 'Score';
  kind: ScoreKind;
  name: Scalars['String']['output'];
};

export type ScoreBucket = {
  __typename?: 'ScoreBucket';
  avg?: Maybe<Scalars['Float']['output']>;
  bucketStart: Scalars['Time']['output'];
  falseCount?: Maybe<Scalars['Int']['output']>;
  max?: Maybe<Scalars['Float']['output']>;
  p50?: Maybe<Scalars['Float']['output']>;
  p90?: Maybe<Scalars['Float']['output']>;
  p99?: Maybe<Scalars['Float']['output']>;
  runCount: Scalars['Int']['output'];
  trueCount?: Maybe<Scalars['Int']['output']>;
};

export type ScoreFilter = {
  timeRange: TimeRangeInput;
};

export enum ScoreKind {
  Boolean = 'BOOLEAN',
  Numeric = 'NUMERIC'
}

export type ScoreSeries = {
  __typename?: 'ScoreSeries';
  bucketSeconds: Scalars['Int']['output'];
  buckets: Array<ScoreBucket>;
  kind: ScoreKind;
  scoreName: Scalars['String']['output'];
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
  expires?: Maybe<Scalars['Time']['output']>;
  user: User;
};

export type SessionFunction = {
  __typename?: 'SessionFunction';
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
};

export type SessionGroup = {
  __typename?: 'SessionGroup';
  failedRunCount: Scalars['Int']['output'];
  failureRate: Scalars['Float']['output'];
  functions: Array<SessionFunction>;
  lastActiveAt: Scalars['Time']['output'];
  runCount: Scalars['Int']['output'];
  sessionId: Scalars['String']['output'];
  sessionKey: Scalars['String']['output'];
};

export type SessionKey = {
  __typename?: 'SessionKey';
  createdAt: Scalars['Time']['output'];
  sessionKey: Scalars['String']['output'];
};

export type SessionRun = {
  __typename?: 'SessionRun';
  endedAt?: Maybe<Scalars['Time']['output']>;
  eventName?: Maybe<Scalars['String']['output']>;
  functionSlug: Scalars['String']['output'];
  id: Scalars['String']['output'];
  queuedAt: Scalars['Time']['output'];
  startedAt?: Maybe<Scalars['Time']['output']>;
  status: Scalars['String']['output'];
};

export type SetUpAccountPayload = {
  __typename?: 'SetUpAccountPayload';
  account?: Maybe<Account>;
};

export type SigningKey = {
  __typename?: 'SigningKey';
  createdAt: Scalars['Time']['output'];
  decryptedValue: Scalars['String']['output'];
  id: Scalars['UUID']['output'];
  isActive: Scalars['Boolean']['output'];
  user?: Maybe<User>;
};

export type SigningKeyRotationCheck = {
  __typename?: 'SigningKeyRotationCheck';
  sdkSupport: Scalars['Boolean']['output'];
  signingKeyFallbackState: SecretCheck;
  signingKeyState: SecretCheck;
};

export type SingletonConfiguration = {
  __typename?: 'SingletonConfiguration';
  key?: Maybe<Scalars['String']['output']>;
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
  batchID?: Maybe<Scalars['ULID']['output']>;
  eventID?: Maybe<Scalars['ULID']['output']>;
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
  cause?: Maybe<Scalars['String']['output']>;
  message: Scalars['String']['output'];
  name?: Maybe<Scalars['String']['output']>;
  stack?: Maybe<Scalars['String']['output']>;
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
  app?: Maybe<App>;
  error?: Maybe<CodedError>;
  sync?: Maybe<Deploy>;
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
  key?: Maybe<Scalars['String']['output']>;
  limit: Scalars['Int']['output'];
  period: Scalars['String']['output'];
};

export type TimeRangeInput = {
  from: Scalars['Time']['input'];
  to: Scalars['Time']['input'];
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
  value?: Maybe<Scalars['Float']['output']>;
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

export type UpdateVercelAppResponse = {
  __typename?: 'UpdateVercelAppResponse';
  success: Scalars['Boolean']['output'];
  vercelApp?: Maybe<VercelApp>;
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
  account?: Maybe<Account>;
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  lastLoginAt?: Maybe<Scalars['Time']['output']>;
  name?: Maybe<Scalars['NullString']['output']>;
  passwordChangedAt?: Maybe<Scalars['Time']['output']>;
  roles?: Maybe<Array<Maybe<Scalars['Role']['output']>>>;
  updatedAt: Scalars['Time']['output'];
};

export type UserlandSpan = {
  __typename?: 'UserlandSpan';
  resourceAttrs?: Maybe<Scalars['Bytes']['output']>;
  scopeName?: Maybe<Scalars['String']['output']>;
  scopeVersion?: Maybe<Scalars['String']['output']>;
  serviceName?: Maybe<Scalars['String']['output']>;
  spanAttrs?: Maybe<Scalars['Bytes']['output']>;
  spanKind?: Maybe<Scalars['String']['output']>;
  spanName?: Maybe<Scalars['String']['output']>;
};

export type VariantMetric = {
  __typename?: 'VariantMetric';
  avg: Scalars['Float']['output'];
  key: Scalars['String']['output'];
  max: Scalars['Float']['output'];
  med: Scalars['Float']['output'];
  min: Scalars['Float']['output'];
  q1: Scalars['Float']['output'];
  q3: Scalars['Float']['output'];
  stddev: Scalars['Float']['output'];
};

export type VariantWeight = {
  __typename?: 'VariantWeight';
  variantName: Scalars['String']['output'];
  weight: Scalars['Float']['output'];
};

export type VercelApp = {
  __typename?: 'VercelApp';
  id: Scalars['UUID']['output'];
  originOverride?: Maybe<Scalars['String']['output']>;
  path?: Maybe<Scalars['String']['output']>;
  projectID: Scalars['String']['output'];
  protectionBypassSecret?: Maybe<Scalars['String']['output']>;
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
  originOverride?: Maybe<Scalars['String']['output']>;
  projectID: Scalars['String']['output'];
  protectionBypassSecret?: Maybe<Scalars['String']['output']>;
  servePath: Scalars['String']['output'];
};

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String']['output'];
  expression?: Maybe<Scalars['String']['output']>;
  foundEventID?: Maybe<Scalars['ULID']['output']>;
  timedOut?: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
};

export type Workflow = {
  __typename?: 'Workflow';
  app: App;
  archivedAt?: Maybe<Scalars['Time']['output']>;
  cancellationRunCount: Scalars['Int']['output'];
  cancellations: CancellationConnection;
  configuration?: Maybe<FunctionConfiguration>;
  current?: Maybe<WorkflowVersion>;
  failureHandler?: Maybe<Workflow>;
  id: Scalars['ID']['output'];
  isArchived: Scalars['Boolean']['output'];
  isParentArchived: Scalars['Boolean']['output'];
  isPaused: Scalars['Boolean']['output'];
  keyQueuesEnabled: Scalars['Boolean']['output'];
  latestVersion?: Maybe<WorkflowVersion>;
  metrics: MetricsResponse;
  name: Scalars['String']['output'];
  previous: Array<Maybe<WorkflowVersion>>;
  /** Lists the estimated number of runs to replay */
  replayCounts: ReplayRunCounts;
  /**
   * A list of all the function's replays.
   *
   * This doesn't include environment-level replays.
   */
  replays: Array<Replay>;
  slug: Scalars['String']['output'];
  triggers: Array<FunctionTrigger>;
  url: Scalars['String']['output'];
  usage: Usage;
};


export type WorkflowCancellationRunCountArgs = {
  input: CancellationRunCountInput;
};


export type WorkflowCancellationsArgs = {
  after?: InputMaybe<Scalars['String']['input']>;
  first?: Scalars['Int']['input'];
};


export type WorkflowMetricsArgs = {
  opts: MetricsRequest;
};


export type WorkflowReplayCountsArgs = {
  from: Scalars['Time']['input'];
  to: Scalars['Time']['input'];
};


export type WorkflowUsageArgs = {
  event?: InputMaybe<Scalars['String']['input']>;
  opts?: InputMaybe<UsageInput>;
};

export type WorkflowResponse = {
  __typename?: 'WorkflowResponse';
  workflow: Workflow;
};

export type WorkflowVersion = {
  __typename?: 'WorkflowVersion';
  createdAt: Scalars['Time']['output'];
  deploy?: Maybe<Deploy>;
  description?: Maybe<Scalars['NullString']['output']>;
  retries: Scalars['Int']['output'];
  throttleCount: Scalars['Int']['output'];
  throttlePeriod: Scalars['String']['output'];
  triggers: Array<FunctionTrigger>;
  updatedAt: Scalars['Time']['output'];
  url: Scalars['String']['output'];
  validFrom?: Maybe<Scalars['Time']['output']>;
  validTo?: Maybe<Scalars['Time']['output']>;
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
  archivedEvent?: Maybe<ArchivedEvent>;
  cdcConnections: Array<CdcConnection>;
  connectWorkerMetrics: ScopedMetricsResponse;
  createdAt: Scalars['Time']['output'];
  event?: Maybe<Event>;
  eventByNames: Array<EventType>;
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
  lastDeployedAt?: Maybe<Scalars['Time']['output']>;
  name: Scalars['String']['output'];
  parentID?: Maybe<Scalars['ID']['output']>;
  replay: Replay;
  run?: Maybe<FunctionRunV2>;
  runTraceSpanOutputByID: RunTraceSpanOutput;
  runTrigger: RunTraceTrigger;
  runs: RunsConnection;
  scopedFunctionStatus: ScopedFunctionStatusResponse;
  scopedMetrics: ScopedMetricsResponse;
  sessionKeys: Array<SessionKey>;
  signingKeys: Array<SigningKey>;
  slug: Scalars['String']['output'];
  test: Scalars['Boolean']['output'];
  type: EnvironmentType;
  unattachedSyncs: Array<Deploy>;
  vercelApps: Array<VercelApp>;
  webhookSigningKey: Scalars['String']['output'];
  workerConnection?: Maybe<ConnectV1WorkerConnection>;
  workerConnections: ConnectV1WorkerConnectionsConnection;
  workflow?: Maybe<Workflow>;
  workflowBySlug?: Maybe<Workflow>;
  workflows: PaginatedWorkflows;
};


export type WorkspaceAppByExternalIdArgs = {
  externalID: Scalars['String']['input'];
};


export type WorkspaceAppCheckArgs = {
  url: Scalars['String']['input'];
};


export type WorkspaceAppsArgs = {
  filter?: InputMaybe<AppsFilter>;
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


export type WorkspaceEventTypeArgs = {
  name: Scalars['String']['input'];
};


export type WorkspaceEventTypesV2Args = {
  after?: InputMaybe<Scalars['String']['input']>;
  filter: EventTypesFilter;
  first?: Scalars['Int']['input'];
};


export type WorkspaceEventV2Args = {
  id: Scalars['ULID']['input'];
};


export type WorkspaceEventsArgs = {
  prefix?: InputMaybe<Scalars['String']['input']>;
};


export type WorkspaceEventsV2Args = {
  after?: InputMaybe<Scalars['String']['input']>;
  filter: EventsFilter;
  first?: Scalars['Int']['input'];
};


export type WorkspaceIngestKeyArgs = {
  id: Scalars['ID']['input'];
};


export type WorkspaceIngestKeysArgs = {
  filter?: InputMaybe<IngestKeyFilter>;
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
  after?: InputMaybe<Scalars['String']['input']>;
  filter: RunsFilterV2;
  first?: Scalars['Int']['input'];
  orderBy: Array<RunsOrderBy>;
  preview?: InputMaybe<Scalars['Boolean']['input']>;
};


export type WorkspaceScopedFunctionStatusArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceScopedMetricsArgs = {
  filter: ScopedMetricsFilter;
};


export type WorkspaceSessionKeysArgs = {
  search?: InputMaybe<Scalars['String']['input']>;
};


export type WorkspaceUnattachedSyncsArgs = {
  after?: InputMaybe<Scalars['Time']['input']>;
  first?: Scalars['Int']['input'];
};


export type WorkspaceWorkerConnectionArgs = {
  connectionId: Scalars['ULID']['input'];
};


export type WorkspaceWorkerConnectionsArgs = {
  after?: InputMaybe<Scalars['String']['input']>;
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
  search?: InputMaybe<Scalars['String']['input']>;
};

export type GetEnvironmentBySlugQueryVariables = Exact<{
  slug: Scalars['String']['input'];
}>;


export type GetEnvironmentBySlugQuery = { __typename?: 'Query', envBySlug?: { __typename?: 'Workspace', id: string, name: string, slug: string, parentID?: string | null, test: boolean, type: EnvironmentType, createdAt: string, lastDeployedAt?: string | null, isArchived: boolean, isAutoArchiveEnabled: boolean, webhookSigningKey: string } | null };


export const GetEnvironmentBySlugDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEnvironmentBySlug"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"envBySlug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"parentID"}},{"kind":"Field","name":{"kind":"Name","value":"test"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastDeployedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isArchived"}},{"kind":"Field","name":{"kind":"Name","value":"isAutoArchiveEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"webhookSigningKey"}}]}}]}}]} as unknown as DocumentNode<GetEnvironmentBySlugQuery, GetEnvironmentBySlugQueryVariables>;