/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import type * as Types from './generated-types';

import type { SpanMetadataKind, SpanMetadataScope } from '@inngest/components/RunDetailsV3/types';
import { api } from './baseApi';
export * from './generated-types';
export type GetEventQueryVariables = Exact<{
  id: string | number;
}>;


export type GetEventQuery = { __typename: 'Query', event: { __typename: 'Event', id: string, name: string | null, createdAt: string | null, status: Types.EventStatus | null, pendingRuns: number | null, raw: string | null, functionRuns: Array<{ __typename: 'FunctionRun', id: string, status: Types.FunctionRunStatus | null, startedAt: string | null, pendingSteps: number | null, output: string | null, function: { __typename: 'Function', name: string } | null, waitingFor: { __typename: 'StepEventWait', expiryTime: string, eventName: string | null, expression: string | null } | null }> | null } | null };

export type GetFunctionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetFunctionsQuery = { __typename: 'Query', functions: Array<{ __typename: 'Function', id: string, slug: string, name: string, url: string, triggers: Array<{ __typename: 'FunctionTrigger', type: Types.FunctionTriggerTypes, value: string }> | null, app: { __typename: 'App', name: string, method: Types.AppMethod } }> | null };

export type GetFunctionQueryVariables = Exact<{
  functionSlug: string;
}>;


export type GetFunctionQuery = { __typename: 'Query', functionBySlug: { __typename: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string, failureHandler: { __typename: 'Function', slug: string } | null, configuration: { __typename: 'FunctionConfiguration', priority: string | null, cancellations: Array<{ __typename: 'CancellationConfiguration', event: string, timeout: string | null, condition: string | null }>, retries: { __typename: 'RetryConfiguration', value: number, isDefault: boolean | null }, eventsBatch: { __typename: 'EventsBatchConfiguration', maxSize: number, timeout: string, key: string | null } | null, concurrency: Array<{ __typename: 'ConcurrencyConfiguration', scope: Types.ConcurrencyScope, key: string | null, limit: { __typename: 'ConcurrencyLimitConfiguration', value: number, isPlanLimit: boolean | null } }>, rateLimit: { __typename: 'RateLimitConfiguration', limit: number, period: string, key: string | null } | null, debounce: { __typename: 'DebounceConfiguration', period: string, key: string | null } | null, throttle: { __typename: 'ThrottleConfiguration', burst: number, key: string | null, limit: number, period: string } | null, singleton: { __typename: 'SingletonConfiguration', key: string | null, mode: Types.SingletonMode } | null }, triggers: Array<{ __typename: 'FunctionTrigger', type: Types.FunctionTriggerTypes, value: string, condition: string | null }> | null, app: { __typename: 'App', name: string, method: Types.AppMethod } } | null };

export type GetAppsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAppsQuery = { __typename: 'Query', apps: Array<{ __typename: 'App', id: string, name: string, appVersion: string | null, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: Types.AppMethod, functions: Array<{ __typename: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string }> }> };

export type GetAppQueryVariables = Exact<{
  id: string;
}>;


export type GetAppQuery = { __typename: 'Query', app: { __typename: 'App', id: string, name: string, appVersion: string | null, sdkLanguage: string, sdkVersion: string, framework: string | null, url: string | null, error: string | null, connected: boolean, functionCount: number, autodiscovered: boolean, method: Types.AppMethod, functions: Array<{ __typename: 'Function', name: string, id: string, concurrency: number, config: string, slug: string, url: string, triggers: Array<{ __typename: 'FunctionTrigger', type: Types.FunctionTriggerTypes, value: string }> | null }> } | null };

export type CreateAppMutationVariables = Exact<{
  input: Types.CreateAppInput;
}>;


export type CreateAppMutation = { __typename: 'Mutation', createApp: { __typename: 'App', url: string | null } };

export type UpdateAppMutationVariables = Exact<{
  input: Types.UpdateAppInput;
}>;


export type UpdateAppMutation = { __typename: 'Mutation', updateApp: { __typename: 'App', url: string | null, id: string } };

export type DeleteAppMutationVariables = Exact<{
  id: string;
}>;


export type DeleteAppMutation = { __typename: 'Mutation', deleteApp: string };

export type InvokeFunctionMutationVariables = Exact<{
  functionSlug: string;
  data?: Record<string, unknown> | null | undefined;
  user?: Record<string, unknown> | null | undefined;
  debugSessionID?: string | null | undefined;
  debugRunID?: string | null | undefined;
}>;


