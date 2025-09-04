import { api } from './baseApi';
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
  Bytes: any;
  /** The environment for the function to be run: `"prod"` or `"test"` */
  Environment: any;
  Map: any;
  Time: any;
  ULID: any;
  UUID: any;
  Uint: any;
  Unknown: any;
};

export type ActionVersionQuery = {
  dsn: Scalars['String'];
  versionMajor?: InputMaybe<Scalars['Int']>;
  versionMinor?: InputMaybe<Scalars['Int']>;
};

export type App = {
  __typename?: 'App';
  appVersion: Maybe<Scalars['String']>;
  autodiscovered: Scalars['Boolean'];
  checksum: Maybe<Scalars['String']>;
  connected: Scalars['Boolean'];
  error: Maybe<Scalars['String']>;
  externalID: Scalars['String'];
  framework: Maybe<Scalars['String']>;
  functionCount: Scalars['Int'];
  functions: Array<Function>;
  id: Scalars['ID'];
  method: AppMethod;
  name: Scalars['String'];
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  url: Maybe<Scalars['String']>;
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
  condition: Maybe<Scalars['String']>;
  event: Scalars['String'];
  timeout: Maybe<Scalars['String']>;
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
  appName: Maybe<Scalars['String']>;
  appVersion: Maybe<Scalars['String']>;
  buildId: Maybe<Scalars['String']>;
  connectedAt: Scalars['Time'];
  cpuCores: Scalars['Int'];
  disconnectReason: Maybe<Scalars['String']>;
  disconnectedAt: Maybe<Scalars['Time']>;
  functionCount: Scalars['Int'];
  gatewayId: Scalars['ULID'];
  groupHash: Scalars['String'];
  id: Scalars['ULID'];
  instanceId: Scalars['String'];
  lastHeartbeatAt: Maybe<Scalars['Time']>;
  memBytes: Scalars['Int'];
  os: Scalars['String'];
  sdkLang: Scalars['String'];
  sdkPlatform: Scalars['String'];
  sdkVersion: Scalars['String'];
  status: ConnectV1ConnectionStatus;
  /** @deprecated buildId is deprecated. Use appVersion instead. */
  syncId: Maybe<Scalars['UUID']>;
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
  from?: InputMaybe<Scalars['Time']>;
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

export type CreateAppInput = {
  url: Scalars['String'];
};

export type DebounceConfiguration = {
  __typename?: 'DebounceConfiguration';
  key: Maybe<Scalars['String']>;
  period: Scalars['String'];
};

export type Event = {
  __typename?: 'Event';
  createdAt: Maybe<Scalars['Time']>;
  externalID: Maybe<Scalars['String']>;
  functionRuns: Maybe<Array<FunctionRun>>;
  id: Scalars['ULID'];
  name: Maybe<Scalars['String']>;
  payload: Maybe<Scalars['String']>;
  pendingRuns: Maybe<Scalars['Int']>;
  raw: Maybe<Scalars['String']>;
  schema: Maybe<Scalars['String']>;
  status: Maybe<EventStatus>;
  totalRuns: Maybe<Scalars['Int']>;
  workspace: Maybe<Workspace>;
};

export type EventQuery = {
  eventId: Scalars['ID'];
  workspaceId?: Scalars['ID'];
};

export type EventSource = {
  __typename?: 'EventSource';
  id: Scalars['ID'];
  name: Maybe<Scalars['String']>;
  sourceKind: Scalars['String'];
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
  envID: Scalars['UUID'];
  id: Scalars['ULID'];
  idempotencyKey: Maybe<Scalars['String']>;
  name: Scalars['String'];
  occurredAt: Scalars['Time'];
  raw: Scalars['String'];
  receivedAt: Scalars['Time'];
  runs: Array<FunctionRunV2>;
  source: Maybe<EventSource>;
  version: Maybe<Scalars['String']>;
};

export type EventsBatchConfiguration = {
  __typename?: 'EventsBatchConfiguration';
  key: Maybe<Scalars['String']>;
  /** The maximum number of events a batch can have. */
  maxSize: Scalars['Int'];
  /** How long to wait before running the function with the batch. */
  timeout: Scalars['String'];
};

export type EventsConnection = {
  __typename?: 'EventsConnection';
  edges: Array<EventsEdge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
};

export type EventsEdge = {
  __typename?: 'EventsEdge';
  cursor: Scalars['String'];
  node: EventV2;
};

export type EventsFilter = {
  eventNames?: InputMaybe<Array<Scalars['String']>>;
  from: Scalars['Time'];
  includeInternalEvents?: Scalars['Boolean'];
  query?: InputMaybe<Scalars['String']>;
  until?: InputMaybe<Scalars['Time']>;
};

export type EventsQuery = {
  lastEventId?: InputMaybe<Scalars['ID']>;
  workspaceId?: Scalars['ID'];
};

export type Function = {
  __typename?: 'Function';
  app: App;
  appID: Scalars['String'];
  concurrency: Scalars['Int'];
  config: Scalars['String'];
  configuration: FunctionConfiguration;
  failureHandler: Maybe<Function>;
  id: Scalars['String'];
  name: Scalars['String'];
  slug: Scalars['String'];
  triggers: Maybe<Array<FunctionTrigger>>;
  url: Scalars['String'];
};

export type FunctionConfiguration = {
  __typename?: 'FunctionConfiguration';
  cancellations: Array<CancellationConfiguration>;
  concurrency: Array<ConcurrencyConfiguration>;
  debounce: Maybe<DebounceConfiguration>;
  eventsBatch: Maybe<EventsBatchConfiguration>;
  priority: Maybe<Scalars['String']>;
  rateLimit: Maybe<RateLimitConfiguration>;
  retries: RetryConfiguration;
  singleton: Maybe<SingletonConfiguration>;
  throttle: Maybe<ThrottleConfiguration>;
};

export type FunctionEvent = {
  __typename?: 'FunctionEvent';
  createdAt: Maybe<Scalars['Time']>;
  functionRun: Maybe<FunctionRun>;
  output: Maybe<Scalars['String']>;
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
  functionSlug: Scalars['String'];
  workspaceId?: Scalars['ID'];
};

export type FunctionRun = {
  __typename?: 'FunctionRun';
  batchCreatedAt: Maybe<Scalars['Time']>;
  batchID: Maybe<Scalars['ULID']>;
  cron: Maybe<Scalars['String']>;
  event: Maybe<Event>;
  eventID: Scalars['ID'];
  events: Array<Event>;
  finishedAt: Maybe<Scalars['Time']>;
  function: Maybe<Function>;
  functionID: Scalars['String'];
  history: Array<RunHistoryItem>;
  historyItemOutput: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  output: Maybe<Scalars['String']>;
  /** @deprecated Field no longer supported */
  pendingSteps: Maybe<Scalars['Int']>;
  startedAt: Maybe<Scalars['Time']>;
  status: Maybe<FunctionRunStatus>;
  waitingFor: Maybe<StepEventWait>;
  workspace: Maybe<Workspace>;
};


export type FunctionRunHistoryItemOutputArgs = {
  id: Scalars['ULID'];
};

export type FunctionRunEvent = FunctionEvent | StepEvent;

export type FunctionRunQuery = {
  functionRunId: Scalars['ID'];
  workspaceId?: Scalars['ID'];
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
  appID: Scalars['UUID'];
  batchCreatedAt: Maybe<Scalars['Time']>;
  cronSchedule: Maybe<Scalars['String']>;
  endedAt: Maybe<Scalars['Time']>;
  eventName: Maybe<Scalars['String']>;
  function: Function;
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
};


export type FunctionRunV2TraceArgs = {
  preview: InputMaybe<Scalars['Boolean']>;
};

export type FunctionRunV2Edge = {
  __typename?: 'FunctionRunV2Edge';
  cursor: Scalars['String'];
  node: FunctionRunV2;
};

export type FunctionRunsQuery = {
  workspaceId?: Scalars['ID'];
};

export enum FunctionStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Running = 'RUNNING'
}

