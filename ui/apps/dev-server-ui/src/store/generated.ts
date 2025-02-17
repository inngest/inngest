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
};

export type ActionVersionQuery = {
  dsn: Scalars['String'];
  versionMajor?: InputMaybe<Scalars['Int']>;
  versionMinor?: InputMaybe<Scalars['Int']>;
};

export type App = {
  __typename?: 'App';
  autodiscovered: Scalars['Boolean'];
  checksum: Maybe<Scalars['String']>;
  connected: Scalars['Boolean'];
  connectionType: AppConnectionType;
  error: Maybe<Scalars['String']>;
  externalID: Scalars['String'];
  framework: Maybe<Scalars['String']>;
  functionCount: Scalars['Int'];
  /** @deprecated connectionType is deprecated. Use method instead. */
  functions: Array<Function>;
  id: Scalars['ID'];
  method: AppMethod;
  name: Scalars['String'];
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  url: Maybe<Scalars['String']>;
};

export enum AppConnectionType {
  Connect = 'CONNECT',
  Serverless = 'SERVERLESS'
}

export enum AppMethod {
  Connect = 'CONNECT',
  Serve = 'SERVE'
}

export type AppsFilterV1 = {
  /** @deprecated connectionType is deprecated. Use method instead. */
  connectionType?: InputMaybe<AppConnectionType>;
  method?: InputMaybe<AppMethod>;
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
  app: Maybe<App>;
  appID: Maybe<Scalars['UUID']>;
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

export enum EventStatus {
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  NoFunctions = 'NO_FUNCTIONS',
  PartiallyFailed = 'PARTIALLY_FAILED',
  Paused = 'PAUSED',
  Running = 'RUNNING'
}

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
  id: Scalars['String'];
  name: Scalars['String'];
  slug: Scalars['String'];
  triggers: Maybe<Array<FunctionTrigger>>;
  url: Scalars['String'];
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
  functionSlug: Scalars['String'];
  user: InputMaybe<Scalars['Map']>;
};