export type InvokeFunctionMutation = { __typename: 'Mutation', invokeFunction: boolean | null };

export type CancelRunMutationVariables = Exact<{
  runID: string;
}>;


export type CancelRunMutation = { __typename: 'Mutation', cancelRun: { __typename: 'FunctionRun', id: string } };

export type RerunMutationVariables = Exact<{
  runID: string;
  debugRunID?: string | null | undefined;
  debugSessionID?: string | null | undefined;
}>;


export type RerunMutation = { __typename: 'Mutation', rerun: string };

export type RerunFromStepMutationVariables = Exact<{
  runID: string;
  fromStep: Types.RerunFromStepInput;
  debugRunID?: string | null | undefined;
  debugSessionID?: string | null | undefined;
}>;


export type RerunFromStepMutation = { __typename: 'Mutation', rerun: string };

export type GetRunsQueryVariables = Exact<{
  appIDs?: Array<string> | string | null | undefined;
  startTime: string;
  status?: Array<Types.FunctionRunStatus> | Types.FunctionRunStatus | null | undefined;
  timeField: Types.RunsV2OrderByField;
  functionRunCursor?: string | null | undefined;
  celQuery?: string | null | undefined;
  preview?: boolean | null | undefined;
}>;


export type GetRunsQuery = { __typename: 'Query', runs: { __typename: 'RunsV2Connection', edges: Array<{ __typename: 'FunctionRunV2Edge', node: { __typename: 'FunctionRunV2', cronSchedule: string | null, eventName: string | null, id: string, isBatch: boolean, queuedAt: string, endedAt: string | null, startedAt: string | null, status: Types.FunctionRunStatus, hasAI: boolean, app: { __typename: 'App', externalID: string, name: string }, function: { __typename: 'Function', name: string, slug: string } } }>, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountRunsQueryVariables = Exact<{
  startTime: string;
  status?: Array<Types.FunctionRunStatus> | Types.FunctionRunStatus | null | undefined;
  timeField: Types.RunsV2OrderByField;
  preview?: boolean | null | undefined;
}>;


export type CountRunsQuery = { __typename: 'Query', runs: { __typename: 'RunsV2Connection', totalCount: number } };

export type TraceDetailsFragment = { __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
    | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
    | { __typename: 'RunStepInfo', type: string | null }
    | { __typename: 'SleepStepInfo', sleepUntil: string }
    | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
    | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
   | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null };

export type GetRunQueryVariables = Exact<{
  runID: string;
  preview?: boolean | null | undefined;
}>;


export type GetRunQuery = { __typename: 'Query', run: { __typename: 'FunctionRunV2', status: Types.FunctionRunStatus, hasAI: boolean, function: { __typename: 'Function', id: string, name: string, slug: string, app: { __typename: 'App', name: string, method: Types.AppMethod } }, trace: { __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
                | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
                | { __typename: 'RunStepInfo', type: string | null }
                | { __typename: 'SleepStepInfo', sleepUntil: string }
                | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
                | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
               | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
              | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
              | { __typename: 'RunStepInfo', type: string | null }
              | { __typename: 'SleepStepInfo', sleepUntil: string }
              | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
              | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
             | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
            | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
            | { __typename: 'RunStepInfo', type: string | null }
            | { __typename: 'SleepStepInfo', sleepUntil: string }
            | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
            | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
           | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
          | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
          | { __typename: 'RunStepInfo', type: string | null }
          | { __typename: 'SleepStepInfo', sleepUntil: string }
          | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
          | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
         | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
        | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
        | { __typename: 'RunStepInfo', type: string | null }
        | { __typename: 'SleepStepInfo', sleepUntil: string }
        | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
        | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
       | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null } | null } | null };

export type GetRunTraceQueryVariables = Exact<{
  runID: string;
}>;


export type GetRunTraceQuery = { __typename: 'Query', runTrace: { __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
              | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
              | { __typename: 'RunStepInfo', type: string | null }
              | { __typename: 'SleepStepInfo', sleepUntil: string }
              | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
              | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
             | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
            | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
            | { __typename: 'RunStepInfo', type: string | null }
            | { __typename: 'SleepStepInfo', sleepUntil: string }
            | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
            | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
           | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
          | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
          | { __typename: 'RunStepInfo', type: string | null }
          | { __typename: 'SleepStepInfo', sleepUntil: string }
          | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
          | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
         | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
        | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
        | { __typename: 'RunStepInfo', type: string | null }
        | { __typename: 'SleepStepInfo', sleepUntil: string }
        | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
        | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
       | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
      | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
      | { __typename: 'RunStepInfo', type: string | null }
      | { __typename: 'SleepStepInfo', sleepUntil: string }
      | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
      | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
     | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null } };

export type GetTraceResultQueryVariables = Exact<{
  traceID: string;
}>;


export type GetTraceResultQuery = { __typename: 'Query', runTraceSpanOutputByID: { __typename: 'RunTraceSpanOutput', input: string | null, data: string | null, error: { __typename: 'StepError', message: string, name: string | null, stack: string | null, cause: unknown } | null } };

export type GetTriggerQueryVariables = Exact<{
  runID: string;
}>;


export type GetTriggerQuery = { __typename: 'Query', runTrigger: { __typename: 'RunTraceTrigger', IDs: Array<string>, payloads: Array<string>, timestamp: string, eventName: string | null, isBatch: boolean, batchID: string | null, cron: string | null } };

export type GetWorkerConnectionsQueryVariables = Exact<{
  appID: string;
  startTime?: string | null | undefined;
  status?: Array<Types.ConnectV1ConnectionStatus> | Types.ConnectV1ConnectionStatus | null | undefined;
  timeField: Types.ConnectV1WorkerConnectionsOrderByField;
  cursor?: string | null | undefined;
  orderBy?: Array<Types.ConnectV1WorkerConnectionsOrderBy> | Types.ConnectV1WorkerConnectionsOrderBy | null | undefined;
  first: number;
}>;


export type GetWorkerConnectionsQuery = { __typename: 'Query', workerConnections: { __typename: 'ConnectV1WorkerConnectionsConnection', totalCount: number, edges: Array<{ __typename: 'ConnectV1WorkerConnectionEdge', node: { __typename: 'ConnectV1WorkerConnection', id: string, gatewayId: string, instanceId: string, workerIp: string, maxWorkerConcurrency: number, connectedAt: string, lastHeartbeatAt: string | null, disconnectedAt: string | null, disconnectReason: string | null, status: Types.ConnectV1ConnectionStatus, groupHash: string, sdkLang: string, sdkVersion: string, sdkPlatform: string, syncId: string | null, appVersion: string | null, functionCount: number, cpuCores: number, memBytes: number, os: string, app: { __typename: 'App', id: string } | null } }>, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, hasPreviousPage: boolean, startCursor: string | null, endCursor: string | null } } };

export type CountWorkerConnectionsQueryVariables = Exact<{
  appID: string;
  startTime: string;
  status?: Array<Types.ConnectV1ConnectionStatus> | Types.ConnectV1ConnectionStatus | null | undefined;
}>;


export type CountWorkerConnectionsQuery = { __typename: 'Query', workerConnections: { __typename: 'ConnectV1WorkerConnectionsConnection', totalCount: number } };

export type GetEventsV2QueryVariables = Exact<{
  cursor?: string | null | undefined;
  startTime: string;
  endTime?: string | null | undefined;
  celQuery?: string | null | undefined;
  eventNames?: Array<string> | string | null | undefined;
  includeInternalEvents?: boolean | null | undefined;
}>;


export type GetEventsV2Query = { __typename: 'Query', eventsV2: { __typename: 'EventsConnection', totalCount: number, edges: Array<{ __typename: 'EventsEdge', node: { __typename: 'EventV2', name: string, id: string, receivedAt: string, runs: Array<{ __typename: 'FunctionRunV2', status: Types.FunctionRunStatus, id: string, startedAt: string | null, endedAt: string | null, function: { __typename: 'Function', name: string, slug: string } }> } }>, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, endCursor: string | null, hasPreviousPage: boolean, startCursor: string | null } } };

