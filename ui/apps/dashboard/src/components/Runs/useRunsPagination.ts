import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Run } from '@inngest/components/RunsPage/types';
import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query';

import {
  useInngestAPIFetch,
  type InngestAPIFetch,
} from '@/queries/useInngestAPIFetch';

import { scanProgressivePages } from './progressiveRuns';
import {
  fetchRunsPage,
  restFunctionRunToTableRun,
  RUNS_CEL_MAX_BYTES,
  RunsAPIError,
  type RestRunsPage,
} from './restRuns';

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
  enabled: boolean;
};

export type ProgressiveSearchState = {
  phase: 'searching' | 'paused' | 'cancelled' | 'complete' | 'error';
  cursor?: string;
  error?: Error;
  cancel: () => void;
  resume: () => void;
};

export function useRunsPagination({
  commonQueryVars,
  enabled,
}: UseRunsPaginationParams) {
  const apiFetch = useInngestAPIFetch(commonQueryVars.environmentSlug);
  const queryClient = useQueryClient();
  const useProgressive = Boolean(commonQueryVars.celQuery);
  const progressive = useProgressiveRuns({
    enabled: enabled && useProgressive,
    apiFetch,
    vars: commonQueryVars,
  });

  const restQuery = useInfiniteQuery({
    enabled: enabled && !useProgressive,
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

  const loadMore = useCallback(() => {
    if (!restQuery.error && !restQuery.isFetching && restQuery.hasNextPage) {
      void restQuery.fetchNextPage();
    }
  }, [
    restQuery.error,
    restQuery.isFetching,
    restQuery.hasNextPage,
    restQuery.fetchNextPage,
  ]);

  const reset = useCallback(() => {
    if (!enabled) return;
    if (useProgressive) {
      progressive.reset();
      return;
    }
    void queryClient.resetQueries({
      queryKey: ['runs-rest-v2', commonQueryVars],
      exact: true,
    });
  }, [commonQueryVars, enabled, progressive, queryClient, useProgressive]);

  if (!enabled) {
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

const REST_PAGE_SIZE = 40;
const PROGRESSIVE_MAX_PASSES = 10;
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
  const pathname =
    functionID && vars.functionAppID
      ? `/v2/apps/${encodeURIComponent(
          vars.functionAppID,
        )}/functions/${encodeURIComponent(functionID)}/runs`
      : '/v2/runs';
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
  if (!vars.functionSlug) {
    for (const appID of vars.restAppIDs ?? []) params.append('appId', appID);
  }
  return fetchRunsPage(apiFetch, pathname, params, signal);
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
          },
          signal: controller.signal,
          displayTarget: REST_PAGE_SIZE,
          maxPasses: PROGRESSIVE_MAX_PASSES,
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
    setError(undefined);
    setPhase('searching');
    setAttempt((value) => value + 1);
  }, []);

  return {
    runs,
    state: {
      phase,
      cursor,
      error,
      cancel,
      resume,
    } satisfies ProgressiveSearchState,
    reset,
  };
}
