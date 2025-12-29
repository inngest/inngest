import { api } from './baseApi';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
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
  Bytes: { input: any; output: any; }
  /** The environment for the function to be run: `"prod"` or `"test"` */
  Environment: { input: any; output: any; }
  Int64: { input: number; output: number; }
  Map: { input: any; output: any; }
  SpanMetadataKind: { input: any; output: any; }
  SpanMetadataScope: { input: any; output: any; }
  SpanMetadataValues: { input: any; output: any; }
  Time: { input: any; output: any; }
  ULID: { input: any; output: any; }
  UUID: { input: any; output: any; }
  Uint: { input: any; output: any; }
  Unknown: { input: any; output: any; }
};

export type ActionVersionQuery = {
  dsn: Scalars['String']['input'];
  versionMajor?: InputMaybe<Scalars['Int']['input']>;
  versionMinor?: InputMaybe<Scalars['Int']['input']>;
};

export type App = {
  __typename?: 'App';
  appVersion: Maybe<Scalars['String']['output']>;
  autodiscovered: Scalars['Boolean']['output'];
  checksum: Maybe<Scalars['String']['output']>;
  connected: Scalars['Boolean']['output'];
  error: Maybe<Scalars['String']['output']>;
  externalID: Scalars['String']['output'];
  framework: Maybe<Scalars['String']['output']>;
  functionCount: Scalars['Int']['output'];
  functions: Array<Function>;
  id: Scalars['ID']['output'];
  method: AppMethod;
  name: Scalars['String']['output'];
  sdkLanguage: Scalars['String']['output'];
  sdkVersion: Scalars['String']['output'];
  url: Maybe<Scalars['String']['output']>;
};

export enum AppMethod {
  Api = 'API',
  Connect = 'CONNECT',
  Serve = 'SERVE'
}

export type AppsFilterV1 = {
  method?: InputMaybe<AppMethod>;
};

export type CancellationConfiguration = {
  __typename?: 'CancellationConfiguration';
  condition: Maybe<Scalars['String']['output']>;
  event: Scalars['String']['output'];
  timeout: Maybe<Scalars['String']['output']>;
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
  disconnectReason: Maybe<Scalars['String']['output']>;
  disconnectedAt: Maybe<Scalars['Time']['output']>;
  functionCount: Scalars['Int']['output'];
  gatewayId: Scalars['ULID']['output'];
  groupHash: Scalars['String']['output'];
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
  /** @deprecated buildId is deprecated. Use appVersion instead. */
  syncId: Maybe<Scalars['UUID']['output']>;
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

export type CreateAppInput = {
  url: Scalars['String']['input'];
};

export type CreateDebugSessionInput = {
  functionSlug: Scalars['String']['input'];
  runID?: InputMaybe<Scalars['String']['input']>;
  workspaceId?: Scalars['ID']['input'];
};

export type CreateDebugSessionResponse = {
  __typename?: 'CreateDebugSessionResponse';
  debugRunID: Scalars['ULID']['output'];
  debugSessionID: Scalars['ULID']['output'];
};

export type DebounceConfiguration = {
  __typename?: 'DebounceConfiguration';
  key: Maybe<Scalars['String']['output']>;
  period: Scalars['String']['output'];
};

export type DebugRun = {
  __typename?: 'DebugRun';
  debugTraces: Maybe<Array<RunTraceSpan>>;
};

export type DebugRunQuery = {
  debugRunID?: InputMaybe<Scalars['String']['input']>;
  functionSlug: Scalars['String']['input'];
  runID?: InputMaybe<Scalars['String']['input']>;
  workspaceId?: Scalars['ID']['input'];
};

export type DebugSession = {
  __typename?: 'DebugSession';
  debugRuns: Maybe<Array<DebugSessionRun>>;
};

export type DebugSessionQuery = {
  debugSessionID?: InputMaybe<Scalars['String']['input']>;
  functionSlug: Scalars['String']['input'];
  runID?: InputMaybe<Scalars['String']['input']>;
  workspaceId?: Scalars['ID']['input'];
};

export type DebugSessionRun = {
  __typename?: 'DebugSessionRun';
  debugRunID: Maybe<Scalars['ULID']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  queuedAt: Scalars['Time']['output'];
  startedAt: Maybe<Scalars['Time']['output']>;
  status: RunTraceSpanStatus;
  tags: Maybe<Array<Scalars['String']['output']>>;
  versions: Maybe<Array<Scalars['String']['output']>>;
};

export type Event = {
  __typename?: 'Event';
  createdAt: Maybe<Scalars['Time']['output']>;
  externalID: Maybe<Scalars['String']['output']>;
  functionRuns: Maybe<Array<FunctionRun>>;
  id: Scalars['ULID']['output'];
  name: Maybe<Scalars['String']['output']>;
  payload: Maybe<Scalars['String']['output']>;
  pendingRuns: Maybe<Scalars['Int']['output']>;
  raw: Maybe<Scalars['String']['output']>;
  schema: Maybe<Scalars['String']['output']>;
  status: Maybe<EventStatus>;
  totalRuns: Maybe<Scalars['Int']['output']>;
  workspace: Maybe<Workspace>;
};

export type EventQuery = {
  eventId: Scalars['ID']['input'];
  workspaceId?: Scalars['ID']['input'];
};

export type EventSource = {
  __typename?: 'EventSource';
  id: Scalars['ID']['output'];
  name: Maybe<Scalars['String']['output']>;
  sourceKind: Scalars['String']['output'];
};

export enum EventStatus {
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  NoFunctions = 'NO_FUNCTIONS',
  PartiallyFailed = 'PARTIALLY_FAILED',
  Paused = 'PAUSED',
  Running = 'RUNNING'
}

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

export type EventsQuery = {
  lastEventId?: InputMaybe<Scalars['ID']['input']>;
  workspaceId?: Scalars['ID']['input'];
};

export type Function = {
  __typename?: 'Function';
  app: App;
  appID: Scalars['String']['output'];
  concurrency: Scalars['Int']['output'];
  config: Scalars['String']['output'];
  configuration: FunctionConfiguration;
  failureHandler: Maybe<Function>;
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
  triggers: Maybe<Array<FunctionTrigger>>;
  url: Scalars['String']['output'];
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

export type FunctionEvent = {
  __typename?: 'FunctionEvent';
  createdAt: Maybe<Scalars['Time']['output']>;
  functionRun: Maybe<FunctionRun>;
  output: Maybe<Scalars['String']['output']>;
  type: Maybe<FunctionEventType>;
  workspace: Maybe<Workspace>;
};

export enum FunctionEventType {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Started = 'STARTED'
}

export type FunctionQuery = {
  functionSlug: Scalars['String']['input'];
  workspaceId?: Scalars['ID']['input'];
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  batchCreatedAt: Maybe<Scalars['Time']['output']>;
  batchID: Maybe<Scalars['ULID']['output']>;
  cron: Maybe<Scalars['String']['output']>;
  event: Maybe<Event>;
  eventID: Scalars['ID']['output'];
  events: Array<Event>;
  finishedAt: Maybe<Scalars['Time']['output']>;
  function: Maybe<Function>;
  functionID: Scalars['String']['output'];
  history: Array<RunHistoryItem>;
  historyItemOutput: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  output: Maybe<Scalars['String']['output']>;
  /** @deprecated Field no longer supported */
  pendingSteps: Maybe<Scalars['Int']['output']>;
  startedAt: Maybe<Scalars['Time']['output']>;
  status: Maybe<FunctionRunStatus>;
  waitingFor: Maybe<StepEventWait>;
  workspace: Maybe<Workspace>;
};


export type FunctionRunHistoryItemOutputArgs = {
  id: Scalars['ULID']['input'];
};

export type FunctionRunEvent = FunctionEvent | StepEvent;

export type FunctionRunQuery = {
  functionRunId: Scalars['ID']['input'];
  workspaceId?: Scalars['ID']['input'];
};

export enum FunctionRunStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Queued = 'QUEUED',
  Running = 'RUNNING'
}

export type FunctionRunV2 = {
  __typename?: 'FunctionRunV2';
  app: App;
  appID: Scalars['UUID']['output'];
  batchCreatedAt: Maybe<Scalars['Time']['output']>;
  cronSchedule: Maybe<Scalars['String']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  eventName: Maybe<Scalars['String']['output']>;
  function: Function;
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
};


export type FunctionRunV2TraceArgs = {
  preview: InputMaybe<Scalars['Boolean']['input']>;
};

export type FunctionRunV2Edge = {
  __typename?: 'FunctionRunV2Edge';
  cursor: Scalars['String']['output'];
  node: FunctionRunV2;
};

export type FunctionRunsQuery = {
  workspaceId?: Scalars['ID']['input'];
};

export enum FunctionStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Running = 'RUNNING'
}

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

