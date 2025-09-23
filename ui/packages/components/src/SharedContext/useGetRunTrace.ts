import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { Trace } from '../RunDetailsV3/types';
import { useShared } from './SharedContext';

export type GetRunTracePayload = {
  runID: string;
};

export type GetRunTraceResult = {
  error?: Error;
  loading: boolean;
  data?: Trace;
};

type UseGetRunTraceOptions = {
  runID?: string;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetRunTrace = ({
  runID,
  refetchInterval,
  enabled = true,
}: UseGetRunTraceOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['runTrace', runID],
    queryFn: useCallback(async () => {
      if (!runID) {
        console.info('no runID provided, skipping getRunTrace');
        return undefined;
      }
      const result = await shared.getRunTrace({ runID });
      if (result.error) {
        throw result.error;
      }
      return result.data;
    }, [shared.getRun, runID]),
    refetchInterval,
    enabled,
  });

  return {
    data: queryResult.data,
    loading: queryResult.isPending,
    error: queryResult.error,
    refetch: queryResult.refetch,
  };
};