export type MutationRerunArgs = {
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
  events: Maybe<Array<Event>>;
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


export type QueryEventsArgs = {
  query: EventsQuery;
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

export type RerunFromStepInput = {
  input?: InputMaybe<Scalars['Bytes']>;
  stepID: Scalars['String'];
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

export type SleepStepInfo = {
  __typename?: 'SleepStepInfo';
  sleepUntil: Scalars['Time'];
};

export type StepError = {
  __typename?: 'StepError';
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

export type StepInfo = InvokeStepInfo | RunStepInfo | SleepStepInfo | WaitForEventStepInfo;

export enum StepOp {
  AiGateway = 'AI_GATEWAY',
  Invoke = 'INVOKE',
  Run = 'RUN',
  Sleep = 'SLEEP',
  WaitForEvent = 'WAIT_FOR_EVENT'
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

export type UpdateAppInput = {
  id: Scalars['String'];
  url: Scalars['String'];
};

export type WaitForEventStepInfo = {
  __typename?: 'WaitForEventStepInfo';
  eventName: Scalars['String'];
  expression: Maybe<Scalars['String']>;
  foundEventID: Maybe<Scalars['ULID']>;
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

export type GetFunctionRunQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunQuery = { __typename?: 'Query', functionRun: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus | null, startedAt: any | null, finishedAt: any | null, output: string | null, pendingSteps: number | null, batchID: any | null, batchCreatedAt: any | null, waitingFor: { __typename?: 'StepEventWait', expiryTime: any, eventName: string | null, expression: string | null } | null, function: { __typename?: 'Function', name: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null } | null, event: { __typename?: 'Event', id: any, raw: string | null } | null, events: Array<{ __typename?: 'Event', createdAt: any | null, id: any, name: string | null, raw: string | null }>, history: Array<{ __typename?: 'RunHistoryItem', attempt: number, createdAt: any, functionVersion: number, groupID: any | null, id: any, stepName: string | null, type: HistoryType, url: string | null, cancel: { __typename?: 'RunHistoryCancel', eventID: any | null, expression: string | null, userID: any | null } | null, sleep: { __typename?: 'RunHistorySleep', until: any } | null, waitForEvent: { __typename?: 'RunHistoryWaitForEvent', eventName: string, expression: string | null, timeout: any } | null, waitResult: { __typename?: 'RunHistoryWaitResult', eventID: any | null, timeout: boolean } | null, invokeFunction: { __typename?: 'RunHistoryInvokeFunction', eventID: any, functionID: string, correlationID: string, timeout: any } | null, invokeFunctionResult: { __typename?: 'RunHistoryInvokeFunctionResult', eventID: any | null, timeout: boolean, runID: any | null } | null }> } | null };

export type GetFunctionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetFunctionsQuery = { __typename?: 'Query', functions: Array<{ __typename?: 'Function', id: string, slug: string, name: string, url: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null, app: { __typename?: 'App', name: string } }> | null };

export type GetAppsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAppsQuery = { __typename?: 'Query', apps: Array<{ __typename?: 'App', id: string, name: string, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: AppMethod, functions: Array<{ __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string }> }> };

export type GetAppQueryVariables = Exact<{
  id: Scalars['UUID'];
}>;


export type GetAppQuery = { __typename?: 'Query', app: { __typename?: 'App', id: string, name: string, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: AppMethod, functions: Array<{ __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string, triggers: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null }> } | null };

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

export type GetTriggersStreamQueryVariables = Exact<{
  limit: Scalars['Int'];
  after: InputMaybe<Scalars['ID']>;
  before: InputMaybe<Scalars['ID']>;
  includeInternalEvents: Scalars['Boolean'];
}>;


export type GetTriggersStreamQuery = { __typename?: 'Query', stream: Array<{ __typename?: 'StreamItem', createdAt: any, id: string, inBatch: boolean, trigger: string, type: StreamType, runs: Array<{ __typename?: 'FunctionRun', batchID: any | null, id: string, events: Array<{ __typename?: 'Event', id: any }>, function: { __typename?: 'Function', name: string } | null } | null> | null }> };

export type GetFunctionRunStatusQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunStatusQuery = { __typename?: 'Query', functionRun: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus | null, function: { __typename?: 'Function', name: string } | null } | null };

export type GetFunctionRunOutputQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunOutputQuery = { __typename?: 'Query', functionRun: { __typename?: 'FunctionRun', id: string, status: FunctionRunStatus | null, output: string | null } | null };

export type GetHistoryItemOutputQueryVariables = Exact<{
  historyItemID: Scalars['ULID'];
  runID: Scalars['ID'];
}>;


export type GetHistoryItemOutputQuery = { __typename?: 'Query', functionRun: { __typename?: 'FunctionRun', historyItemOutput: string | null } | null };

export type InvokeFunctionMutationVariables = Exact<{
  functionSlug: Scalars['String'];
  data: InputMaybe<Scalars['Map']>;
  user: InputMaybe<Scalars['Map']>;
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

export type TraceDetailsFragment = { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | null };

export type GetRunQueryVariables = Exact<{
  runID: Scalars['String'];
}>;


export type GetRunQuery = { __typename?: 'Query', run: { __typename?: 'FunctionRunV2', hasAI: boolean, function: { __typename?: 'Function', id: string, name: string, slug: string, app: { __typename?: 'App', name: string } }, trace: { __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, childrenSpans: Array<{ __typename?: 'RunTraceSpan', name: string, status: RunTraceSpanStatus, attempts: number | null, queuedAt: any, startedAt: any | null, endedAt: any | null, isRoot: boolean, outputID: string | null, spanID: string, stepID: string | null, stepOp: StepOp | null, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | null }>, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | null }>, stepInfo: { __typename: 'InvokeStepInfo', triggeringEventID: any, functionID: string, timeout: any, returnEventID: any | null, runID: any | null, timedOut: boolean | null } | { __typename: 'RunStepInfo', type: string | null } | { __typename: 'SleepStepInfo', sleepUntil: any } | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: any, foundEventID: any | null, timedOut: boolean | null } | null } | null } | null };

export type GetTraceResultQueryVariables = Exact<{
  traceID: Scalars['String'];
}>;


export type GetTraceResultQuery = { __typename?: 'Query', runTraceSpanOutputByID: { __typename?: 'RunTraceSpanOutput', input: any | null, data: any | null, error: { __typename?: 'StepError', message: string, name: string | null, stack: string | null } | null } };

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


export type GetWorkerConnectionsQuery = { __typename?: 'Query', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', edges: Array<{ __typename?: 'ConnectV1WorkerConnectionEdge', node: { __typename?: 'ConnectV1WorkerConnection', id: any, gatewayId: any, instanceId: string, workerIp: string, connectedAt: any, lastHeartbeatAt: any | null, disconnectedAt: any | null, disconnectReason: string | null, status: ConnectV1ConnectionStatus, groupHash: string, sdkLang: string, sdkVersion: string, sdkPlatform: string, syncId: any | null, buildId: string | null, functionCount: number, cpuCores: number, memBytes: number, os: string, app: { __typename?: 'App', id: string } | null } }>, pageInfo: { __typename?: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountWorkerConnectionsQueryVariables = Exact<{
  appID: Scalars['UUID'];
  status: InputMaybe<Array<ConnectV1ConnectionStatus> | ConnectV1ConnectionStatus>;
}>;


export type CountWorkerConnectionsQuery = { __typename?: 'Query', workerConnections: { __typename?: 'ConnectV1WorkerConnectionsConnection', totalCount: number } };

export const TraceDetailsFragmentDoc = `
    fragment TraceDetails on RunTraceSpan {
  name
  status
  attempts
  queuedAt
  startedAt
  endedAt
  isRoot
  outputID
  spanID
  stepID
  stepOp
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
export const GetFunctionRunDocument = `
    query GetFunctionRun($id: ID!) {
  functionRun(query: {functionRunId: $id}) {
    id
    status
    startedAt
    finishedAt
    output
    pendingSteps
    waitingFor {
      expiryTime
      eventName
      expression
    }
    function {
      name
      triggers {
        type
        value
      }
    }
    event {
      id
      raw
    }
    batchID
    batchCreatedAt
    events {
      createdAt
      id
      name
      raw
    }
    history {
      attempt
      cancel {
        eventID
        expression
        userID
      }
      createdAt
      functionVersion
      groupID
      id
      sleep {
        until
      }
      stepName
      type
      url
      waitForEvent {
        eventName
        expression
        timeout
      }
      waitResult {
        eventID
        timeout
      }
      invokeFunction {
        eventID
        functionID
        correlationID
        timeout
      }
      invokeFunctionResult {
        eventID
        timeout
        runID
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
export const GetAppsDocument = `
    query GetApps {
  apps {
    id
    name
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
export const GetTriggersStreamDocument = `
    query GetTriggersStream($limit: Int!, $after: ID, $before: ID, $includeInternalEvents: Boolean!) {
  stream(
    query: {limit: $limit, after: $after, before: $before, includeInternalEvents: $includeInternalEvents}
  ) {
    createdAt
    id
    inBatch
    trigger
    type
    runs {
      batchID
      events {
        id
      }
      id
      function {
        name
      }
    }
  }
}
    `;
export const GetFunctionRunStatusDocument = `
    query GetFunctionRunStatus($id: ID!) {
  functionRun(query: {functionRunId: $id}) {
    id
    function {
      name
    }
    status
  }
}
    `;
export const GetFunctionRunOutputDocument = `
    query GetFunctionRunOutput($id: ID!) {
  functionRun(query: {functionRunId: $id}) {
    id
    status
    output
  }
}
    `;
export const GetHistoryItemOutputDocument = `
    query GetHistoryItemOutput($historyItemID: ULID!, $runID: ID!) {
  functionRun(query: {functionRunId: $runID}) {
    historyItemOutput(id: $historyItemID)
  }
}
    `;
export const InvokeFunctionDocument = `
    mutation InvokeFunction($functionSlug: String!, $data: Map, $user: Map) {
  invokeFunction(data: $data, functionSlug: $functionSlug, user: $user)
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
    mutation RerunFromStep($runID: ULID!, $fromStep: RerunFromStepInput!) {
  rerun(runID: $runID, fromStep: $fromStep)
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
    query GetRun($runID: String!) {
  run(runID: $runID) {
    function {
      app {
        name
      }
      id
      name
      slug
    }
    trace {
      ...TraceDetails
      childrenSpans {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
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
        buildId
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
  }
}
    `;
export const CountWorkerConnectionsDocument = `
    query CountWorkerConnections($appID: UUID!, $status: [ConnectV1ConnectionStatus!]) {
  workerConnections(
    filter: {appIDs: [$appID], status: $status, timeField: CONNECTED_AT}
    orderBy: [{field: CONNECTED_AT, direction: DESC}]
  ) {
    totalCount
  }
}
    `;

const injectedRtkApi = api.injectEndpoints({
  endpoints: (build) => ({
    GetEvent: build.query<GetEventQuery, GetEventQueryVariables>({
      query: (variables) => ({ document: GetEventDocument, variables })
    }),
    GetFunctionRun: build.query<GetFunctionRunQuery, GetFunctionRunQueryVariables>({
      query: (variables) => ({ document: GetFunctionRunDocument, variables })
    }),
    GetFunctions: build.query<GetFunctionsQuery, GetFunctionsQueryVariables | void>({
      query: (variables) => ({ document: GetFunctionsDocument, variables })
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
    GetTriggersStream: build.query<GetTriggersStreamQuery, GetTriggersStreamQueryVariables>({
      query: (variables) => ({ document: GetTriggersStreamDocument, variables })
    }),
    GetFunctionRunStatus: build.query<GetFunctionRunStatusQuery, GetFunctionRunStatusQueryVariables>({
      query: (variables) => ({ document: GetFunctionRunStatusDocument, variables })
    }),
    GetFunctionRunOutput: build.query<GetFunctionRunOutputQuery, GetFunctionRunOutputQueryVariables>({
      query: (variables) => ({ document: GetFunctionRunOutputDocument, variables })
    }),
    GetHistoryItemOutput: build.query<GetHistoryItemOutputQuery, GetHistoryItemOutputQueryVariables>({
      query: (variables) => ({ document: GetHistoryItemOutputDocument, variables })
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
  }),
});

export { injectedRtkApi as api };
export const { useGetEventQuery, useLazyGetEventQuery, useGetFunctionRunQuery, useLazyGetFunctionRunQuery, useGetFunctionsQuery, useLazyGetFunctionsQuery, useGetAppsQuery, useLazyGetAppsQuery, useGetAppQuery, useLazyGetAppQuery, useCreateAppMutation, useUpdateAppMutation, useDeleteAppMutation, useGetTriggersStreamQuery, useLazyGetTriggersStreamQuery, useGetFunctionRunStatusQuery, useLazyGetFunctionRunStatusQuery, useGetFunctionRunOutputQuery, useLazyGetFunctionRunOutputQuery, useGetHistoryItemOutputQuery, useLazyGetHistoryItemOutputQuery, useInvokeFunctionMutation, useCancelRunMutation, useRerunMutation, useRerunFromStepMutation, useGetRunsQuery, useLazyGetRunsQuery, useCountRunsQuery, useLazyCountRunsQuery, useGetRunQuery, useLazyGetRunQuery, useGetTraceResultQuery, useLazyGetTraceResultQuery, useGetTriggerQuery, useLazyGetTriggerQuery, useGetWorkerConnectionsQuery, useLazyGetWorkerConnectionsQuery, useCountWorkerConnectionsQuery, useLazyCountWorkerConnectionsQuery } = injectedRtkApi;