export type FunctionTrigger = {
  __typename?: 'FunctionTrigger';
  condition: Maybe<Scalars['String']>;
  type: FunctionTriggerTypes;
  value: Scalars['String'];
};

export enum FunctionTriggerTypes {
  Cron = 'CRON',
  Event = 'EVENT'
}

export type FunctionVersion = {
  __typename?: 'FunctionVersion';
  config: Scalars['String'];
  createdAt: Scalars['Time'];
  functionId: Scalars['ID'];
  updatedAt: Scalars['Time'];
  validFrom: Maybe<Scalars['Time']>;
  validTo: Maybe<Scalars['Time']>;
  version: Scalars['Uint'];
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
  functionID: Scalars['String'];
  returnEventID: Maybe<Scalars['ULID']>;
  runID: Maybe<Scalars['ULID']>;
  timedOut: Maybe<Scalars['Boolean']>;
  timeout: Scalars['Time'];
  triggeringEventID: Scalars['ULID'];
};

export type Mutation = {
  __typename?: 'Mutation';
  cancelRun: FunctionRun;
  createApp: App;
  deleteApp: Scalars['String'];
  deleteAppByName: Scalars['Boolean'];
  invokeFunction: Maybe<Scalars['Boolean']>;
  rerun: Scalars['ULID'];
  updateApp: App;
};


