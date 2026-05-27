import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

import { useShared } from './SharedContext';

export type GetRunLinkagePayload = {
  runID: string;
};

export type RunDeferSummary = {
  id: string;
  userDeferID: string;
  fnSlug: string;
  status: string;
  run: {
    id: string;
    status: string;
    function: { name: string; slug: string };
  } | null;
};

export type RunDeferredFromSummary = {
  parentRunID: string;
  parentRun: {
    id: string;
    status: string;
    function: { name: string; slug: string };
    defers: RunDeferSummary[];
  } | null;
};

export type RunInvokedFromSummary = {
  parentRunID: string;
  parentRun: {
    id: string;
    status: string;
    function: { name: string; slug: string };
  } | null;
  stepName: string | null;
};

export type GetRunLinkageData = {
  defers: RunDeferSummary[];
  // A batched child can descend from several parents, so this is a list.
  deferredFrom: RunDeferredFromSummary[];
  invokedFrom: RunInvokedFromSummary | null;
};

export type GetRunLinkageResult = {
  error?: Error;
  loading: boolean;
  data?: GetRunLinkageData;
};

type UseGetRunLinkageOptions = {
  runID?: string;
  refetchInterval?: number;
  enabled?: boolean;
};

export const useGetRunLinkage = ({
  runID,
  refetchInterval,
  enabled = true,
}: UseGetRunLinkageOptions) => {
  const shared = useShared();

  const queryResult = useQuery({
    queryKey: ['run-linkage', runID],
    queryFn: useCallback(async () => {
      if (!runID || !shared.getRunLinkage) {
        return undefined;
      }
      const result = await shared.getRunLinkage({ runID });
      if (result.error) {
        throw result.error;
      }
      return result.data;
    }, [shared.getRunLinkage, runID]),
    refetchInterval,
    enabled: enabled && Boolean(shared.getRunLinkage),
  });

  return {
    data: queryResult.data,
    loading: queryResult.isPending,
    error: queryResult.error,
    refetch: queryResult.refetch,
  };
};