export type FunctionVersion = {
  __typename?: 'FunctionVersion';
  config: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  functionId: Scalars['ID']['output'];
  updatedAt: Scalars['Time']['output'];
  validFrom: Maybe<Scalars['Time']['output']>;
  validTo: Maybe<Scalars['Time']['output']>;
  version: Scalars['Uint']['output'];
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

export type InvokeStepInfo = {
  __typename?: 'InvokeStepInfo';
  functionID: Scalars['String']['output'];
  returnEventID: Maybe<Scalars['ULID']['output']>;
  runID: Maybe<Scalars['ULID']['output']>;
  timedOut: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
  triggeringEventID: Scalars['ULID']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  cancelRun: FunctionRun;
  createApp: App;
  createDebugSession: CreateDebugSessionResponse;
  deleteApp: Scalars['String']['output'];
  deleteAppByName: Scalars['Boolean']['output'];
  invokeFunction: Maybe<Scalars['Boolean']['output']>;
  rerun: Scalars['ULID']['output'];
  updateApp: App;
};


export type MutationCancelRunArgs = {
  runID: Scalars['ULID']['input'];
};


export type MutationCreateAppArgs = {
  input: CreateAppInput;
};


export type MutationCreateDebugSessionArgs = {
  input: CreateDebugSessionInput;
};


export type MutationDeleteAppArgs = {
  id: Scalars['String']['input'];
};


export type MutationDeleteAppByNameArgs = {
  name: Scalars['String']['input'];
};


export type MutationInvokeFunctionArgs = {
  data: InputMaybe<Scalars['Map']['input']>;
  debugRunID: InputMaybe<Scalars['ULID']['input']>;
  debugSessionID: InputMaybe<Scalars['ULID']['input']>;
  functionSlug: Scalars['String']['input'];
  user: InputMaybe<Scalars['Map']['input']>;
};


export type MutationRerunArgs = {
  debugRunID: InputMaybe<Scalars['ULID']['input']>;
  debugSessionID: InputMaybe<Scalars['ULID']['input']>;
  fromStep: InputMaybe<RerunFromStepInput>;
  runID: Scalars['ULID']['input'];
};


export type MutationUpdateAppArgs = {
  input: UpdateAppInput;
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

export type Query = {
  __typename?: 'Query';
  app: Maybe<App>;
  apps: Array<App>;
  debugRun: Maybe<DebugRun>;
  debugSession: Maybe<DebugSession>;
  event: Maybe<Event>;
  eventV2: EventV2;
  events: Maybe<Array<Event>>;
  eventsV2: EventsConnection;
  functionBySlug: Maybe<Function>;
  functionRun: Maybe<FunctionRun>;
  functions: Maybe<Array<Function>>;
  run: Maybe<FunctionRunV2>;
  runTrace: RunTraceSpan;
  runTraceSpanOutputByID: RunTraceSpanOutput;
  runTrigger: RunTraceTrigger;
  runs: RunsV2Connection;
  stream: Array<StreamItem>;
  workerConnection: Maybe<ConnectV1WorkerConnection>;
  workerConnections: ConnectV1WorkerConnectionsConnection;
};


export type QueryAppArgs = {
  id: Scalars['UUID']['input'];
};


export type QueryAppsArgs = {
  filter: InputMaybe<AppsFilterV1>;
};


export type QueryDebugRunArgs = {
  query: DebugRunQuery;
};


export type QueryDebugSessionArgs = {
  query: DebugSessionQuery;
};


export type QueryEventArgs = {
  query: EventQuery;
};


export type QueryEventV2Args = {
  id: Scalars['ULID']['input'];
};


export type QueryEventsArgs = {
  query: EventsQuery;
};


export type QueryEventsV2Args = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: EventsFilter;
  first?: Scalars['Int']['input'];
};


export type QueryFunctionBySlugArgs = {
  query: FunctionQuery;
};


export type QueryFunctionRunArgs = {
  query: FunctionRunQuery;
};


export type QueryRunArgs = {
  runID: Scalars['String']['input'];
};


export type QueryRunTraceArgs = {
  runID: Scalars['String']['input'];
};


export type QueryRunTraceSpanOutputByIdArgs = {
  outputID: Scalars['String']['input'];
};


export type QueryRunTriggerArgs = {
  runID: Scalars['String']['input'];
};


export type QueryRunsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: RunsFilterV2;
  first?: Scalars['Int']['input'];
  orderBy: Array<RunsV2OrderBy>;
  preview: InputMaybe<Scalars['Boolean']['input']>;
};


