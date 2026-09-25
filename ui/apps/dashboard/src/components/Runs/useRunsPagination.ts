import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Run } from '@inngest/components/RunsPage/types';
import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import { useQuery } from 'urql';

import {
  useInngestAPIFetch,
  type InngestAPIFetch,
} from '@/queries/useInngestAPIFetch';

import { GetRunsDocument } from './queries';
import { scanProgressivePages } from './progressiveRuns';
import {
  fetchRunsPage,
  restFunctionRunToTableRun,
  RUNS_CEL_MAX_BYTES,
  RunsAPIError,
  type RestRunsPage,
} from './restRuns';
import { parseRunsData } from './utils';

type UseRunsPaginationParams = {
  commonQueryVars: {
    appIDs: string[] | null;
    restAppIDs: string[] | null | undefined;
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
  shouldUseREST: boolean;
  pause: boolean;
};

export type ProgressiveSearchState = {
  phase: 'searching' | 'paused' | 'cancelled' | 'complete' | 'error';
  cursor?: string;
  hasCompletedScanResponse: boolean;
  error?: Error;
  cancel: () => void;
  resume: () => void;
};

export function useRunsPagination({
  commonQueryVars,
  tracePreviewEnabled,
  shouldUseREST,
  pause,
}: UseRunsPaginationParams) {
  const apiFetch = useInngestAPIFetch(commonQueryVars.environmentSlug);
  const queryClient = useQueryClient();
  const progressive = useProgressiveRuns({
    enabled: !pause && shouldUseREST && Boolean(commonQueryVars.celQuery),
    apiFetch,
    vars: commonQueryVars,
  });
  const useProgressive = shouldUseREST && Boolean(commonQueryVars.celQuery);
  const [cursor, setCursor] = useState<string | null>(null);
  const [allRuns, setAllRuns] = useState<Run[]>([]);

  const [queryRes, refetch] = useQuery({
    pause: pause || shouldUseREST,
    query: GetRunsDocument,
    requestPolicy: 'network-only',
    variables: {
      ...commonQueryVars,
      functionRunCursor: cursor,
      preview: tracePreviewEnabled,
    },
  });

  const restQuery = useInfiniteQuery({
    enabled: !pause && shouldUseREST && !useProgressive,
    queryKey: ['runs-rest-v2', commonQueryVars],
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      fetchRestRuns(apiFetch, commonQueryVars, pageParam, signal),
    getNextPageParam: (lastPage) =>
      lastPage.page.hasMore ? lastPage.page.cursor : undefined,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    // fetchRunsPage already performs bounded 429 backoff. Do not let the query
    // client repeat that retry cycle after it is exhausted.
    retry: (failureCount, error) =>
      !(error instanceof RunsAPIError && error.status === 429) &&
      failureCount < 3,
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
    if (shouldUseREST) {
      if (!restQuery.error && !restQuery.isFetching && restQuery.hasNextPage) {
        void restQuery.fetchNextPage();
      }
      return;
    }
    if (!queryRes.fetching && hasNextPage && pageInfo?.endCursor) {
      setCursor(pageInfo.endCursor);
    }
  }, [
    shouldUseREST,
    restQuery.error,
    restQuery.isFetching,
    restQuery.hasNextPage,
    restQuery.fetchNextPage,
    queryRes.fetching,
    hasNextPage,
    pageInfo?.endCursor,
  ]);

  const reset = useCallback(() => {
    if (pause) return;
    if (useProgressive) {
      progressive.reset();
      return;
    }
    if (shouldUseREST) {
      void queryClient.resetQueries({
        queryKey: ['runs-rest-v2', commonQueryVars],
        exact: true,
      });
      return;
    }
    setCursor(null);
    setAllRuns([]);
    refetch();
  }, [
    commonQueryVars,
    pause,
    progressive,
    queryClient,
    refetch,
    shouldUseREST,
    useProgressive,
  ]);

  if (pause) {
    return {
      runs: [],
      isLoading: true,
      isLoadingInitial: true,
      isLoadingMore: false,
      hasNextPage: false,
      loadMore: () => {},
      reset,
      error: undefined,
      progressiveSearch: undefined,
    };
  }

  if (useProgressive) {
    return {
      runs: progressive.runs,
      isLoading: progressive.state.phase === 'searching',
      isLoadingInitial:
        progressive.state.phase === 'searching' &&
        progressive.runs.length === 0,
      isLoadingMore:
        progressive.state.phase === 'searching' && progressive.runs.length > 0,
      hasNextPage: false,
      loadMore: () => {},
      reset,
      error: progressive.state.error,
      progressiveSearch: progressive.state,
    };
  }

  if (shouldUseREST) {
    return {
      runs: restRuns,
      isLoading: restQuery.isFetching,
      isLoadingInitial: restQuery.isLoading,
      isLoadingMore: restQuery.isFetchingNextPage,
      hasNextPage: restQuery.hasNextPage ?? false,
      loadMore,
      reset,
      error: restQuery.error,
      progressiveSearch: undefined,
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
    progressiveSearch: undefined,
  };
}

const REST_PAGE_SIZE = 40;
const PROGRESSIVE_MAX_MS = 10_000;
const PROGRESSIVE_MIN_PASS_INTERVAL_MS = 250;

export function fetchRestRuns(
  apiFetch: InngestAPIFetch,
  vars: UseRunsPaginationParams['commonQueryVars'],
  cursor: string | undefined,
  signal: AbortSignal,
): Promise<RestRunsPage> {
  if (
    vars.celQuery &&
    new TextEncoder().encode(vars.celQuery).byteLength > RUNS_CEL_MAX_BYTES
  ) {
    return Promise.reject(
      new RunsAPIError('Query cannot exceed 2048 bytes', 'query_too_long', 422),
    );
  }
  // GraphQL identifies workflows by their app-qualified slug, while this REST
  // route expects the configured function ID.
  const functionID =
    vars.functionSlug &&
    vars.functionAppID &&
    vars.functionSlug.startsWith(`${vars.functionAppID}-`)
      ? vars.functionSlug.slice(vars.functionAppID.length + 1)
      : vars.functionSlug;
  const params = new URLSearchParams({
    from: vars.startTime,
    timeField: vars.timeField,
    order: 'DESC',
    limit: String(REST_PAGE_SIZE),
    include: 'deferred_from',
  });
  if (vars.endTime) params.set('until', vars.endTime);
  if (cursor) params.set('cursor', cursor);
  if (vars.celQuery) params.set('query', vars.celQuery);
  if (vars.isDeferred !== null) {
    params.set('isDeferred', String(vars.isDeferred));
  }
  for (const status of vars.status ?? []) params.append('status', status);
  if (functionID && vars.functionAppID) {
    params.set('appId', vars.functionAppID);
    params.set('functionId', functionID);
  } else if (!vars.functionSlug) {
    for (const appID of vars.restAppIDs ?? []) params.append('appId', appID);
  }
  return fetchRunsPage(apiFetch, '/v2/runs', params, signal);
}

export function useProgressiveRuns({
  enabled,
  apiFetch,
  vars,
}: {
  enabled: boolean;
  apiFetch: InngestAPIFetch;
  vars: UseRunsPaginationParams['commonQueryVars'];
}) {
  const inputKey = JSON.stringify(vars);
  const lifecycleKey = `${enabled}:${inputKey}`;
  const frozenVars = useMemo(() => vars, [inputKey]);
  const [activeLifecycleKey, setActiveLifecycleKey] = useState(lifecycleKey);
  const [runs, setRuns] = useState<Run[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [hasMore, setHasMore] = useState(true);
  const [hasCompletedScanResponse, setHasCompletedScanResponse] =
    useState(false);
  const [phase, setPhase] =
    useState<ProgressiveSearchState['phase']>('searching');
  const [error, setError] = useState<Error>();
  const [attempt, setAttempt] = useState(0);
  const abortRef = useRef<AbortController>();

  useEffect(() => {
    if (activeLifecycleKey === lifecycleKey) return;
    setRuns([]);
    setCursor(undefined);
    setHasMore(true);
    setHasCompletedScanResponse(false);
    setError(undefined);
    setPhase('searching');
    setAttempt((value) => value + 1);
    setActiveLifecycleKey(lifecycleKey);
  }, [activeLifecycleKey, lifecycleKey]);

  useEffect(() => {
    if (
      !enabled ||
      activeLifecycleKey !== lifecycleKey ||
      phase !== 'searching' ||
      !hasMore
    ) {
      return;
    }
    const controller = new AbortController();
    abortRef.current = controller;
    let disposed = false;

    void (async () => {
      try {
        const reason = await scanProgressivePages({
          initialCursor: cursor,
          initialItems: runs,
          fetchPage: async (nextCursor) => {
            const passStartedAt = Date.now();
            const page = await fetchRestRuns(
              apiFetch,
              frozenVars,
              nextCursor,
              controller.signal,
            );
            const delay =
              PROGRESSIVE_MIN_PASS_INTERVAL_MS - (Date.now() - passStartedAt);
            if (page.page.hasMore && delay > 0) {
              await new Promise((resolve) => setTimeout(resolve, delay));
              controller.signal.throwIfAborted();
            }
            return {
              items: page.data.map(restFunctionRunToTableRun),
              cursor: page.page.cursor,
              hasMore: page.page.hasMore,
            };
          },
          onCommit: (items, nextCursor, more) => {
            if (disposed || controller.signal.aborted) return;
            setRuns(items);
            setCursor(nextCursor);
            setHasMore(more);
            setHasCompletedScanResponse(true);
          },
          signal: controller.signal,
          displayTarget: REST_PAGE_SIZE,
          maxMilliseconds: PROGRESSIVE_MAX_MS,
        });
        if (!disposed) setPhase(reason === 'complete' ? 'complete' : 'paused');
      } catch (err) {
        if (controller.signal.aborted || disposed) return;
        setError(
          err instanceof Error ? err : new Error('Unable to search runs'),
        );
        setPhase('error');
      }
    })();

    return () => {
      disposed = true;
      controller.abort();
    };
  }, [
    activeLifecycleKey,
    apiFetch,
    attempt,
    enabled,
    frozenVars,
    hasMore,
    lifecycleKey,
    phase,
  ]);

  const resume = useCallback(() => {
    if (!hasMore) return;
    setError(undefined);
    setPhase('searching');
    setAttempt((value) => value + 1);
  }, [hasMore]);
  const cancel = useCallback(() => {
    abortRef.current?.abort();
    setPhase('cancelled');
  }, []);
  const reset = useCallback(() => {
    abortRef.current?.abort();
    setRuns([]);
    setCursor(undefined);
    setHasMore(true);
    setHasCompletedScanResponse(false);
    setError(undefined);
    setPhase('searching');
    setAttempt((value) => value + 1);
  }, []);
  const isCurrentLifecycle = activeLifecycleKey === lifecycleKey;

  return {
    runs: isCurrentLifecycle ? runs : [],
    state: {
      phase: isCurrentLifecycle ? phase : 'searching',
      cursor: isCurrentLifecycle ? cursor : undefined,
      hasCompletedScanResponse: isCurrentLifecycle
        ? hasCompletedScanResponse
        : false,
      error: isCurrentLifecycle ? error : undefined,
      cancel,
      resume,
    } satisfies ProgressiveSearchState,
    reset,
  };
}
