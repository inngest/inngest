import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { Trace } from '../RunDetailsShared/types';
import { useShared } from './SharedContext';

export type GetRunPayload = {
  runID: string;
};

export type GetRunData = {
  app: {
    externalID: string;
    name: string;
  };
  fn: {
    id: string;
    name: string;
    slug: string;
  };
  id: string;
  trace: Trace;
  hasAI: boolean;
  status: string;
  isDurableEndpoint?: boolean;
};

export type GetRunResult = {
  error?: Error;
  loading: boolean;
  data?: GetRunData;
};

type UseGetRunOptions = {
  runID?: string;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetRun = ({ runID, refetchInterval, enabled = true }: UseGetRunOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['run', runID],
    queryFn: useCallback(async () => {
      if (!runID) {
        console.info('no runID provided, skipping getRun');
        return undefined;
      }
      const result = await shared.getRun({ runID });
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