export type MutationCancelRunArgs = {
  runID: Scalars['ULID'];
};


export type MutationCreateAppArgs = {
  input: CreateAppInput;
};


export type MutationDeleteAppArgs = {
  id: Scalars['String'];
};


export type MutationDeleteAppByNameArgs = {
  name: Scalars['String'];
};


export type MutationInvokeFunctionArgs = {
  data: InputMaybe<Scalars['Map']>;
  debugRunID: InputMaybe<Scalars['ULID']>;
  debugSessionID: InputMaybe<Scalars['ULID']>;
  functionSlug: Scalars['String'];
  user: InputMaybe<Scalars['Map']>;
};


export type MutationRerunArgs = {
  debugRunID: InputMaybe<Scalars['ULID']>;
  debugSessionID: InputMaybe<Scalars['ULID']>;
  fromStep: InputMaybe<RerunFromStepInput>;
  runID: Scalars['ULID'];
};


export type MutationUpdateAppArgs = {
  input: UpdateAppInput;
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

export type Query = {
  __typename?: 'Query';
  app: Maybe<App>;
  apps: Array<App>;
  event: Maybe<Event>;
  eventV2: EventV2;
  events: Maybe<Array<Event>>;
  eventsV2: EventsConnection;
  functionBySlug: Maybe<Function>;
  functionRun: Maybe<FunctionRun>;
  functions: Maybe<Array<Function>>;
  run: Maybe<FunctionRunV2>;
  runTraceSpanOutputByID: RunTraceSpanOutput;
  runTrigger: RunTraceTrigger;
  runs: RunsV2Connection;
  stream: Array<StreamItem>;
  workerConnection: Maybe<ConnectV1WorkerConnection>;
  workerConnections: ConnectV1WorkerConnectionsConnection;
};


export type QueryAppArgs = {
  id: Scalars['UUID'];
};


export type QueryAppsArgs = {
  filter: InputMaybe<AppsFilterV1>;
};


export type QueryEventArgs = {
  query: EventQuery;
};


export type QueryEventV2Args = {
  id: Scalars['ULID'];
};


export type QueryEventsArgs = {
  query: EventsQuery;
};


export type QueryEventsV2Args = {
  after: InputMaybe<Scalars['String']>;
  filter: EventsFilter;
  first?: Scalars['Int'];
};


export type QueryFunctionBySlugArgs = {
  query: FunctionQuery;
};


export type QueryFunctionRunArgs = {
  query: FunctionRunQuery;
};


export type QueryRunArgs = {
  runID: Scalars['String'];
};


export type QueryRunTraceSpanOutputByIdArgs = {
  outputID: Scalars['String'];
};


export type QueryRunTriggerArgs = {
  runID: Scalars['String'];
};


export type QueryRunsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: RunsFilterV2;
  first?: Scalars['Int'];
  orderBy: Array<RunsV2OrderBy>;
};


export type QueryStreamArgs = {
  query: StreamQuery;
};


export type QueryWorkerConnectionArgs = {
  connectionId: Scalars['ULID'];
};


export type QueryWorkerConnectionsArgs = {
  after: InputMaybe<Scalars['String']>;
  filter: ConnectV1WorkerConnectionsFilter;
  first?: Scalars['Int'];
  orderBy: Array<ConnectV1WorkerConnectionsOrderBy>;
};

export type RateLimitConfiguration = {
  __typename?: 'RateLimitConfiguration';
  key: Maybe<Scalars['String']>;
  limit: Scalars['Int'];
  period: Scalars['String'];
};

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

export type RunStepInfo = {
  __typename?: 'RunStepInfo';
  type: Maybe<Scalars['String']>;
};

