'use client';

import { useMemo } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';

import { useEventTypes } from '@/components/EventTypes/useEventTypes';
import { buildSchemaEntriesFromQueryData } from './queries';
import type { SchemaEntry } from './types';

export function useSchemasQuery(search: string) {
  const getEventTypes = useEventTypes();

  const query = useInfiniteQuery({
    queryKey: ['schema-explorer', { nameSearch: search || null }],
    queryFn: ({ pageParam }: { pageParam: string | null }) =>
      getEventTypes({ archived: false, cursor: pageParam, nameSearch: search || null }),
    getNextPageParam: (lastPage) => {
      if (!lastPage || !lastPage.pageInfo.hasNextPage) return undefined;
      return lastPage.pageInfo.endCursor;
    },
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    initialPageParam: null,
  });

  const entries = useMemo<SchemaEntry[]>(
    () => buildSchemaEntriesFromQueryData(query.data),
    [query.data]
  );

  return {
    entries,
    error: null,
    fetchNextPage: () => {},
    hasNextPage: false,
    isFetchingNextPage: false,
    isLoading: false,
  };
}
