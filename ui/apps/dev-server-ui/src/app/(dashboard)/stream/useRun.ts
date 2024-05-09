import { useMemo } from 'react';
import type { Event } from '@inngest/components/types/event';
import {
  baseFetchSkipped,
  baseFetchSucceeded,
  baseInitialFetchFailed,
  baseInitialFetchLoading,
  baseRefetchLoading,
  type FetchResult,
} from '@inngest/components/types/fetch';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { HistoryParser } from '@inngest/components/utils/historyParser';

import { useGetFunctionRunQuery } from '@/store/generated';

type Data = {
  func: Pick<Function, 'name' | 'triggers'>;
  history: HistoryParser;
  run: Pick<
    FunctionRun,
    'batchCreatedAt' | 'batchID' | 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'
  > & {
    events: Event[];
  };
};

export function useRun(runID: string | null): FetchResult<Data, { skippable: true }> {
  const skip = !runID;
  const query = useGetFunctionRunQuery(
    { id: runID ?? '' },
    { pollingInterval: 1000, skip, refetchOnMountOrArgChange: true }
  );

  const data = useMemo(() => {
    const rawRun = query.data?.functionRun;
    if (!rawRun || !rawRun.function) {
      return undefined;
    }

    return {
      func: {
        ...rawRun.function,
        triggers: rawRun.function.triggers ?? [],
      },
      history: new HistoryParser(rawRun.history ?? []),
      run: {
        ...rawRun,
        batchCreatedAt: rawRun.batchCreatedAt ? new Date(rawRun.batchCreatedAt) : null,
        endedAt: rawRun.finishedAt,
        events: (rawRun.events ?? []).map((event) => {
          return {
            ...event,
            name: event.name ?? 'Unknown',
            payload: event.raw ?? 'null',
            receivedAt: event.createdAt ? new Date(event.createdAt) : new Date(),
          };
        }),
        status: rawRun.status ?? 'RUNNING',
      },
    } as const;
  }, [query.data?.functionRun]);

  if (query.isLoading) {
    if (!data) {
      return {
        ...baseInitialFetchLoading,
        refetch: query.refetch,
      };
    }

    return {
      ...baseRefetchLoading,
      data,
      refetch: query.refetch,
    };
  }

  if (skip) {
    return baseFetchSkipped;
  }

  if (query.error) {
    return {
      ...baseInitialFetchFailed,
      error: new Error(query.error.message),
      refetch: query.refetch,
    };
  }

  if (!data) {
    // Should be unreachable.
    return {
      ...baseInitialFetchFailed,
      error: new Error('finished loading but missing data'),
      refetch: query.refetch,
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
    refetch: query.refetch,
  };
}
