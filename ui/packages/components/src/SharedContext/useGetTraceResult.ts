import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import { useShared } from './SharedContext';

export type GetTraceResultPayload = {
  traceID: string;
  preview?: boolean;
};

export type TraceResult = {
  input: string | null;
  data: string | null;
  error: {
    message: string;
    name: string | null;
    stack: string | null;
    cause: unknown;
  } | null;
};

type UseGetTraceResultOptions = {
  traceID?: string | null;
  preview?: boolean;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetTraceResult = ({
  traceID,
  preview,
  refetchInterval,
  enabled = true,
}: UseGetTraceResultOptions) => {
  const shared = useShared();

  const isQueryEnabled = enabled && Boolean(traceID);

  const queryResult = useQuery({
    queryKey: ['trace-result', traceID, { preview }],
    queryFn: useCallback(async () => {
      if (!traceID) {
        console.info('no traceID provided, skipping getTraceResult');
        return undefined;
      }
      return await shared.getTraceResult({ traceID, preview });
    }, [shared.getTraceResult, traceID, preview]),
    refetchInterval,
    enabled: isQueryEnabled,
  });

  return {
    data: queryResult.data,
    loading: isQueryEnabled && queryResult.isPending,
    error: queryResult.error,
    refetch: queryResult.refetch,
  };
};