export type GetEventV2QueryVariables = Exact<{
  eventID: string;
}>;


export type GetEventV2Query = { __typename: 'Query', eventV2: { __typename: 'EventV2', name: string, id: string, receivedAt: string, idempotencyKey: string | null, occurredAt: string, version: string | null, source: { __typename: 'EventSource', name: string | null } | null } };

export type GetEventV2PayloadQueryVariables = Exact<{
  eventID: string;
}>;


export type GetEventV2PayloadQuery = { __typename: 'Query', eventV2: { __typename: 'EventV2', raw: string } };

export type GetEventV2RunsQueryVariables = Exact<{
  eventID: string;
}>;


export type GetEventV2RunsQuery = { __typename: 'Query', eventV2: { __typename: 'EventV2', name: string, runs: Array<{ __typename: 'FunctionRunV2', status: Types.FunctionRunStatus, id: string, startedAt: string | null, endedAt: string | null, function: { __typename: 'Function', name: string, slug: string }, trace: { __typename: 'RunTraceSpan', skipReason: string | null, skipExistingRunID: string | null } | null }> } };

export type CreateDebugSessionMutationVariables = Exact<{
  input: Types.CreateDebugSessionInput;
}>;


export type CreateDebugSessionMutation = { __typename: 'Mutation', createDebugSession: { __typename: 'CreateDebugSessionResponse', debugSessionID: string, debugRunID: string } };

