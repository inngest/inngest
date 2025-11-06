'use client';

import { useCallback, useMemo } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { buildSchemaEntriesFromQueryData } from './queries';
import type { SchemaEntry } from './types';
import { useEventTypeSchemas } from './useEventTypeSchemas';

// Hard cap to guard against excessive auto-fetching.
const MAX_SCHEMA_ITEMS = 800;

export function useSchemasQuery(search: string) {
  const isSchemaWidgetEnabled = useBooleanFlag('insights-schema-widget');

  const getEventTypes = useEventTypeSchemas();
  const env = useEnvironment();

  const query = useInfiniteQuery({
    enabled: isSchemaWidgetEnabled.value,
    gcTime: 0,
    queryKey: ['schema-explorer-event-types', env.id, { nameSearch: search || null }],
    queryFn: ({ pageParam }: { pageParam: string | null }) =>
      getEventTypes({ cursor: pageParam, nameSearch: search || null }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.pageInfo.hasNextPage) return undefined;
      return lastPage.pageInfo.endCursor;
    },
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    initialPageParam: null,
  });

  const entriesWithFails = useMemo<(null | SchemaEntry)[]>(
    () => buildSchemaEntriesFromQueryData(query.data),
    [query.data]
  );

  const remoteCount = useMemo(
    () => entriesWithFails.filter((e) => !e?.isShared).length,
    [entriesWithFails]
  );
  const hasFetchedMax = remoteCount >= MAX_SCHEMA_ITEMS;

  const entries: SchemaEntry[] = useMemo(
    () => entriesWithFails.filter((e) => e !== null) as SchemaEntry[],
    [entriesWithFails]
  );

  const guardedFetchNextPage = useCallback(() => {
    if (hasFetchedMax) {
      console.error('Max schemas fetched.');
      return;
    }
    query.fetchNextPage();
  }, [hasFetchedMax, query]);

  return {
    entries,
    error: query.error,
    fetchNextPage: guardedFetchNextPage,
    hasNextPage: query.hasNextPage,
    hasFetchedMax,
    isFetchingNextPage: query.isFetchingNextPage,
    isLoading: query.isPending,
  };
}
