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

  const getEventTypeSchemas = useEventTypeSchemas();
  const env = useEnvironment();

  const query = useInfiniteQuery({
    enabled: isSchemaWidgetEnabled.value,
    queryKey: ['schema-explorer-event-types', env.id, { nameSearch: search || null }],
    queryFn: ({ pageParam }: { pageParam: string | null }) =>
      getEventTypeSchemas({ cursor: pageParam, nameSearch: search || null }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.pageInfo.hasNextPage) return undefined;
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

  const remoteCount = useMemo(() => entries.filter((e) => !e.isShared).length, [entries]);
  const hasFetchedMax = remoteCount >= MAX_SCHEMA_ITEMS;

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
