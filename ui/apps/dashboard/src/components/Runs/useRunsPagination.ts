import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Run } from '@inngest/components/RunsPage/types';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useQuery } from 'urql';

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

export type ProgressiveSearchState = {
  phase: 'searching' | 'paused' | 'cancelled' | 'complete' | 'error';
  cursor?: string;
  error?: Error;
  cancel: () => void;
  resume: () => void;
};

export function useRunsPagination({
  commonQueryVars,
  tracePreviewEnabled,
  useREST,
}: UseRunsPaginationParams) {
  const progressive = useProgressiveRuns({
    enabled: useREST && Boolean(commonQueryVars.celQuery),
    vars: commonQueryVars,
  });
  const useProgressive = useREST && Boolean(commonQueryVars.celQuery);
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
    enabled: useREST && !useProgressive,
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
    if (useProgressive) {
      progressive.reset();
      return;
    }
    if (useREST) {
      void restQuery.refetch();
      return;
    }
    setCursor(null);
    setAllRuns([]);
    refetch();
  }, [refetch, progressive, restQuery.refetch, useProgressive, useREST]);

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
const PROGRESSIVE_MAX_PASSES = 10;
const PROGRESSIVE_MAX_MS = 10_000;
const PROGRESSIVE_MIN_PASS_INTERVAL_MS = 250;

function fetchRestRuns(
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
  if (vars.isDeferred !== null) {
    params.set('isDeferred', String(vars.isDeferred));
  }
  for (const status of vars.status ?? []) params.append('status', status);
  if (!vars.functionSlug) {
    for (const appID of vars.appIDs ?? []) params.append('appId', appID);
  }

  return fetchRunsPage(pathname, params, vars.environmentSlug, signal);
}

function useProgressiveRuns({
  enabled,
  vars,
}: {
  enabled: boolean;
  vars: UseRunsPaginationParams['commonQueryVars'];
}) {
  const inputKey = JSON.stringify(vars);
  const frozenVars = useMemo(() => vars, [inputKey]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [hasMore, setHasMore] = useState(true);
  const [phase, setPhase] =
    useState<ProgressiveSearchState['phase']>('searching');
  const [error, setError] = useState<Error>();
  const [attempt, setAttempt] = useState(0);
  const abortRef = useRef<AbortController>();

  useEffect(() => {
    setRuns([]);
    setCursor(undefined);
    setHasMore(true);
    setError(undefined);
    setPhase('searching');
    setAttempt((value) => value + 1);
  }, [enabled, inputKey]);

  useEffect(() => {
    if (!enabled || phase !== 'searching' || !hasMore) return;
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
              frozenVars,
              nextCursor,
              controller.signal,
            );
            const delay =
              PROGRESSIVE_MIN_PASS_INTERVAL_MS -
              (Date.now() - passStartedAt);
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
  }, [attempt, enabled, frozenVars, hasMore, phase]);

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
