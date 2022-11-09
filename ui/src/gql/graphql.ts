/* eslint-disable */
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
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
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  payload?: Maybe<Scalars['String']>;
  schema?: Maybe<Scalars['String']>;
  timeline?: Maybe<EventTimeline>;
};

export type EventTimeline = {
  __typename?: 'EventTimeline';
  event: Event;
  functionRuns?: Maybe<Array<FunctionRun>>;
};

export type EventTimelineQuery = {
  eventId: Scalars['ID'];
};

export type EventsQuery = {
  lastEventId?: InputMaybe<Scalars['ID']>;
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

export type FunctionRun = {
  __typename?: 'FunctionRun';
  functionVersion?: Maybe<FunctionVersion>;
  id: Scalars['ID'];
  startedAt?: Maybe<Scalars['Time']>;
  status?: Maybe<Scalars['String']>;
  steps?: Maybe<Array<FunctionRunStep>>;
};

export type FunctionRunQuery = {
  functionRunId: Scalars['ID'];
};

export type FunctionRunStep = {
  __typename?: 'FunctionRunStep';
  functionRun?: Maybe<FunctionRun>;
  output?: Maybe<Scalars['String']>;
  startedAt?: Maybe<Scalars['Time']>;
  status?: Maybe<Scalars['String']>;
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
  eventTimeline?: Maybe<EventTimeline>;
  events?: Maybe<Array<Event>>;
  functionRun?: Maybe<FunctionRun>;
};


export type QueryActionVersionArgs = {
  query: ActionVersionQuery;
};


export type QueryEventTimelineArgs = {
  query: EventTimelineQuery;
};


export type QueryEventsArgs = {
  query: EventsQuery;
};


export type QueryFunctionRunArgs = {
  query: FunctionRunQuery;
};

export type UpdateActionVersionInput = {
  dsn: Scalars['String'];
  enabled?: InputMaybe<Scalars['Boolean']>;
  versionMajor: Scalars['Int'];
  versionMinor: Scalars['Int'];
};

export type GetEventsStreamQueryVariables = Exact<{
  query: EventsQuery;
}>;


export type GetEventsStreamQuery = { __typename?: 'Query', events?: Array<{ __typename?: 'Event', id: string, name?: string | null, createdAt?: any | null, payload?: string | null }> | null };


export const GetEventsStreamDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetEventsStream"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"EventsQuery"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"events"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}}]}}]}}]} as unknown as DocumentNode<GetEventsStreamQuery, GetEventsStreamQueryVariables>;