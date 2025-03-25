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
import type { FunctionRun } from '@inngest/components/types/functionRun';

import { FunctionRunStatus, useGetEventQuery } from '@/store/generated';

type Data = Event & { functionRuns: Pick<FunctionRun, 'id' | 'name' | 'output' | 'status'>[] };

export function useEvent(eventID: string | null): FetchResult<Data, { skippable: true }> {
  const skip = !eventID;
  const query = useGetEventQuery({ id: eventID ?? '' }, { pollingInterval: 1000, skip });

  // In addition to memoizing, this hook will also transform the API data into
  // the shape our shared UI expects.
  const data = useMemo((): Data | undefined => {
    const { event } = query.data ?? {};

    if (!event) {
      return undefined;
    }

    const functionRuns: Data['functionRuns'] = (event.functionRuns ?? []).map((run) => {
      return {
        id: run.id,
        name: run.function?.name ?? 'Unknown',
        output: run.output ?? null,
        status: run.status ?? FunctionRunStatus.Running,
      };
    });

    return {
      ...event,
      functionRuns,
      name: event.name ?? 'Unknown',
      payload: event.raw ?? 'null',
      receivedAt: event.createdAt ? new Date(event.createdAt) : new Date(),
    };
  }, [query.data?.event]);

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