export type RunTraceSpan = {
  __typename?: 'RunTraceSpan';
  appID: Scalars['UUID'];
  attempts: Maybe<Scalars['Int']>;
  childrenSpans: Array<RunTraceSpan>;
  duration: Maybe<Scalars['Int']>;
  endedAt: Maybe<Scalars['Time']>;
  functionID: Scalars['UUID'];
  isRoot: Scalars['Boolean'];
  isUserland: Scalars['Boolean'];
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
  stepType: Scalars['String'];
  traceID: Scalars['String'];
  userlandSpan: Maybe<UserlandSpan>;
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
  Queued = 'QUEUED',
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

export type RunsFilterV2 = {
  appIDs?: InputMaybe<Array<Scalars['UUID']>>;
  from: Scalars['Time'];
  functionIDs?: InputMaybe<Array<Scalars['UUID']>>;
  query?: InputMaybe<Scalars['String']>;
  status?: InputMaybe<Array<FunctionRunStatus>>;
  timeField?: InputMaybe<RunsV2OrderByField>;
  until?: InputMaybe<Scalars['Time']>;
};

export enum RunsOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type RunsV2Connection = {
  __typename?: 'RunsV2Connection';
  edges: Array<FunctionRunV2Edge>;
  pageInfo: PageInfo;
  totalCount: Scalars['Int'];
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
  key: Maybe<Scalars['String']>;
  mode: SingletonMode;
};

export enum SingletonMode {
  Cancel = 'CANCEL',
  Skip = 'SKIP'
}

export type SleepStepInfo = {
  __typename?: 'SleepStepInfo';
  sleepUntil: Scalars['Time'];
};

export type StepError = {
  __typename?: 'StepError';
  cause: Maybe<Scalars['Unknown']>;
  message: Scalars['String'];
  name: Maybe<Scalars['String']>;
  stack: Maybe<Scalars['String']>;
};

export type StepEvent = {
  __typename?: 'StepEvent';
  createdAt: Maybe<Scalars['Time']>;
  functionRun: Maybe<FunctionRun>;
  name: Maybe<Scalars['String']>;
  output: Maybe<Scalars['String']>;
  stepID: Maybe<Scalars['String']>;
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
  eventName: Maybe<Scalars['String']>;
  expiryTime: Scalars['Time'];
  expression: Maybe<Scalars['String']>;
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
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  inBatch: Scalars['Boolean'];
  runs: Maybe<Array<Maybe<FunctionRun>>>;
  trigger: Scalars['String'];
  type: StreamType;
};

export type StreamQuery = {
  after?: InputMaybe<Scalars['ID']>;
  before?: InputMaybe<Scalars['ID']>;
  includeInternalEvents?: InputMaybe<Scalars['Boolean']>;
  limit?: Scalars['Int'];
};

export enum StreamType {
  Cron = 'CRON',
  Event = 'EVENT'
}

export type ThrottleConfiguration = {
  __typename?: 'ThrottleConfiguration';
  burst: Scalars['Int'];
  key: Maybe<Scalars['String']>;
  limit: Scalars['Int'];
  period: Scalars['String'];
};

export type UpdateAppInput = {
  id: Scalars['String'];
  url: Scalars['String'];
};

export type UserlandSpan = {
  __typename?: 'UserlandSpan';
  resourceAttrs: Maybe<Scalars['Bytes']>;
  scopeName: Maybe<Scalars['String']>;
  scopeVersion: Maybe<Scalars['String']>;
  serviceName: Maybe<Scalars['String']>;
  spanAttrs: Maybe<Scalars['Bytes']>;
  spanKind: Maybe<Scalars['String']>;
  spanName: Maybe<Scalars['String']>;
};

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String'];
  expression: Maybe<Scalars['String']>;
  foundEventID: Maybe<Scalars['ULID']>;
  timedOut: Maybe<Scalars['Boolean']>;
  timeout: Scalars['Time'];
};

export type WaitForSignalStepInfo = {
  __typename?: 'WaitForSignalStepInfo';
  signal: Scalars['String'];
  timedOut: Maybe<Scalars['Boolean']>;
  timeout: Scalars['Time'];
};

export type Workspace = {
  __typename?: 'Workspace';
  id: Scalars['ID'];
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
}>;


export type RerunMutation = { __typename?: 'Mutation', rerun: any };

export type RerunFromStepMutationVariables = Exact<{
  runID: Scalars['ULID'];
  fromStep: RerunFromStepInput;
  debugSessionID?: InputMaybe<Scalars['ULID']>;
  debugRunID?: InputMaybe<Scalars['ULID']>;
}>;


export type RerunFromStepMutation = { __typename?: 'Mutation', rerun: any };

export type GetRunsQueryVariables = Exact<{
  appIDs: InputMaybe<Array<Scalars['UUID']> | Scalars['UUID']>;
  startTime: Scalars['Time'];
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsV2OrderByField;
  functionRunCursor?: InputMaybe<Scalars['String']>;
  celQuery?: InputMaybe<Scalars['String']>;
}>;


