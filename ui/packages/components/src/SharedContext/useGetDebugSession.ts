import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { Trace } from '../RunDetailsV3/types';
import { useShared } from './SharedContext';

export type GetDebugSessionPayload = {
  functionSlug: string;
  debugSessionID?: string;
  runID?: string;
};

export type DebugSessionRun = {
  status: Trace['status'];
  queuedAt: Trace['queuedAt'];
  startedAt: Trace['startedAt'];
  endedAt: Trace['endedAt'];
  debugRunID: Trace['debugRunID'];
  tags: [string];
  versions: [string];
};

export type DebugSessionResult = {
  error?: Error;
  loading: boolean;
  data?: {
    debugRuns: DebugSessionRun[];
  };
};

type UseGetDebugSessionOptions = {
  functionSlug?: string;
  debugSessionID?: string;
  runID?: string;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetDebugSession = ({
  functionSlug,
  debugSessionID,
  runID,
  refetchInterval,
  enabled = true,
}: UseGetDebugSessionOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['debugSession', functionSlug, debugSessionID, runID],
    queryFn: useCallback(async () => {
      if (!functionSlug) {
        console.info('no functionSlug provided, skipping getDebugSession');
        return undefined;
      }
      const result = await shared.getDebugSession({ functionSlug, debugSessionID, runID });
      if (result.error) {
        throw result.error;
      }

      return result.data || { debugRuns: [] };
    }, [shared.getDebugSession, functionSlug, debugSessionID, runID]),
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
