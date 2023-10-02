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
  /** The environment for the function to be run: `"prod"` or `"test"` */
  Environment: any;
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
  checksum?: Maybe<Scalars['String']>;
  connected: Scalars['Boolean'];
  error?: Maybe<Scalars['String']>;
  framework?: Maybe<Scalars['String']>;
  functionCount: Scalars['Int'];
  functions: Array<Function>;
  id: Scalars['ID'];
  name: Scalars['String'];
  sdkLanguage: Scalars['String'];
  sdkVersion: Scalars['String'];
  url?: Maybe<Scalars['String']>;
};

export type CreateAppInput = {
  url: Scalars['String'];
};

export type Event = {
  __typename?: 'Event';
  createdAt?: Maybe<Scalars['Time']>;
  functionRuns?: Maybe<Array<FunctionRun>>;
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  payload?: Maybe<Scalars['String']>;
  pendingRuns?: Maybe<Scalars['Int']>;
  raw?: Maybe<Scalars['String']>;
  schema?: Maybe<Scalars['String']>;
  status?: Maybe<EventStatus>;
  totalRuns?: Maybe<Scalars['Int']>;
  workspace?: Maybe<Workspace>;
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
  concurrency: Scalars['Int'];
  config: Scalars['String'];
  id: Scalars['String'];
  name: Scalars['String'];
  slug: Scalars['String'];
  triggers?: Maybe<Array<FunctionTrigger>>;
  url: Scalars['String'];
};

export type FunctionEvent = {
  __typename?: 'FunctionEvent';
  createdAt?: Maybe<Scalars['Time']>;
  functionRun?: Maybe<FunctionRun>;
  output?: Maybe<Scalars['String']>;
  type?: Maybe<FunctionEventType>;
  workspace?: Maybe<Workspace>;
};

export enum FunctionEventType {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Started = 'STARTED'
}

export type FunctionRun = {
  __typename?: 'FunctionRun';
  event?: Maybe<Event>;
  finishedAt?: Maybe<Scalars['Time']>;
  function?: Maybe<Function>;
  functionID: Scalars['String'];
  history: Array<RunHistoryItem>;
  historyItemOutput: Scalars['String'];
  id: Scalars['ID'];
  /** @deprecated Field no longer supported */
  name?: Maybe<Scalars['String']>;
  output?: Maybe<Scalars['String']>;
  /** @deprecated Field no longer supported */
  pendingSteps?: Maybe<Scalars['Int']>;
  startedAt?: Maybe<Scalars['Time']>;
  status?: Maybe<FunctionRunStatus>;
  /** @deprecated Field no longer supported */
  timeline?: Maybe<Array<FunctionRunEvent>>;
  waitingFor?: Maybe<StepEventWait>;
  workspace?: Maybe<Workspace>;
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
  Running = 'RUNNING'
}

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
  validFrom?: Maybe<Scalars['Time']>;
  validTo?: Maybe<Scalars['Time']>;
  version: Scalars['Uint'];
};

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

export type Mutation = {
  __typename?: 'Mutation';
  createApp: App;
  deleteApp: Scalars['String'];
  updateApp: App;
};


export type MutationCreateAppArgs = {
  input: CreateAppInput;
};


export type MutationDeleteAppArgs = {
  id: Scalars['String'];
};


export type MutationUpdateAppArgs = {
  input: UpdateAppInput;
};

