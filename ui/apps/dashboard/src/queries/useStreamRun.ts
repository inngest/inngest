import { useCallback } from 'react';
import { useTransport } from '@connectrpc/connect-query';
import { createClient } from '@connectrpc/connect';
import { V2 } from '@inngest/components/proto/api/v2/service_pb';
import type {
  RunData,
  RunTraceSpan as ProtoRunTraceSpan,
  StepInfo as ProtoStepInfo,
} from '@inngest/components/proto/api/v2/service_pb';
import {
  RunTraceSpanStatus,
  StepOp,
} from '@inngest/components/proto/api/v2/service_pb';
import type {
  StreamRunPayload,
  StreamRunCallbacks,
  StreamRunData,
  StreamRunHandler,
} from '@inngest/components/SharedContext/useStreamRun';
import type {
  Trace,
  StepInfoInvoke,
  StepInfoSleep,
  StepInfoWait,
  StepInfoRun,
  StepInfoSignal,
  SpanMetadata,
} from '@inngest/components/RunDetailsV3/types';
import { useEnvironment } from '@/components/Environments/environment-context';

//
// Map proto status enum to string
const mapProtoStatus = (status: RunTraceSpanStatus): string => {
  switch (status) {
    case RunTraceSpanStatus.RUNNING:
      return 'RUNNING';
    case RunTraceSpanStatus.COMPLETED:
      return 'COMPLETED';
    case RunTraceSpanStatus.FAILED:
      return 'FAILED';
    case RunTraceSpanStatus.CANCELLED:
      return 'CANCELLED';
    case RunTraceSpanStatus.QUEUED:
      return 'QUEUED';
    case RunTraceSpanStatus.WAITING:
      return 'WAITING';
    default:
      return 'UNKNOWN';
  }
};

//
// Map proto step op enum to string
const mapProtoStepOp = (stepOp?: StepOp): string | null => {
  if (stepOp === undefined) return null;
  switch (stepOp) {
    case StepOp.RUN:
      return 'Run';
    case StepOp.SLEEP:
      return 'Sleep';
    case StepOp.WAIT_FOR_EVENT:
      return 'WaitForEvent';
    case StepOp.INVOKE:
      return 'Invoke';
    case StepOp.AI_GATEWAY:
      return 'AIGateway';
    case StepOp.WAIT_FOR_SIGNAL:
      return 'WaitForSignal';
    default:
      return null;
  }
};

//
// Convert proto step info to our type
const convertStepInfo = (
  protoInfo?: ProtoStepInfo,
):
  | StepInfoInvoke
  | StepInfoSleep
  | StepInfoWait
  | StepInfoRun
  | StepInfoSignal
  | null => {
  if (!protoInfo) return null;

  if (protoInfo.info.case === 'invoke') {
    const invoke = protoInfo.info.value;
    return {
      triggeringEventID: invoke.triggeringEventId,
      functionID: invoke.functionId,
      timeout: invoke.timeout,
      returnEventID: invoke.returnEventId ?? null,
      runID: invoke.runId ?? null,
      timedOut: invoke.timedOut ?? null,
    };
  }
  if (protoInfo.info.case === 'sleep') {
    return { sleepUntil: protoInfo.info.value.sleepUntil };
  }
  if (protoInfo.info.case === 'waitForEvent') {
    const wait = protoInfo.info.value;
    return {
      eventName: wait.eventName,
      expression: wait.expression ?? null,
      timeout: wait.timeout,
      foundEventID: wait.foundEventId ?? null,
      timedOut: wait.timedOut ?? null,
    };
  }
  if (protoInfo.info.case === 'run') {
    return { type: protoInfo.info.value.type ?? null };
  }
  if (protoInfo.info.case === 'waitForSignal') {
    const signal = protoInfo.info.value;
    return {
      signal: signal.signal,
      timeout: signal.timeout,
      timedOut: signal.timedOut ?? null,
    };
  }
  return null;
};