export type QueryStreamArgs = {
  query: StreamQuery;
};


export type QueryWorkerConnectionArgs = {
  connectionId: Scalars['ULID']['input'];
};


export type QueryWorkerConnectionsArgs = {
  after: InputMaybe<Scalars['String']['input']>;
  filter: ConnectV1WorkerConnectionsFilter;
  first?: Scalars['Int']['input'];
  orderBy: Array<ConnectV1WorkerConnectionsOrderBy>;
};

export type RateLimitConfiguration = {
  __typename?: 'RateLimitConfiguration';
  key: Maybe<Scalars['String']['output']>;
  limit: Scalars['Int']['output'];
  period: Scalars['String']['output'];
};

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

export type RunStep = {
  __typename?: 'RunStep';
  name: Scalars['String']['output'];
  stepID: Scalars['String']['output'];
  stepOp: Maybe<StepOp>;
};

export type RunStepInfo = {
  __typename?: 'RunStepInfo';
  type: Maybe<Scalars['String']['output']>;
};

export type RunTraceSpan = {
  __typename?: 'RunTraceSpan';
  appID: Scalars['UUID']['output'];
  attempts: Maybe<Scalars['Int']['output']>;
  childrenSpans: Array<RunTraceSpan>;
  debugPaused: Scalars['Boolean']['output'];
  debugRunID: Maybe<Scalars['ULID']['output']>;
  debugSessionID: Maybe<Scalars['ULID']['output']>;
  duration: Maybe<Scalars['Int']['output']>;
  endedAt: Maybe<Scalars['Time']['output']>;
  functionID: Scalars['UUID']['output'];
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
  Queued = 'QUEUED',
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

export type RunsFilterV2 = {
  appIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  from: Scalars['Time']['input'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']['input']>>;
  query?: InputMaybe<Scalars['String']['input']>;
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<RunsV2OrderByField>;
  until?: InputMaybe<Scalars['Time']['input']>;
};

export enum RunsOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type RunsV2Connection = {
  __typename?: 'RunsV2Connection';
  edges: Array<FunctionRunV2Edge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int']['output'];
};


export type RunsV2ConnectionTotalCountArgs = {
  preview: InputMaybe<Scalars['Boolean']['input']>;
};

export type RunsV2OrderBy = {
  direction: RunsOrderByDirection;
  field: RunsV2OrderByField;
};

export enum RunsV2OrderByField {
  EndedAt = 'ENDED_AT',
  QueuedAt = 'QUEUED_AT',
  StartedAt = 'STARTED_AT'
}

export type SingletonConfiguration = {
  __typename?: 'SingletonConfiguration';
  key: Maybe<Scalars['String']['output']>;
  mode: SingletonMode;
};

export enum SingletonMode {
  Cancel = 'CANCEL',
  Skip = 'SKIP'
}

export type SleepStepInfo = {
  __typename?: 'SleepStepInfo';
  sleepUntil: Scalars['Time']['output'];
};

export type SpanMetadata = {
  __typename?: 'SpanMetadata';
  kind: Scalars['SpanMetadataKind']['output'];
  scope: Scalars['SpanMetadataScope']['output'];
  values: Scalars['SpanMetadataValues']['output'];
};

export type StepError = {
  __typename?: 'StepError';
  cause: Maybe<Scalars['Unknown']['output']>;
  message: Scalars['String']['output'];
  name: Maybe<Scalars['String']['output']>;
  stack: Maybe<Scalars['String']['output']>;
};

export type StepEvent = {
  __typename?: 'StepEvent';
  createdAt: Maybe<Scalars['Time']['output']>;
  functionRun: Maybe<FunctionRun>;
  name: Maybe<Scalars['String']['output']>;
  output: Maybe<Scalars['String']['output']>;
  stepID: Maybe<Scalars['String']['output']>;
  type: Maybe<StepEventType>;
  waitingFor: Maybe<StepEventWait>;
  workspace: Maybe<Workspace>;
};

export enum StepEventType {
  Completed = 'COMPLETED',
  Errored = 'ERRORED',
  Failed = 'FAILED',
  Scheduled = 'SCHEDULED',
  Started = 'STARTED',
  Waiting = 'WAITING'
}

export type StepEventWait = {
  __typename?: 'StepEventWait';
  eventName: Maybe<Scalars['String']['output']>;
  expiryTime: Scalars['Time']['output'];
  expression: Maybe<Scalars['String']['output']>;
};

export type StepInfo = InvokeStepInfo | RunStepInfo | SleepStepInfo | WaitForEventStepInfo | WaitForSignalStepInfo;

export enum StepOp {
  AiGateway = 'AI_GATEWAY',
  Invoke = 'INVOKE',
  Run = 'RUN',
  Sleep = 'SLEEP',
  WaitForEvent = 'WAIT_FOR_EVENT',
  WaitForSignal = 'WAIT_FOR_SIGNAL'
}

export type StreamItem = {
  __typename?: 'StreamItem';
  createdAt: Scalars['Time']['output'];
  id: Scalars['ID']['output'];
  inBatch: Scalars['Boolean']['output'];
  runs: Maybe<Array<Maybe<FunctionRun>>>;
  trigger: Scalars['String']['output'];
  type: StreamType;
};

export type StreamQuery = {
  after?: InputMaybe<Scalars['ID']['input']>;
  before?: InputMaybe<Scalars['ID']['input']>;
  includeInternalEvents?: InputMaybe<Scalars['Boolean']['input']>;
  limit?: Scalars['Int']['input'];
};

export enum StreamType {
  Cron = 'CRON',
  Event = 'EVENT'
}

export type ThrottleConfiguration = {
  __typename?: 'ThrottleConfiguration';
  burst: Scalars['Int']['output'];
  key: Maybe<Scalars['String']['output']>;
  limit: Scalars['Int']['output'];
  period: Scalars['String']['output'];
};

export type UpdateAppInput = {
  id: Scalars['String']['input'];
  url: Scalars['String']['input'];
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

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String']['output'];
  expression: Maybe<Scalars['String']['output']>;
  foundEventID: Maybe<Scalars['ULID']['output']>;
  timedOut: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
};

export type WaitForSignalStepInfo = {
  __typename?: 'WaitForSignalStepInfo';
  signal: Scalars['String']['output'];
  timedOut: Maybe<Scalars['Boolean']['output']>;
  timeout: Scalars['Time']['output'];
};

export type Workspace = {
  __typename?: 'Workspace';
  id: Scalars['ID']['output'];
};

export type GetEventQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetEventQuery = { __typename?: 'Query', event: { __typename?: 'Event', id: any, name: string | null, createdAt: any | null, status: EventStatus | null, pendingRuns: number | null, raw: string | null, functionRuns: Array<{ __typename?: 'FunctionRun', id: string, status: FunctionRunStatus | null, startedAt: any | null, pendingSteps: number | null, output: string | null, function: { __typename?: 'Function', name: string } | null, waitingFor: { __typename?: 'StepEventWait', expiryTime: any, eventName: string | null, expression: string | null } | null }> | null } | null };

export type GetFunctionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetFunctionsQuery = { __typename?: 'Query', functions: Array<{ __typename?: 'Function', id: string, slug: string, name: string, url: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null, app: { __typename?: 'App', name: string } }> | null };

export type GetFunctionQueryVariables = Exact<{
  functionSlug: Scalars['String'];
}>;


export type GetFunctionQuery = { __typename?: 'Query', functionBySlug: { __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string, failureHandler: { __typename?: 'Function', slug: string } | null, configuration: { __typename?: 'FunctionConfiguration', priority: string | null, cancellations: Array<{ __typename?: 'CancellationConfiguration', event: string, timeout: string | null, condition: string | null }>, retries: { __typename?: 'RetryConfiguration', value: number, isDefault: boolean | null }, eventsBatch: { __typename?: 'EventsBatchConfiguration', maxSize: number, timeout: string, key: string | null } | null, concurrency: Array<{ __typename?: 'ConcurrencyConfiguration', scope: ConcurrencyScope, key: string | null, limit: { __typename?: 'ConcurrencyLimitConfiguration', value: number, isPlanLimit: boolean | null } }>, rateLimit: { __typename?: 'RateLimitConfiguration', limit: number, period: string, key: string | null } | null, debounce: { __typename?: 'DebounceConfiguration', period: string, key: string | null } | null, throttle: { __typename?: 'ThrottleConfiguration', burst: number, key: string | null, limit: number, period: string } | null, singleton: { __typename?: 'SingletonConfiguration', key: string | null, mode: SingletonMode } | null }, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string, condition: string | null }> | null, app: { __typename?: 'App', name: string } } | null };

export type GetAppsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAppsQuery = { __typename?: 'Query', apps: Array<{ __typename?: 'App', id: string, name: string, appVersion: string | null, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: AppMethod, functions: Array<{ __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string }> }> };

export type GetAppQueryVariables = Exact<{
  id: Scalars['UUID'];
}>;


export type GetAppQuery = { __typename?: 'Query', app: { __typename?: 'App', id: string, name: string, appVersion: string | null, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: AppMethod, functions: Array<{ __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null }> } | null };

export type CreateAppMutationVariables = Exact<{
  input: CreateAppInput;
}>;


export type CreateAppMutation = { __typename?: 'Mutation', createApp: { __typename?: 'App', url: string | null } };

export type UpdateAppMutationVariables = Exact<{
  input: UpdateAppInput;
}>;


export type UpdateAppMutation = { __typename?: 'Mutation', updateApp: { __typename?: 'App', url: string | null, id: string } };

export type DeleteAppMutationVariables = Exact<{
  id: Scalars['String'];
}>;


export type DeleteAppMutation = { __typename?: 'Mutation', deleteApp: string };

export type InvokeFunctionMutationVariables = Exact<{
  functionSlug: Scalars['String'];
  data: InputMaybe<Scalars['Map']>;
  user: InputMaybe<Scalars['Map']>;
  debugSessionID?: InputMaybe<Scalars['ULID']>;
  debugRunID?: InputMaybe<Scalars['ULID']>;
}>;


export type InvokeFunctionMutation = { __typename?: 'Mutation', invokeFunction: boolean | null };

export type CancelRunMutationVariables = Exact<{
  runID: Scalars['ULID'];
}>;


export type CancelRunMutation = { __typename?: 'Mutation', cancelRun: { __typename?: 'FunctionRun', id: string } };

export type RerunMutationVariables = Exact<{
  runID: Scalars['ULID'];
  debugRunID?: InputMaybe<Scalars['ULID']>;
  debugSessionID?: InputMaybe<Scalars['ULID']>;
}>;


export type RerunMutation = { __typename?: 'Mutation', rerun: any };

export type RerunFromStepMutationVariables = Exact<{
  runID: Scalars['ULID'];
  fromStep: RerunFromStepInput;
  debugRunID?: InputMaybe<Scalars['ULID']>;
  debugSessionID?: InputMaybe<Scalars['ULID']>;
}>;


export type RerunFromStepMutation = { __typename?: 'Mutation', rerun: any };

export type GetRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  startTime: Scalars['Time'];
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsV2OrderByField;
  functionRunCursor?: InputMaybe<Scalars['String']>;
  celQuery?: InputMaybe<Scalars['String']>;
  preview?: InputMaybe<Scalars['Boolean']>;
}>;