export type Query = {
  __typename?: 'Query';
  apps: Array<App>;
  event?: Maybe<Event>;
  events?: Maybe<Array<Event>>;
  functionRun?: Maybe<FunctionRun>;
  functionRuns?: Maybe<Array<FunctionRun>>;
  functions?: Maybe<Array<Function>>;
  stream: Array<StreamItem>;
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


export type QueryFunctionRunsArgs = {
  query: FunctionRunsQuery;
};


export type QueryStreamArgs = {
  query: StreamQuery;
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

export type StepEvent = {
  __typename?: 'StepEvent';
  createdAt?: Maybe<Scalars['Time']>;
  functionRun?: Maybe<FunctionRun>;
  name?: Maybe<Scalars['String']>;
  output?: Maybe<Scalars['String']>;
  stepID?: Maybe<Scalars['String']>;
  type?: Maybe<StepEventType>;
  waitingFor?: Maybe<StepEventWait>;
  workspace?: Maybe<Workspace>;
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
  eventName?: Maybe<Scalars['String']>;
  expiryTime: Scalars['Time'];
  expression?: Maybe<Scalars['String']>;
};

export type StreamItem = {
  __typename?: 'StreamItem';
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  runs?: Maybe<Array<Maybe<FunctionRun>>>;
  trigger: Scalars['String'];
  type: StreamType;
};

export type StreamQuery = {
  after?: InputMaybe<Scalars['Time']>;
  before?: InputMaybe<Scalars['Time']>;
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

export type Workspace = {
  __typename?: 'Workspace';
  id: Scalars['ID'];
};

export type GetEventsStreamQueryVariables = Exact<{ [key: string]: never; }>;


export type GetEventsStreamQuery = { __typename?: 'Query', events?: Array<{ __typename?: 'Event', id: string, name?: string | null, createdAt?: any | null, status?: EventStatus | null, totalRuns?: number | null }> | null };

export type GetFunctionsStreamQueryVariables = Exact<{ [key: string]: never; }>;


export type GetFunctionsStreamQuery = { __typename?: 'Query', functionRuns?: Array<{ __typename?: 'FunctionRun', id: string, status?: FunctionRunStatus | null, startedAt?: any | null, pendingSteps?: number | null, name?: string | null, event?: { __typename?: 'Event', id: string } | null }> | null };

export type GetEventQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetEventQuery = { __typename?: 'Query', event?: { __typename?: 'Event', id: string, name?: string | null, createdAt?: any | null, status?: EventStatus | null, pendingRuns?: number | null, raw?: string | null, functionRuns?: Array<{ __typename?: 'FunctionRun', id: string, name?: string | null, status?: FunctionRunStatus | null, startedAt?: any | null, pendingSteps?: number | null, waitingFor?: { __typename?: 'StepEventWait', expiryTime: any, eventName?: string | null, expression?: string | null } | null }> | null } | null };

export type GetFunctionRunQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunQuery = { __typename?: 'Query', functionRun?: { __typename?: 'FunctionRun', id: string, name?: string | null, status?: FunctionRunStatus | null, startedAt?: any | null, pendingSteps?: number | null, waitingFor?: { __typename?: 'StepEventWait', expiryTime: any, eventName?: string | null, expression?: string | null } | null, event?: { __typename?: 'Event', id: string, raw?: string | null } | null, timeline?: Array<{ __typename: 'FunctionEvent', createdAt?: any | null, output?: string | null, functionType?: FunctionEventType | null } | { __typename: 'StepEvent', createdAt?: any | null, output?: string | null, name?: string | null, stepType?: StepEventType | null, waitingFor?: { __typename?: 'StepEventWait', expiryTime: any, eventName?: string | null, expression?: string | null } | null }> | null } | null };

export type GetFunctionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetFunctionsQuery = { __typename?: 'Query', functions?: Array<{ __typename?: 'Function', id: string, slug: string, name: string, url: string, triggers?: Array<{ __typename?: 'FunctionTrigger', type: FunctionTriggerTypes, value: string }> | null }> | null };

export type GetAppsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAppsQuery = { __typename?: 'Query', apps: Array<{ __typename?: 'App', id: string, name: string, sdkLanguage: string, sdkVersion: string, framework?: string | null, url?: string | null, error?: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, functions: Array<{ __typename?: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string }> }> };

export type CreateAppMutationVariables = Exact<{
  input: CreateAppInput;
}>;


export type CreateAppMutation = { __typename?: 'Mutation', createApp: { __typename?: 'App', url?: string | null } };

export type UpdateAppMutationVariables = Exact<{
  input: UpdateAppInput;
}>;


export type UpdateAppMutation = { __typename?: 'Mutation', updateApp: { __typename?: 'App', url?: string | null, id: string } };

export type DeleteAppMutationVariables = Exact<{
  id: Scalars['String'];
}>;


export type DeleteAppMutation = { __typename?: 'Mutation', deleteApp: string };

export type GetTriggersStreamQueryVariables = Exact<{
  limit: Scalars['Int'];
  after?: InputMaybe<Scalars['Time']>;
  before?: InputMaybe<Scalars['Time']>;
}>;


export type GetTriggersStreamQuery = { __typename?: 'Query', stream: Array<{ __typename?: 'StreamItem', createdAt: any, id: string, trigger: string, type: StreamType, runs?: Array<{ __typename?: 'FunctionRun', id: string } | null> | null }> };

export type GetFunctionRunStatusQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunStatusQuery = { __typename?: 'Query', functionRun?: { __typename?: 'FunctionRun', id: string, status?: FunctionRunStatus | null, function?: { __typename?: 'Function', name: string } | null } | null };

export type GetFunctionRunOutputQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type GetFunctionRunOutputQuery = { __typename?: 'Query', functionRun?: { __typename?: 'FunctionRun', id: string, output?: string | null, status?: FunctionRunStatus | null } | null };


export const GetEventsStreamDocument = `
    query GetEventsStream {
  events(query: {}) {
    id
    name
    createdAt
    status
    totalRuns
  }
}
    `;
export const GetFunctionsStreamDocument = `
    query GetFunctionsStream {
  functionRuns(query: {}) {
    id
    status
    startedAt
    pendingSteps
    name
    event {
      id
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
      id
      name
      status
      startedAt
      pendingSteps
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
    name
    status
    startedAt
    pendingSteps
    waitingFor {
      expiryTime
      eventName
      expression
    }
    event {
      id
      raw
    }
    timeline {
      __typename
      ... on StepEvent {
        stepType: type
        createdAt
        output
        name
        waitingFor {
          expiryTime
          eventName
          expression
        }
      }
      ... on FunctionEvent {
        functionType: type
        createdAt
        output
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
    query GetTriggersStream($limit: Int!, $after: Time, $before: Time) {
  stream(query: {limit: $limit, after: $after, before: $before}) {
    createdAt
    id
    trigger
    type
    runs {
      id
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
    output
    status
  }
}
    `;

const injectedRtkApi = api.injectEndpoints({
  endpoints: (build) => ({
    GetEventsStream: build.query<GetEventsStreamQuery, GetEventsStreamQueryVariables | void>({
      query: (variables) => ({ document: GetEventsStreamDocument, variables })
    }),
    GetFunctionsStream: build.query<GetFunctionsStreamQuery, GetFunctionsStreamQueryVariables | void>({
      query: (variables) => ({ document: GetFunctionsStreamDocument, variables })
    }),
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
  }),
});

export { injectedRtkApi as api };
export const { useGetEventsStreamQuery, useLazyGetEventsStreamQuery, useGetFunctionsStreamQuery, useLazyGetFunctionsStreamQuery, useGetEventQuery, useLazyGetEventQuery, useGetFunctionRunQuery, useLazyGetFunctionRunQuery, useGetFunctionsQuery, useLazyGetFunctionsQuery, useGetAppsQuery, useLazyGetAppsQuery, useCreateAppMutation, useUpdateAppMutation, useDeleteAppMutation, useGetTriggersStreamQuery, useLazyGetTriggersStreamQuery, useGetFunctionRunStatusQuery, useLazyGetFunctionRunStatusQuery, useGetFunctionRunOutputQuery, useLazyGetFunctionRunOutputQuery } = injectedRtkApi;

