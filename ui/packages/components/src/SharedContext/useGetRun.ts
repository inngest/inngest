import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { Trace } from '../RunDetailsV3/types';
import { useShared } from './SharedContext';

export type GetRunPayload = {
  runID: string;

  /**
   * If `true`, traces will be fetched using the incoming tracing data.
   */
  preview?: boolean;
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
};

export type GetRunResult = {
  error?: Error;
  loading: boolean;
  data?: GetRunData;
};

type UseGetRunOptions = {
  runID?: string;
  preview?: boolean;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetRun = ({
  runID,
  preview,
  refetchInterval,
  enabled = true,
}: UseGetRunOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['run', runID, { preview }],
    queryFn: useCallback(async () => {
      if (!runID) {
        console.info('no runID provided, skipping getRun');
        return undefined;
      }
      const result = await shared.getRun({ runID, preview });
      if (result.error) {
        throw result.error;
      }
      return result.data;
    }, [shared.getRun, runID, preview]),
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