//
// Convert proto trace span to our Trace type
const convertProtoTraceToTrace = (protoSpan: ProtoRunTraceSpan): Trace => ({
  name: protoSpan.name,
  status: mapProtoStatus(protoSpan.status),
  attempts: protoSpan.attempts ?? null,
  queuedAt: protoSpan.queuedAt,
  startedAt: protoSpan.startedAt ?? null,
  endedAt: protoSpan.endedAt ?? null,
  isRoot: protoSpan.isRoot,
  outputID: protoSpan.outputId ?? null,
  spanID: protoSpan.spanId,
  stepID: protoSpan.stepId ?? null,
  stepOp: mapProtoStepOp(protoSpan.stepOp),
  stepType: protoSpan.stepType ?? null,
  childrenSpans: protoSpan.childrenSpans.map(convertProtoTraceToTrace),
  stepInfo: convertStepInfo(protoSpan.stepInfo),
  isUserland: protoSpan.isUserland,
  userlandSpan: protoSpan.userlandSpan
    ? {
        spanName: protoSpan.userlandSpan.spanName ?? null,
        spanKind: protoSpan.userlandSpan.spanKind ?? null,
        serviceName: protoSpan.userlandSpan.serviceName ?? null,
        scopeName: protoSpan.userlandSpan.scopeName ?? null,
        scopeVersion: protoSpan.userlandSpan.scopeVersion ?? null,
        spanAttrs: protoSpan.userlandSpan.spanAttrs ?? null,
        resourceAttrs: protoSpan.userlandSpan.resourceAttrs ?? null,
      }
    : null,
  debugRunID: protoSpan.debugRunId ?? null,
  debugSessionID: protoSpan.debugSessionId ?? null,
  metadata: protoSpan.metadata?.map(
    (m): SpanMetadata => ({
      kind: m.kind as SpanMetadata['kind'],
      scope: m.scope as SpanMetadata['scope'],
      values: JSON.parse(m.values || '{}'),
      updatedAt: m.updatedAt,
    }),
  ),
});

//
// Convert proto RunData to StreamRunData
const convertProtoRunData = (
  protoRun: RunData,
  runID: string,
): StreamRunData => ({
  id: runID,
  status: protoRun.status,
  hasAI: protoRun.hasAi,
  app: {
    name: protoRun.function?.app?.name ?? '',
    externalID: protoRun.function?.app?.externalId ?? '',
  },
  fn: {
    id: protoRun.function?.id ?? '',
    name: protoRun.function?.name ?? '',
    slug: protoRun.function?.slug ?? '',
  },
  trace: protoRun.trace
    ? convertProtoTraceToTrace(protoRun.trace)
    : ({} as Trace),
});

//
// Hook that provides the StreamRunHandler for SharedContext.
export const useStreamRun = (): StreamRunHandler => {
  const transport = useTransport();
  const envID = useEnvironment().id;

  return useCallback(
    (payload: StreamRunPayload, callbacks: StreamRunCallbacks) => {
      if (!payload.runID) {
        callbacks.onError(new Error('runID is required'));
        return () => {};
      }

      if (!envID) {
        callbacks.onError(new Error('envID is required'));
        return () => {};
      }

      const abortController = new AbortController();

      const runStream = async () => {
        try {
          const client = createClient(V2, transport);
          const stream = client.streamRun(
            { envId: envID, runId: payload.runID },
            {
              signal: abortController.signal,
            },
          );

          for await (const item of stream) {
            if (item.run) {
              callbacks.onData(convertProtoRunData(item.run, payload.runID));
            }
          }

          callbacks.onComplete();
        } catch (err) {
          if (abortController.signal.aborted) {
            callbacks.onComplete();
            return;
          }

          callbacks.onError(
            err instanceof Error ? err : new Error(String(err)),
          );
        }
      };

      runStream();

      //
      // Return cleanup function
      return () => {
        abortController.abort();
      };
    },
    [transport, envID],
  );
};