export type GetDebugRunQueryVariables = Exact<{
  query: Types.DebugRunQuery;
}>;


export type GetDebugRunQuery = { __typename: 'Query', debugRun: { __typename: 'DebugRun', debugTraces: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, childrenSpans: Array<{ __typename: 'RunTraceSpan', name: string, status: Types.RunTraceSpanStatus, attempts: number | null, queuedAt: string, startedAt: string | null, endedAt: string | null, isRoot: boolean, isUserland: boolean, outputID: string | null, debugRunID: string | null, debugSessionID: string | null, spanID: string, stepID: string | null, stepOp: Types.StepOp | null, stepType: string, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
              | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
              | { __typename: 'RunStepInfo', type: string | null }
              | { __typename: 'SleepStepInfo', sleepUntil: string }
              | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
              | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
             | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
            | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
            | { __typename: 'RunStepInfo', type: string | null }
            | { __typename: 'SleepStepInfo', sleepUntil: string }
            | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
            | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
           | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
          | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
          | { __typename: 'RunStepInfo', type: string | null }
          | { __typename: 'SleepStepInfo', sleepUntil: string }
          | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
          | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
         | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }>, userlandSpan: { __typename: 'UserlandSpan', spanName: string | null, spanKind: string | null, serviceName: string | null, scopeName: string | null, scopeVersion: string | null, spanAttrs: string | null, resourceAttrs: string | null } | null, metadata: Array<{ __typename: 'SpanMetadata', scope: SpanMetadataScope, kind: SpanMetadataKind, values: Record<string, unknown>, updatedAt: string }>, stepInfo:
        | { __typename: 'InvokeStepInfo', triggeringEventID: string, functionID: string, timeout: string, returnEventID: string | null, runID: string | null, timedOut: boolean | null }
        | { __typename: 'RunStepInfo', type: string | null }
        | { __typename: 'SleepStepInfo', sleepUntil: string }
        | { __typename: 'WaitForEventStepInfo', eventName: string, expression: string | null, timeout: string, foundEventID: string | null, timedOut: boolean | null }
        | { __typename: 'WaitForSignalStepInfo', signal: string, timeout: string, timedOut: boolean | null }
       | null, response: { __typename: 'RunTraceSpanResponseInfo', statusCode: number, headers: Record<string, string|string[]> } | null }> | null } | null };

export type GetDebugSessionQueryVariables = Exact<{
  query: Types.DebugSessionQuery;
}>;


export type GetDebugSessionQuery = { __typename: 'Query', debugSession: { __typename: 'DebugSession', debugRuns: Array<{ __typename: 'DebugSessionRun', status: Types.RunTraceSpanStatus, queuedAt: string, startedAt: string | null, endedAt: string | null, debugRunID: string | null, tags: Array<string> | null, versions: Array<string> | null }> | null } | null };

// TypedDocumentString is typed as a constructor returning `string` so that
// the generated document constants are assignable to RTK Query's expected
// `string | DocumentNode` base-query argument.
// At runtime the String constructor is used, whose wrapper objects coerce to
// primitive strings wherever graphql-request needs them.
// eslint-disable-next-line @typescript-eslint/no-redeclare
const TypedDocumentString = String as unknown as new <TResult, TVariables>(
  value: string,
  meta?: Record<string, any>,
) => string;
export const TraceDetailsFragmentDoc = new TypedDocumentString(`
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
  metadata {
    scope
    kind
    values
    updatedAt
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
  response {
    statusCode
    headers
  }
}
    `, {"fragmentName":"TraceDetails"});
export const GetEventDocument = new TypedDocumentString(`
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
    `);
export const GetFunctionsDocument = new TypedDocumentString(`
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
      method
    }
    url
  }
}
    `);
export const GetFunctionDocument = new TypedDocumentString(`
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
      method
    }
    url
  }
}
    `);
export const GetAppsDocument = new TypedDocumentString(`
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
    `);
export const GetAppDocument = new TypedDocumentString(`
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
    `);