export type GetRunsQuery = { __typename?: 'Query', runs: { __typename?: 'RunsV2Connection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', cronSchedule: string | null, eventName: string | null, id: any, isBatch: boolean, queuedAt: any, endedAt: any | null, startedAt: any | null, status: FunctionRunStatus, hasAI: boolean, app: { __typename?: 'App', externalID: string, name: string }, function: { __typename?: 'Function', name: string, slug: string } } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountRunsQueryVariables = Exact<{
  startTime: Scalars['Time'];
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsV2OrderByField;
  preview?: InputMaybe<Scalars['Boolean']>;
}>;


export type CountRunsQuery = { __typename?: 'Query', runs: { __typename?: 'RunsV2Connection', totalCount: number } };

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null };

export type GetRunQueryVariables = Exact<{
  runID: Scalars['String'];
  preview: InputMaybe<Scalars['Boolean']>;
}>;


export type GetRunQuery = { __typename?: 'Query', run: { __typename?: 'FunctionRunV2', status: FunctionRunStatus, hasAI: boolean, function: { __typename?: 'Function', id: string, name: string, slug: string, app: { __typename?: 'App', name: string } }, trace: { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null } | null } | null };

export type GetRunTraceQueryVariables = Exact<{
  runID: Scalars['String'];
}>;


