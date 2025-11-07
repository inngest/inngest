'use client';

import { useCallback, useMemo, useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { buildSchemaEntriesFromQueryData } from './queries';
import type { SchemaEntry } from './types';
import { useEventTypeSchemas } from './useEventTypeSchemas';

// Hard cap to guard against excessive auto-fetching. Right now, there are sometimes
// issues presenting schemas. The code in place should make sure that fetched schemas
// always "take up vertical space" to make sure that the loading trigger gets pushed
// down sufficiently, but we'll put a hard cap on the number of fetches to be safe
// until things are sufficiently stable.
//
// NOTE: This number is basically a cap on the number of fetches while the user is in
// a single "Insights" session, including debounced search calls. This is why it's much
// higher than MAX_SCHEMA_ITEMS / <page size>. Unlike MAX_SCHEMA_ITEMS, this caps should
// not be hit during reasonable use.
const MAX_FETCHES = 150;

// Hard cap to guard against excessive fetching and to encourage the use of search.
const MAX_SCHEMA_ITEMS = 800;

export function useSchemasQuery(search: string) {
  const isSchemaWidgetEnabled = useBooleanFlag('insights-schema-widget');

  const [numFetches, setNumFetches] = useState(0);

  const getEventTypeSchemas = useEventTypeSchemas();
  const env = useEnvironment();

  const query = useInfiniteQuery({
    enabled: isSchemaWidgetEnabled.value && numFetches < MAX_FETCHES,
    queryKey: ['schema-explorer-event-types', env.id, { nameSearch: search || null }],
    queryFn: ({ pageParam }: { pageParam: string | null }) => {
      setNumFetches((numFetches) => numFetches + 1);
      return getEventTypeSchemas({ cursor: pageParam, nameSearch: search || null });
    },
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
  const hasFetchedMax = remoteCount >= MAX_SCHEMA_ITEMS || numFetches >= MAX_FETCHES;

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
