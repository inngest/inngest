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
};

export type ActionVersion = {
  __typename?: 'ActionVersion';
  config: Scalars['String'];
  createdAt: Scalars['Time'];
  dsn: Scalars['String'];
  name: Scalars['String'];
  validFrom?: Maybe<Scalars['Time']>;
  validTo?: Maybe<Scalars['Time']>;
  versionMajor: Scalars['Int'];
  versionMinor: Scalars['Int'];
};

export type ActionVersionQuery = {
  dsn: Scalars['String'];
  versionMajor?: InputMaybe<Scalars['Int']>;
  versionMinor?: InputMaybe<Scalars['Int']>;
};

export type Config = {
  __typename?: 'Config';
  execution?: Maybe<ExecutionConfig>;
};

export type CreateActionVersionInput = {
  config: Scalars['String'];
};

export type DeployFunctionInput = {
  config: Scalars['String'];
  env?: InputMaybe<Scalars['Environment']>;
  live?: InputMaybe<Scalars['Boolean']>;
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

export type ExecutionConfig = {
  __typename?: 'ExecutionConfig';
  drivers?: Maybe<ExecutionDriversConfig>;
};

export type ExecutionDockerDriverConfig = {
  __typename?: 'ExecutionDockerDriverConfig';
  namespace?: Maybe<Scalars['String']>;
  registry?: Maybe<Scalars['String']>;
};

export type ExecutionDriversConfig = {
  __typename?: 'ExecutionDriversConfig';
  docker?: Maybe<ExecutionDockerDriverConfig>;
};

export type Function = {
  __typename?: 'Function';
  concurrency?: Maybe<Scalars['Int']>;
  id?: Maybe<Scalars['String']>;
  name?: Maybe<Scalars['String']>;
  triggers?: Maybe<Array<FunctionTrigger>>;
  url?: Maybe<Scalars['String']>;
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
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  pendingSteps?: Maybe<Scalars['Int']>;
  startedAt?: Maybe<Scalars['Time']>;
  status?: Maybe<FunctionRunStatus>;
  timeline?: Maybe<Array<FunctionRunEvent>>;
  waitingFor?: Maybe<StepEventWait>;
  workspace?: Maybe<Workspace>;
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

export type FunctionTrigger = {
  __typename?: 'FunctionTrigger';
  cron?: Maybe<Scalars['String']>;
  event?: Maybe<Scalars['String']>;
};

export type FunctionVersion = {
  __typename?: 'FunctionVersion';
  config: Scalars['String'];
  createdAt: Scalars['Time'];
  functionId: Scalars['ID'];
  updatedAt: Scalars['Time'];
  validFrom?: Maybe<Scalars['Time']>;
  validTo?: Maybe<Scalars['Time']>;
  version: Scalars['Int'];
};

export type Mutation = {
  __typename?: 'Mutation';
  createActionVersion?: Maybe<ActionVersion>;
  deployFunction?: Maybe<FunctionVersion>;
  updateActionVersion?: Maybe<ActionVersion>;
};


export type MutationCreateActionVersionArgs = {
  input: CreateActionVersionInput;
};


export type MutationDeployFunctionArgs = {
  input: DeployFunctionInput;
};


export type MutationUpdateActionVersionArgs = {
  input: UpdateActionVersionInput;
};

export type Query = {
  __typename?: 'Query';
  actionVersion?: Maybe<ActionVersion>;
  config?: Maybe<Config>;
  event?: Maybe<Event>;
  events?: Maybe<Array<Event>>;
  functionRun?: Maybe<FunctionRun>;
  functionRuns?: Maybe<Array<FunctionRun>>;
  functions?: Maybe<Array<Function>>;
};


export type QueryActionVersionArgs = {
  query: ActionVersionQuery;
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

export type UpdateActionVersionInput = {
  dsn: Scalars['String'];
  enabled?: InputMaybe<Scalars['Boolean']>;
  versionMajor: Scalars['Int'];
  versionMinor: Scalars['Int'];
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


export type GetFunctionsQuery = { __typename?: 'Query', functions?: Array<{ __typename?: 'Function', id?: string | null, name?: string | null, url?: string | null, triggers?: Array<{ __typename?: 'FunctionTrigger', event?: string | null, cron?: string | null }> | null }> | null };


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
    name
    triggers {
      event
      cron
    }
    url
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
  }),
});

export { injectedRtkApi as api };
export const { useGetEventsStreamQuery, useLazyGetEventsStreamQuery, useGetFunctionsStreamQuery, useLazyGetFunctionsStreamQuery, useGetEventQuery, useLazyGetEventQuery, useGetFunctionRunQuery, useLazyGetFunctionRunQuery, useGetFunctionsQuery, useLazyGetFunctionsQuery } = injectedRtkApi;