export type GetRunTraceQuery = { __typename?: 'Query', runTrace: { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null } };

export type GetTraceResultQueryVariables = Exact<{
  traceID: Scalars['String'];
}>;


export type GetTraceResultQuery = { __typename?: 'Query', runTraceSpanOutputByID: { __typename?: 'RunTraceSpanOutput', input: any | null, data: any | null, error: { __typename?: 'StepError', message: string, name: string | null, stack: string | null, cause: any | null } | null } };

export type GetTriggerQueryVariables = Exact<{
  runID: Scalars['String'];
}>;


export type GetTriggerQuery = { __typename?: 'Query', runTrigger: { __typename?: 'RunTraceTrigger', IDs: Array<any>, payloads: Array<any>, timestamp: any, eventName: string | null, isBatch: boolean, batchID: any | null, cron: string | null } };

export type GetWorkerConnectionsQueryVariables = Exact<{
  appID: Scalars['UUID'];
  startTime: InputMaybe<Scalars['Time']>;
  status: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
  timeField: ConnectV1WorkerConnectionsOrderByField;
  cursor?: InputMaybe<Scalars['String']>;
  orderBy?: InputMaybe<Array<ConnectV1WorkerConnectionsOrderBy> | ConnectV1WorkerConnectionsOrderBy>;
  first: Scalars['Int'];
}>;


