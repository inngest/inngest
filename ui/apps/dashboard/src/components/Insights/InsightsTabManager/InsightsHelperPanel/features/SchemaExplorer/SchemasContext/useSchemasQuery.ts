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
    getNextPageParam: (lastPage) =>
      lastPage?.pageInfo?.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
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
    error: query.error ? 'Failed to load custom schemas' : null,
    fetchNextPage: query.fetchNextPage,
    hasNextPage: query.hasNextPage ?? false,
    isFetchingNextPage: query.isFetchingNextPage,
    isLoading: query.isPending || (query.isFetching && !query.isFetchingNextPage),
  };
}