export type GetRunsQuery = { __typename?: 'Query', runs: { __typename?: 'RunsV2Connection', edges: Array<{ __typename?: 'FunctionRunV2Edge', node: { __typename?: 'FunctionRunV2', cronSchedule: string | null, eventName: string | null, id: any, isBatch: boolean, queuedAt: any, endedAt: any | null, startedAt: any | null, status: FunctionRunStatus, hasAI: boolean, app: { __typename?: 'App', externalID: string, name: string }, function: { __typename?: 'Function', name: string, slug: string } } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountRunsQueryVariables = Exact<{
  startTime: Scalars['Time'];
  status: InputMaybe<Array<FunctionRunStatus> | FunctionRunStatus>;
  timeField: RunsV2OrderByField;
}>;


export type CountRunsQuery = { __typename?: 'Query', runs: { __typename?: 'RunsV2Connection', totalCount: number } };

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null };

export type GetRunQueryVariables = Exact<{
  runID: Scalars['String'];
  preview: InputMaybe<Scalars['Boolean']>;
}>;


export type GetRunQuery = { __typename?: 'Query', run: { __typename?: 'FunctionRunV2', hasAI: boolean, function: { __typename?: 'Function', id: string, name: string, slug: string, app: { __typename?: 'App', name: string } }, trace: { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, isUserland: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepType: string, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null }>, userlandSpan: { __typename?: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: any | null, resourceAttrs: any | null } | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: any, timedOut: boolean | null } | null } | null } | null };

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


export type GetWorkerConnectionsQuery = { __typename?: 'Query', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number, edges: Array<{ __typename?: 'ConnectV1WorkerConnectionEdge', node: { __typename?: 'ConnectV1WorkerConnection', id: any, gatewayId: any, instanceId: string, workerIp: string, connectedAt: any, lastHeartbeatAt: any | null, disconnectedAt: any | null, disconnectReason: string | null, status: ConnectV1ConnectionStatus, groupHash: string, sdkLang: string, sdkVersion: string, sdkPlatform: string, syncId: any | null, appVersion: string | null, functionCount: number, cpuCores: number, memBytes: number, os: string, app: { __typename?: 'App', id: string } | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

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
    mutation Rerun($runID: ULID!) {
  rerun(runID: $runID)
}
    `;
export const RerunFromStepDocument = `
    mutation RerunFromStep($runID: ULID!, $fromStep: RerunFromStepInput!, $debugSessionID: ULID = null, $debugRunID: ULID = null) {
  rerun(
    runID: $runID
    fromStep: $fromStep
    debugSessionID: $debugSessionID
    debugRunID: $debugRunID
  )
}
    `;
export const GetRunsDocument = `
    query GetRuns($appIDs: [UUID!], $startTime: Time!, $status: [FunctionRunStatus!], $timeField: RunsV2OrderByField!, $functionRunCursor: String = null, $celQuery: String = null) {
  runs(
    filter: {appIDs: $appIDs, from: $startTime, status: $status, timeField: $timeField, query: $celQuery}
    orderBy: [{field: $timeField, direction: DESC}]
    after: $functionRunCursor
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
    query CountRuns($startTime: Time!, $status: [FunctionRunStatus!], $timeField: RunsV2OrderByField!) {
  runs(
    filter: {from: $startTime, status: $status, timeField: $timeField}
    orderBy: [{field: $timeField, direction: DESC}]
  ) {
    totalCount
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
    query GetEventsV2($cursor: String, $startTime: Time!, $endTime: Time, $celQuery: String = null, $eventNames: [String!] = null, $includeInternalEvents: Boolean = true) {
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
  }),
});

export { injectedRtkApi as api };
export const { useGetEventQuery, useLazyGetEventQuery, useGetFunctionsQuery, useLazyGetFunctionsQuery, useGetFunctionQuery, useLazyGetFunctionQuery, useGetAppsQuery, useLazyGetAppsQuery, useGetAppQuery, useLazyGetAppQuery, useCreateAppMutation, useUpdateAppMutation, useDeleteAppMutation, useInvokeFunctionMutation, useCancelRunMutation, useRerunMutation, useRerunFromStepMutation, useGetRunsQuery, useLazyGetRunsQuery, useCountRunsQuery, useLazyCountRunsQuery, useGetRunQuery, useLazyGetRunQuery, useGetTraceResultQuery, useLazyGetTraceResultQuery, useGetTriggerQuery, useLazyGetTriggerQuery, useGetWorkerConnectionsQuery, useLazyGetWorkerConnectionsQuery, useCountWorkerConnectionsQuery, useLazyCountWorkerConnectionsQuery, useGetEventsV2Query, useLazyGetEventsV2Query, useGetEventV2Query, useLazyGetEventV2Query, useGetEventV2PayloadQuery, useLazyGetEventV2PayloadQuery, useGetEventV2RunsQuery, useLazyGetEventV2RunsQuery } = injectedRtkApi;