export type GetWorkerConnectionsQuery = { __typename?: 'Query', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number, edges: Array<{ __typename?: 'ConnectV1WorkerConnectionEdge', node: { __typename?: 'ConnectV1WorkerConnection', id: any, gatewayId: any, instanceId: string, workerIp: string, maxWorkerConcurrency: number, connectedAt: any, lastHeartbeatAt: any | null, disconnectedAt: any | null, disconnectReason: string | null, status: ConnectV1ConnectionStatus, groupHash: string, sdkLang: string, sdkVersion: string, sdkPlatform: string, syncId: any | null, appVersion: string | null, functionCount: number, cpuCores: number, memBytes: number, os: string, app: { __typename?: 'App', id: string } | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountWorkerConnectionsQueryVariables = Exact<{
  appID: Scalars['UUID'];
  startTime: Scalars['Time'];
  status: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
}>;


export type CountWorkerConnectionsQuery = { __typename?: 'Query', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number } };

export type GetEventsV2QueryVariables = Exact<{
  cursor: InputMaybe<Scalars['String']>;
  startTime: Scalars['Time'];
  endTime: InputMaybe<Scalars['Time']>;
  celQuery?: InputMaybe<Scalars['String']>;
  eventNames?: InputMaybe<Array<Scalars['String']> | Scalars['String']>;
  includeInternalEvents?: InputMaybe<Scalars['Boolean']>;
}>;


export type GetEventsV2Query = { __typename?: 'Query', eventsV2: { __typename?: 'EventsConnection', totalCount: number, edges: Array<{ __typename?: 'EventsEdge', node: { __typename?: 'EventV2', name: string, id: any, receivedAt: any, runs: Array<{ __typename?: 'FunctionRunV2', status: FunctionRunStatus, id: any, startedAt: any | null, endedAt: any | null, function: { __typename?: 'Function', name: string, slug: string } }> } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, endCursor: string | null, hasPreviousPage: boolean, startCursor: string | null } } };

export type GetEventV2QueryVariables = Exact<{
  eventID: Scalars['ULID'];
}>;


export type GetEventV2Query = { __typename?: 'Query', eventV2: { __typename?: 'EventV2', name: string, id: any, receivedAt: any, idempotencyKey: string | null, occurredAt: any, version: string | null, source: { __typename?: 'EventSource', name: string | null } | null } };

export type GetEventV2PayloadQueryVariables = Exact<{
  eventID: Scalars['ULID'];
}>;


export type GetEventV2PayloadQuery = { __typename?: 'Query', eventV2: { __typename?: 'EventV2', raw: string } };

export type GetEventV2RunsQueryVariables = Exact<{
  eventID: Scalars['ULID'];
}>;


export type GetEventV2RunsQuery = { __typename?: 'Query', eventV2: { __typename?: 'EventV2', name: string, runs: Array<{ __typename?: 'FunctionRunV2', status: FunctionRunStatus, id: any, startedAt: any | null, endedAt: any | null, function: { __typename?: 'Function', name: string, slug: string } }> } };

export type CreateDebugSessionMutationVariables = Exact<{
  input: CreateDebugSessionInput;
}>;


export type CreateDebugSessionMutation = { __typename?: 'Mutation', createDebugSession: { __typename?: 'CreateDebugSessionResponse', debugSessionID: any, debugRunID: any } };

export type GetDebugRunQueryVariables = Exact<{
  query: DebugRunQuery;
}>;


export type GetDebugRunQuery = { __typename?: 'Query', debugRun: { __typename?: 'DebugRun', debugTraces: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: any | null, debugSessionID: any | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }> | null } | null };

export type GetDebugSessionQueryVariables = Exact<{
  query: DebugSessionQuery;
}>;


export type GetDebugSessionQuery = { __typename?: 'Query', debugSession: { __typename?: 'DebugSession', debugRuns: Array<{ __typename?: 'DebugSessionRun', status: RunTraceSpanStatus, queuedAt: any, startedAt: any | null, endedAt: any | null, debugRunID: any | null, tags: Array<string> | null, versions: Array<string> | null }> | null } | null };

export const TraceDetailsFragmentDoc = `
    fragment TraceDetails on RunTraceSpan {
  name
  status
  attempts
  queuedAt
  startedAt
  endedAt
  isRoot
  isUserland
  userlandSpan {
    spanName
    spanKind
    serviceName
    scopeName
    scopeVersion
    spanAttrs
    resourceAttrs
  }
  outputID
  debugRunID
  debugSessionID
  spanID
  stepID
  stepOp
  stepType
  stepInfo {
    __typename
    ... on InvokeStepInfo {
      triggeringEventID
      functionID
      timeout
      returnEventID
      runID
      timedOut
    }
    ... on SleepStepInfo {
      sleepUntil
    }
    ... on WaitForEventStepInfo {
      eventName
      expression
      timeout
      foundEventID
      timedOut
    }
    ... on RunStepInfo {
      type
    }
    ... on WaitForSignalStepInfo {
      signal
      timeout
      timedOut
    }
  }
}
    `;
export const GetEventDocument = `
    query GetEvent($id: ID!) {
  event(query: {eventId: $id}) {
    id
    name
    createdAt
    status
    pendingRuns
    raw
    functionRuns {
      function {
        name
      }
      id
      status
      startedAt
      pendingSteps
      output
      waitingFor {
        expiryTime
        eventName
        expression
      }
    }
  }
}
    `;
export const GetFunctionsDocument = `
    query GetFunctions {
  functions {
    id
    slug
    name
    triggers {
      type
      value
    }
    app {
      name
    }
    url
  }
}
    `;
export const GetFunctionDocument = `
    query GetFunction($functionSlug: String!) {
  functionBySlug(query: {functionSlug: $functionSlug}) {
    name
    id
    failureHandler {
      slug
    }
    concurrency
    config
    configuration {
      cancellations {
        event
        timeout
        condition
      }
      retries {
        value
        isDefault
      }
      priority
      eventsBatch {
        maxSize
        timeout
        key
      }
      concurrency {
        scope
        limit {
          value
          isPlanLimit
        }
        key
      }
      rateLimit {
        limit
        period
        key
      }
      debounce {
        period
        key
      }
      throttle {
        burst
        key
        limit
        period
      }
      singleton {
        key
        mode
      }
    }
    slug
    triggers {
      type
      value
      condition
    }
    app {
      name
    }
    url
  }
}
    `;
export const GetAppsDocument = `
    query GetApps {
  apps {
    id
    name
    appVersion
    sdkLanguage
    sdkVersion
    framework
    url
    error
    connected
    functionCount
    autodiscovered
    method
    functions {
      name
      id
      concurrency
      config
      slug
      url
    }
  }
}
    `;
export const GetAppDocument = `
    query GetApp($id: UUID!) {
  app(id: $id) {
    id
    name
    appVersion
    sdkLanguage
    sdkVersion
    framework
    url
    error
    connected
    functionCount
    autodiscovered
    method
    functions {
      name
      id
      concurrency
      config
      slug
      url
      triggers {
        type
        value
      }
    }
  }
}
    `;
export const CreateAppDocument = `
    mutation CreateApp($input: CreateAppInput!) {
  createApp(input: $input) {
    url
  }
}
    `;
export const UpdateAppDocument = `
    mutation UpdateApp($input: UpdateAppInput!) {
  updateApp(input: $input) {
    url
    id
  }
}
    `;
export const DeleteAppDocument = `
    mutation DeleteApp($id: String!) {
  deleteApp(id: $id)
}
    `;
export const InvokeFunctionDocument = `
    mutation InvokeFunction($functionSlug: String!, $data: Map, $user: Map, $debugSessionID: ULID = null, $debugRunID: ULID = null) {
  invokeFunction(
    data: $data
    functionSlug: $functionSlug
    user: $user
    debugSessionID: $debugSessionID
    debugRunID: $debugRunID
  )
}
    `;
export const CancelRunDocument = `
    mutation CancelRun($runID: ULID!) {
  cancelRun(runID: $runID) {
    id
  }
}
    `;
export const RerunDocument = `
    mutation Rerun($runID: ULID!, $debugRunID: ULID = null, $debugSessionID: ULID = null) {
  rerun(runID: $runID, debugRunID: $debugRunID, debugSessionID: $debugSessionID)
}
    `;
export const RerunFromStepDocument = `
    mutation RerunFromStep($runID: ULID!, $fromStep: RerunFromStepInput!, $debugRunID: ULID = null, $debugSessionID: ULID = null) {
  rerun(
    runID: $runID
    fromStep: $fromStep
    debugRunID: $debugRunID
    debugSessionID: $debugSessionID
  )
}
    `;
export const GetRunsDocument = `
    query GetRuns($appIDs: [UUID!], $startTime: Time!, $status: [FunctionRunStatus!], $timeField: RunsV2OrderByField!, $functionRunCursor: String = null, $celQuery: String = null, $preview: Boolean = false) {
  runs(
    filter: {appIDs: $appIDs, from: $startTime, status: $status, timeField: $timeField, query: $celQuery}
    orderBy: [{field: $timeField, direction: DESC}]
    after: $functionRunCursor
    preview: $preview
  ) {
    edges {
      node {
        app {
          externalID
          name
        }
        cronSchedule
        eventName
        function {
          name
          slug
        }
        id
        isBatch
        queuedAt
        endedAt
        startedAt
        status
        hasAI
      }
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
  }
}
    `;
export const CountRunsDocument = `
    query CountRuns($startTime: Time!, $status: [FunctionRunStatus!], $timeField: RunsV2OrderByField!, $preview: Boolean = false) {
  runs(
    filter: {from: $startTime, status: $status, timeField: $timeField}
    orderBy: [{field: $timeField, direction: DESC}]
    preview: $preview
  ) {
    totalCount(preview: $preview)
  }
}
    `;
export const GetRunDocument = `
    query GetRun($runID: String!, $preview: Boolean) {
  run(runID: $runID) {
    function {
      app {
        name
      }
      id
      name
      slug
    }
    status
    trace(preview: $preview) {
      ...TraceDetails
      childrenSpans {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
            }
          }
        }
      }
    }
    hasAI
  }
}
    ${TraceDetailsFragmentDoc}`;
export const GetRunTraceDocument = `
    query GetRunTrace($runID: String!) {
  runTrace(runID: $runID) {
    ...TraceDetails
    childrenSpans {
      ...TraceDetails
      childrenSpans {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
          }
        }
      }
    }
  }
}
    ${TraceDetailsFragmentDoc}`;
export const GetTraceResultDocument = `
    query GetTraceResult($traceID: String!) {
  runTraceSpanOutputByID(outputID: $traceID) {
    input
    data
    error {
      message
      name
      stack
      cause
    }
  }
}
    `;
export const GetTriggerDocument = `
    query GetTrigger($runID: String!) {
  runTrigger(runID: $runID) {
    IDs
    payloads
    timestamp
    eventName
    isBatch
    batchID
    cron
  }
}
    `;
export const GetWorkerConnectionsDocument = `
    query GetWorkerConnections($appID: UUID!, $startTime: Time, $status: [ConnectV1ConnectionStatus!], $timeField: ConnectV1WorkerConnectionsOrderByField!, $cursor: String = null, $orderBy: [ConnectV1WorkerConnectionsOrderBy!] = [], $first: Int!) {
  workerConnections(
    first: $first
    filter: {appIDs: [$appID], from: $startTime, status: $status, timeField: $timeField}
    orderBy: $orderBy
    after: $cursor
  ) {
    edges {
      node {
        id
        gatewayId
        instanceId
        workerIp
        maxWorkerConcurrency
        app {
          id
        }
        connectedAt
        lastHeartbeatAt
        disconnectedAt
        disconnectReason
        status
        groupHash
        sdkLang
        sdkVersion
        sdkPlatform
        syncId
        appVersion
        functionCount
        cpuCores
        memBytes
        os
      }
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
    totalCount
  }
}
    `;
export const CountWorkerConnectionsDocument = `
    query CountWorkerConnections($appID: UUID!, $startTime: Time!, $status: [ConnectV1ConnectionStatus!]) {
  workerConnections(
    filter: {appIDs: [$appID], from: $startTime, status: $status, timeField: CONNECTED_AT}
    orderBy: [{field: CONNECTED_AT, direction: DESC}]
  ) {
    totalCount
  }
}
    `;
export const GetEventsV2Document = `
    query GetEventsV2($cursor: String, $startTime: Time!, $endTime: Time, $celQuery: String = null, $eventNames: [String!] = null, $includeInternalEvents: Boolean = false) {
  eventsV2(
    first: 50
    after: $cursor
    filter: {from: $startTime, until: $endTime, query: $celQuery, eventNames: $eventNames, includeInternalEvents: $includeInternalEvents}
  ) {
    edges {
      node {
        name
        id
        receivedAt
        runs {
          status
          id
          startedAt
          endedAt
          function {
            name
            slug
          }
        }
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
      hasPreviousPage
      startCursor
    }
  }
}
    `;
export const GetEventV2Document = `
    query GetEventV2($eventID: ULID!) {
  eventV2(id: $eventID) {
    name
    id
    receivedAt
    idempotencyKey
    occurredAt
    version
    source {
      name
    }
  }
}
    `;
export const GetEventV2PayloadDocument = `
    query GetEventV2Payload($eventID: ULID!) {
  eventV2(id: $eventID) {
    raw
  }
}
    `;
export const GetEventV2RunsDocument = `
    query GetEventV2Runs($eventID: ULID!) {
  eventV2(id: $eventID) {
    name
    runs {
      status
      id
      startedAt
      endedAt
      function {
        name
        slug
      }
    }
  }
}
    `;
export const CreateDebugSessionDocument = `
    mutation CreateDebugSession($input: CreateDebugSessionInput!) {
  createDebugSession(input: $input) {
    debugSessionID
    debugRunID
  }
}
    `;
export const GetDebugRunDocument = `
    query GetDebugRun($query: DebugRunQuery!) {
  debugRun(query: $query) {
    debugTraces {
      ...TraceDetails
      childrenSpans {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
          }
        }
      }
    }
  }
}
    ${TraceDetailsFragmentDoc}`;
export const GetDebugSessionDocument = `
    query GetDebugSession($query: DebugSessionQuery!) {
  debugSession(query: $query) {
    debugRuns {
      status
      queuedAt
      startedAt
      endedAt
      debugRunID
      tags
      versions
    }
  }
}
    `;

const injectedRtkApi = api.injectEndpoints({
  endpoints: (build) => ({
    GetEvent: build.query<GetEventQuery, GetEventQueryVariables>({
      query: (variables) => ({ document: GetEventDocument, variables })
    }),
    GetFunctions: build.query<GetFunctionsQuery, GetFunctionsQueryVariables | void>({
      query: (variables) => ({ document: GetFunctionsDocument, variables })
    }),
    GetFunction: build.query<GetFunctionQuery, GetFunctionQueryVariables>({
      query: (variables) => ({ document: GetFunctionDocument, variables })
    }),
    GetApps: build.query<GetAppsQuery, GetAppsQueryVariables | void>({
      query: (variables) => ({ document: GetAppsDocument, variables })
    }),
    GetApp: build.query<GetAppQuery, GetAppQueryVariables>({
      query: (variables) => ({ document: GetAppDocument, variables })
    }),
    CreateApp: build.mutation<CreateAppMutation, CreateAppMutationVariables>({
      query: (variables) => ({ document: CreateAppDocument, variables })
    }),
    UpdateApp: build.mutation<UpdateAppMutation, UpdateAppMutationVariables>({
      query: (variables) => ({ document: UpdateAppDocument, variables })
    }),
    DeleteApp: build.mutation<DeleteAppMutation, DeleteAppMutationVariables>({
      query: (variables) => ({ document: DeleteAppDocument, variables })
    }),
    InvokeFunction: build.mutation<InvokeFunctionMutation, InvokeFunctionMutationVariables>({
      query: (variables) => ({ document: InvokeFunctionDocument, variables })
    }),
    CancelRun: build.mutation<CancelRunMutation, CancelRunMutationVariables>({
      query: (variables) => ({ document: CancelRunDocument, variables })
    }),
    Rerun: build.mutation<RerunMutation, RerunMutationVariables>({
      query: (variables) => ({ document: RerunDocument, variables })
    }),
    RerunFromStep: build.mutation<RerunFromStepMutation, RerunFromStepMutationVariables>({
      query: (variables) => ({ document: RerunFromStepDocument, variables })
    }),
    GetRuns: build.query<GetRunsQuery, GetRunsQueryVariables>({
      query: (variables) => ({ document: GetRunsDocument, variables })
    }),
    CountRuns: build.query<CountRunsQuery, CountRunsQueryVariables>({
      query: (variables) => ({ document: CountRunsDocument, variables })
    }),
    GetRun: build.query<GetRunQuery, GetRunQueryVariables>({
      query: (variables) => ({ document: GetRunDocument, variables })
    }),
    GetRunTrace: build.query<GetRunTraceQuery, GetRunTraceQueryVariables>({
      query: (variables) => ({ document: GetRunTraceDocument, variables })
    }),
    GetTraceResult: build.query<GetTraceResultQuery, GetTraceResultQueryVariables>({
      query: (variables) => ({ document: GetTraceResultDocument, variables })
    }),
    GetTrigger: build.query<GetTriggerQuery, GetTriggerQueryVariables>({
      query: (variables) => ({ document: GetTriggerDocument, variables })
    }),
    GetWorkerConnections: build.query<GetWorkerConnectionsQuery, GetWorkerConnectionsQueryVariables>({
      query: (variables) => ({ document: GetWorkerConnectionsDocument, variables })
    }),
    CountWorkerConnections: build.query<CountWorkerConnectionsQuery, CountWorkerConnectionsQueryVariables>({
      query: (variables) => ({ document: CountWorkerConnectionsDocument, variables })
    }),
    GetEventsV2: build.query<GetEventsV2Query, GetEventsV2QueryVariables>({
      query: (variables) => ({ document: GetEventsV2Document, variables })
    }),
    GetEventV2: build.query<GetEventV2Query, GetEventV2QueryVariables>({
      query: (variables) => ({ document: GetEventV2Document, variables })
    }),
    GetEventV2Payload: build.query<GetEventV2PayloadQuery, GetEventV2PayloadQueryVariables>({
      query: (variables) => ({ document: GetEventV2PayloadDocument, variables })
    }),
    GetEventV2Runs: build.query<GetEventV2RunsQuery, GetEventV2RunsQueryVariables>({
      query: (variables) => ({ document: GetEventV2RunsDocument, variables })
    }),
    CreateDebugSession: build.mutation<CreateDebugSessionMutation, CreateDebugSessionMutationVariables>({
      query: (variables) => ({ document: CreateDebugSessionDocument, variables })
    }),
    GetDebugRun: build.query<GetDebugRunQuery, GetDebugRunQueryVariables>({
      query: (variables) => ({ document: GetDebugRunDocument, variables })
    }),
    GetDebugSession: build.query<GetDebugSessionQuery, GetDebugSessionQueryVariables>({
      query: (variables) => ({ document: GetDebugSessionDocument, variables })
    }),
  }),
});

export { injectedRtkApi as api };
export const { useGetEventQuery, useLazyGetEventQuery, useGetFunctionsQuery, useLazyGetFunctionsQuery, useGetFunctionQuery, useLazyGetFunctionQuery, useGetAppsQuery, useLazyGetAppsQuery, useGetAppQuery, useLazyGetAppQuery, useCreateAppMutation, useUpdateAppMutation, useDeleteAppMutation, useInvokeFunctionMutation, useCancelRunMutation, useRerunMutation, useRerunFromStepMutation, useGetRunsQuery, useLazyGetRunsQuery, useCountRunsQuery, useLazyCountRunsQuery, useGetRunQuery, useLazyGetRunQuery, useGetRunTraceQuery, useLazyGetRunTraceQuery, useGetTraceResultQuery, useLazyGetTraceResultQuery, useGetTriggerQuery, useLazyGetTriggerQuery, useGetWorkerConnectionsQuery, useLazyGetWorkerConnectionsQuery, useCountWorkerConnectionsQuery, useLazyCountWorkerConnectionsQuery, useGetEventsV2Query, useLazyGetEventsV2Query, useGetEventV2Query, useLazyGetEventV2Query, useGetEventV2PayloadQuery, useLazyGetEventV2PayloadQuery, useGetEventV2RunsQuery, useLazyGetEventV2RunsQuery, useCreateDebugSessionMutation, useGetDebugRunQuery, useLazyGetDebugRunQuery, useGetDebugSessionQuery, useLazyGetDebugSessionQuery } = injectedRtkApi;

