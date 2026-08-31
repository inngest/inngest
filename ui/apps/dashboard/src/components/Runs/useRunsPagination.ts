import { useCallback, useEffect, useMemo, useState } from 'react';
import type { Run } from '@inngest/components/RunsPage/types';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useQuery } from 'urql';

import { GetRunsDocument } from './queries';
import {
  fetchRunsPage,
  restFunctionRunToTableRun,
  type RestRunsPage,
} from './restRuns';
import { parseRunsData } from './utils';

type UseRunsPaginationParams = {
  commonQueryVars: {
    appIDs: string[] | null;
    environmentID: string;
    functionSlug: string | null;
    startTime: string;
    endTime: string | null;
    status: any[] | null;
    timeField: any;
    celQuery: string | undefined;
    isDeferred: boolean | null;
    environmentSlug: string;
    functionAppID: string | null;
  };
  tracePreviewEnabled: boolean;
  useREST: boolean;
};

export function useRunsPagination({
  commonQueryVars,
  tracePreviewEnabled,
  useREST,
}: UseRunsPaginationParams) {
  const [cursor, setCursor] = useState<string | null>(null);
  const [allRuns, setAllRuns] = useState<Run[]>([]);

  const [queryRes, refetch] = useQuery({
    pause: useREST,
    query: GetRunsDocument,
    requestPolicy: 'network-only',
    variables: {
      ...commonQueryVars,
      functionRunCursor: cursor,
      preview: tracePreviewEnabled,
    },
  });

  const restQuery = useInfiniteQuery({
    enabled: useREST,
    queryKey: ['runs-rest-v2', commonQueryVars],
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      fetchRestRuns(commonQueryVars, pageParam, signal),
    getNextPageParam: (lastPage) =>
      lastPage.page.hasMore ? lastPage.page.cursor : undefined,
    refetchInterval: commonQueryVars.celQuery ? false : 1000,
  });

  const restRuns = useMemo(() => {
    const byID = new Map<string, Run>();
    for (const page of restQuery.data?.pages ?? []) {
      for (const run of page.data) {
        const mapped = restFunctionRunToTableRun(run);
        byID.set(mapped.id, mapped);
      }
    }
    return [...byID.values()];
  }, [restQuery.data?.pages]);

  const newRuns = useMemo(() => {
    return parseRunsData(queryRes.data?.environment.runs.edges);
  }, [queryRes.data?.environment.runs.edges]);

  const pageInfo = queryRes.data?.environment.runs.pageInfo;
  const hasNextPage = pageInfo?.hasNextPage ?? false;

  // Create a stable stringified version of commonQueryVars for dependency tracking
  const queryVarsKey = useMemo(
    () => JSON.stringify(commonQueryVars),
    [commonQueryVars],
  );

  // When new data comes in, either replace (first page) or append (subsequent pages)
  useEffect(() => {
    if (newRuns.length > 0) {
      if (cursor === null) {
        // First page - replace all runs
        setAllRuns(newRuns);
      } else {
        // Subsequent pages - append only if we don't already have this data
        setAllRuns((prev) => {
          // Check if we already appended this page (avoid duplicates)
          const firstNewRun = newRuns[0];
          if (
            prev.length > 0 &&
            firstNewRun &&
            prev.some((r) => r.id === firstNewRun.id)
          ) {
            return prev;
          }
          return [...prev, ...newRuns];
        });
      }
    }
  }, [newRuns, cursor]);

  // Reset when filter variables change
  useEffect(() => {
    setCursor(null);
    setAllRuns([]);
  }, [queryVarsKey]);

  const loadMore = useCallback(() => {
    if (useREST) {
      if (!restQuery.isFetching && restQuery.hasNextPage) {
        void restQuery.fetchNextPage();
      }
      return;
    }
    if (!queryRes.fetching && hasNextPage && pageInfo?.endCursor) {
      setCursor(pageInfo.endCursor);
    }
  }, [
    useREST,
    restQuery.isFetching,
    restQuery.hasNextPage,
    restQuery.fetchNextPage,
    queryRes.fetching,
    hasNextPage,
    pageInfo?.endCursor,
  ]);

  const reset = useCallback(() => {
    if (useREST) {
      void restQuery.refetch();
      return;
    }
    setCursor(null);
    setAllRuns([]);
    refetch();
  }, [refetch, restQuery.refetch, useREST]);

  if (useREST) {
    return {
      runs: restRuns,
      isLoading: restQuery.isFetching,
      isLoadingInitial: restQuery.isLoading,
      isLoadingMore: restQuery.isFetchingNextPage,
      hasNextPage: restQuery.hasNextPage ?? false,
      loadMore,
      reset,
      error: restQuery.error,
    };
  }

  return {
    runs: allRuns,
    isLoading: queryRes.fetching,
    isLoadingInitial: queryRes.fetching && cursor === null,
    isLoadingMore: queryRes.fetching && cursor !== null,
    hasNextPage,
    loadMore,
    reset,
    error: queryRes.error,
  };
}

const REST_PAGE_SIZE = 40;

function fetchRestRuns(
  vars: UseRunsPaginationParams['commonQueryVars'],
  cursor: string | undefined,
  signal: AbortSignal,
): Promise<RestRunsPage> {
  const pathname =
    vars.functionSlug && vars.functionAppID
      ? `/v2/apps/${encodeURIComponent(vars.functionAppID)}/functions/${encodeURIComponent(vars.functionSlug)}/runs`
      : '/v2/runs';
  const params = new URLSearchParams({
    from: vars.startTime,
    timeField: vars.timeField,
    order: 'DESC',
    limit: String(REST_PAGE_SIZE),
  });
  if (vars.endTime) params.set('until', vars.endTime);
  if (cursor) params.set('cursor', cursor);
  if (vars.celQuery) params.set('query', vars.celQuery);
  if (vars.isDeferred !== null)
    params.set('isDeferred', String(vars.isDeferred));
  for (const status of vars.status ?? []) params.append('status', status);
  if (!vars.functionSlug) {
    for (const appID of vars.appIDs ?? []) params.append('appId', appID);
  }
  return fetchRunsPage(pathname, params, vars.environmentSlug, signal);
}
