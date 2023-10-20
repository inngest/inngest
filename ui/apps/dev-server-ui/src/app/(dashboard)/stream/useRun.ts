import { useMemo } from 'react';
import {
  baseFetchFailed,
  baseFetchLoading,
  baseFetchSkipped,
  baseFetchSucceeded,
  type FetchResult,
} from '@inngest/components/types/fetch';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { HistoryParser } from '@inngest/components/utils/historyParser';

import { useGetFunctionRunQuery } from '@/store/generated';

type Data = {
  func: Pick<Function, 'name' | 'triggers'>;
  history: HistoryParser;
  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
};

export function useRun(runID: string | null): FetchResult<Data, { skippable: true }> {
  const skip = !runID;
  const query = useGetFunctionRunQuery(
    { id: runID ?? '' },
    { pollingInterval: 1000, skip, refetchOnMountOrArgChange: true }
  );

  const data = useMemo((): Data | undefined => {
    const rawRun = query.data?.functionRun;
    if (!rawRun || !rawRun.function) {
      return undefined;
    }

    return {
      func: {
        ...rawRun.function,
        name: rawRun.name ?? 'unknown',
        triggers: rawRun.function.triggers ?? [],
      },
      history: new HistoryParser(rawRun.history ?? []),
      run: {
        ...rawRun,
        endedAt: rawRun.finishedAt,
        status: rawRun.status ?? 'RUNNING',
      },
    } as const;
  }, [query.data?.functionRun]);

  if (query.isLoading) {
    return baseFetchLoading;
  }

  if (skip) {
    return baseFetchSkipped;
  }

  if (query.error) {
    return {
      ...baseFetchFailed,
      error: new Error(query.error.message),
    };
  }

  if (!data) {
    // Should be unreachable.
    return {
      ...baseFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
  };
}
