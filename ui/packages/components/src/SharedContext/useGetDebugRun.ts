import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import { useShared } from './SharedContext';

export type GetDebugRunPayload = {
  functionSlug: string;
  debugRunID?: string;
  runID?: string;
};

export type RunStep = {
  stepID: string | null;
  name: string;
  stepOp?: string | null;
};

export type RunTraceSpan = {
  runID: string;
  spanID: string;
  traceID: string;
  name: string;
  status: string;
  attempts?: number | null;
  duration?: number | null;
  queuedAt: string;
  startedAt?: string;
  endedAt?: string;
  stepID?: string | null;
  stepOp?: string | null;
  isRoot: boolean;
  parentSpanID?: string | null;
  isUserland: boolean;
  debugRunID?: string | null;
  debugSessionID?: string | null;
  childrenSpans?: RunTraceSpan[] | null;
};

export type DebugRunData = {
  debugRun: RunTraceSpan;
  runSteps?: RunStep[] | null;
};

export type DebugRunResult = {
  error?: Error;
  loading: boolean;
  data?: DebugRunData;
};

type UseGetDebugRunOptions = {
  functionSlug?: string;
  debugRunID?: string;
  runID?: string;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetDebugRun = ({
  functionSlug,
  debugRunID,
  runID,
  refetchInterval,
  enabled = true,
}: UseGetDebugRunOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['debugRun', functionSlug, debugRunID, runID],
    queryFn: useCallback(async () => {
      if (!functionSlug) {
        console.info('no functionSlug provided, skipping getDebugRun');
        return undefined;
      }
      const result = await shared.getDebugRun({ functionSlug, debugRunID, runID });
      if (result.error) {
        throw result.error;
      }
      return result.data;
    }, [shared.getDebugRun, functionSlug, debugRunID, runID]),
    refetchInterval,
    enabled: enabled && !!functionSlug,
  });

  return {
    data: queryResult.data,
    loading: queryResult.isPending,
    error: queryResult.error,
    refetch: queryResult.refetch,
  };
};