export const CreateAppDocument = new TypedDocumentString(`
    mutation CreateApp($input: CreateAppInput!) {
  createApp(input: $input) {
    url
  }
}
    `);
export const UpdateAppDocument = new TypedDocumentString(`
    mutation UpdateApp($input: UpdateAppInput!) {
  updateApp(input: $input) {
    url
    id
  }
}
    `);
export const DeleteAppDocument = new TypedDocumentString(`
    mutation DeleteApp($id: String!) {
  deleteApp(id: $id)
}
    `);
export const InvokeFunctionDocument = new TypedDocumentString(`
    mutation InvokeFunction($functionSlug: String!, $data: Map, $user: Map, $debugSessionID: ULID = null, $debugRunID: ULID = null) {
  invokeFunction(
    data: $data
    functionSlug: $functionSlug
    user: $user
    debugSessionID: $debugSessionID
    debugRunID: $debugRunID
  )
}
    `);
export const CancelRunDocument = new TypedDocumentString(`
    mutation CancelRun($runID: ULID!) {
  cancelRun(runID: $runID) {
    id
  }
}
    `);
export const RerunDocument = new TypedDocumentString(`
    mutation Rerun($runID: ULID!, $debugRunID: ULID = null, $debugSessionID: ULID = null) {
  rerun(runID: $runID, debugRunID: $debugRunID, debugSessionID: $debugSessionID)
}
    `);
export const RerunFromStepDocument = new TypedDocumentString(`
    mutation RerunFromStep($runID: ULID!, $fromStep: RerunFromStepInput!, $debugRunID: ULID = null, $debugSessionID: ULID = null) {
  rerun(
    runID: $runID
    fromStep: $fromStep
    debugRunID: $debugRunID
    debugSessionID: $debugSessionID
  )
}
    `);
export const GetRunsDocument = new TypedDocumentString(`
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
    `);
export const CountRunsDocument = new TypedDocumentString(`
    query CountRuns($startTime: Time!, $status: [FunctionRunStatus!], $timeField: RunsV2OrderByField!, $preview: Boolean = false) {
  runs(
    filter: {from: $startTime, status: $status, timeField: $timeField}
    orderBy: [{field: $timeField, direction: DESC}]
    preview: $preview
  ) {
    totalCount(preview: $preview)
  }
}
    `);
export const GetRunDocument = new TypedDocumentString(`
    query GetRun($runID: String!, $preview: Boolean) {
  run(runID: $runID) {
    function {
      app {
        name
        method
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
  metadata {
    scope
    kind
    values
    updatedAt
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
  response {
    statusCode
    headers
  }
}`);
export const GetRunTraceDocument = new TypedDocumentString(`
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
  metadata {
    scope
    kind
    values
    updatedAt
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
  response {
    statusCode
    headers
  }
}`);
export const GetTraceResultDocument = new TypedDocumentString(`
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
    `);
export const GetTriggerDocument = new TypedDocumentString(`
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
    `);
export const GetWorkerConnectionsDocument = new TypedDocumentString(`
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
    `);
export const CountWorkerConnectionsDocument = new TypedDocumentString(`
    query CountWorkerConnections($appID: UUID!, $startTime: Time!, $status: [ConnectV1ConnectionStatus!]) {
  workerConnections(
    filter: {appIDs: [$appID], from: $startTime, status: $status, timeField: CONNECTED_AT}
    orderBy: [{field: CONNECTED_AT, direction: DESC}]
  ) {
    totalCount
  }
}
    `);
export const GetEventsV2Document = new TypedDocumentString(`
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
    `);
export const GetEventV2Document = new TypedDocumentString(`
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
    `);
export const GetEventV2PayloadDocument = new TypedDocumentString(`
    query GetEventV2Payload($eventID: ULID!) {
  eventV2(id: $eventID) {
    raw
  }
}
    `);
export const GetEventV2RunsDocument = new TypedDocumentString(`
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
      trace(preview: true) {
        skipReason
        skipExistingRunID
      }
    }
  }
}
    `);
export const CreateDebugSessionDocument = new TypedDocumentString(`
    mutation CreateDebugSession($input: CreateDebugSessionInput!) {
  createDebugSession(input: $input) {
    debugSessionID
    debugRunID
  }
}
    `);
export const GetDebugRunDocument = new TypedDocumentString(`
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
  metadata {
    scope
    kind
    values
    updatedAt
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
  response {
    statusCode
    headers
  }
}`);
export const GetDebugSessionDocument = new TypedDocumentString(`
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
    `);

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

