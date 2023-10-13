import { useMemo } from 'react';
import type { Event } from '@inngest/components/types/event';
import type { FetchResult } from '@inngest/components/types/fetch';
import type { FunctionRun } from '@inngest/components/types/functionRun';

import { FunctionRunStatus, useGetEventQuery } from '@/store/generated';

type Data = Event & { functionRuns: FunctionRun[] };

export function useEvent(eventID: string): FetchResult<Data> {
  const query = useGetEventQuery({ id: eventID }, { pollingInterval: 1500 });

  // In addition to memoizing, this hook will also transform the API data into
  // the shape our shared UI expects.
  const data = useMemo((): Data | undefined => {
    const { event } = query.data ?? {};

    if (!event) {
      return undefined;
    }

    const functionRuns: FunctionRun[] = (event.functionRuns ?? []).map((run) => {
      return {
        ...run,
        name: run.name ?? 'Unknown',
        output: run.output ?? undefined,
        status: run.status ?? FunctionRunStatus.Running,
      };
    });

    return {
      ...event,
      createdAt: event.createdAt ? new Date(event.createdAt) : new Date(),
      functionRuns,
      payload: event.raw ?? 'null',
      name: event.name ?? 'Unknown',
    };
  }, [query.data?.event]);

  if (query.isLoading) {
    return { data: undefined, error: undefined, isLoading: true };
  }

  if (!data) {
    return {
      data: undefined,
      error: new Error('finished loading but missing event'),
      isLoading: false,
    };
  }

  return { data, error: undefined, isLoading: false };
}
