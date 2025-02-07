'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { RunsActionMenu } from '@inngest/components/RunsPage/ActionMenu';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
  useStringArraySearchParam,
  useValidatedArraySearchParam,
  useValidatedSearchParam,
} from '@inngest/components/hooks/useSearchParam';
import {
  FunctionRunTimeField,
  isFunctionRunStatus,
  isFunctionTimeField,
} from '@inngest/components/types/functionRun';
import { toMaybeDate } from '@inngest/components/utils/date';
import { useInfiniteQuery } from '@tanstack/react-query';

import SendEventButton from '@/components/Event/SendEventButton';
import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useGetTrigger } from '@/hooks/useGetTrigger';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { client } from '@/store/baseApi';
import {
  CountRunsDocument,
  GetRunsDocument,
  useGetAppsQuery,
  type CountRunsQuery,
  type GetRunsQuery,
} from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

const pollInterval = 400;

export default function Page() {
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [filterApp] = useStringArraySearchParam('filterApp');
  const [totalCount, setTotalCount] = useState<number>();
  const [filteredStatus] = useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);
  const [timeField = FunctionRunTimeField.QueuedAt] = useValidatedSearchParam(
    'timeField',
    isFunctionTimeField
  );
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const [search] = useSearchParam('search');
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });
  const appsRes = useGetAppsQuery();

  const queryFn = useCallback(
    async ({ pageParam }: { pageParam: string | null }) => {
      const data: GetRunsQuery = await client.request(GetRunsDocument, {
        appIDs: filterApp,
        functionRunCursor: pageParam,
        startTime: calculatedStartTime,
        endTime: endTime,
        status: filteredStatus,
        timeField,
        celQuery: search,
      });

      const edges = data.runs.edges.map((edge) => {
        let durationMS: number | null = null;
        const startedAt = toMaybeDate(edge.node.startedAt);
        if (startedAt) {
          const endedAt = toMaybeDate(edge.node.endedAt);
          durationMS = (endedAt ?? new Date()).getTime() - startedAt.getTime();
        }

        return {
          ...edge.node,
          durationMS,
        };
      });
      return {
        ...data.runs,
        edges,
      };
    },
    [filterApp, filteredStatus, calculatedStartTime, timeField, search]
  );

  const { data, fetchNextPage, isFetching, hasNextPage } = useInfiniteQuery({
    queryKey: ['runs'],
    queryFn,
    refetchInterval: autoRefresh ? pollInterval : false,
    initialPageParam: null,
    getNextPageParam: (lastPage) => {
      if (!lastPage) {
        return undefined;
      }

      return lastPage.pageInfo.endCursor;
    },
  });

  useEffect(() => {
    setTotalCount(undefined);

    (async () => {
      const data: CountRunsQuery = await client.request(CountRunsDocument, {
        startTime: calculatedStartTime,
        endTime,
        status: filteredStatus,
        timeField,
        celQuery: search,
      });
      setTotalCount(data.runs.totalCount);
    })();
  }, [calculatedStartTime, endTime, filteredStatus, timeField, search]);

  const runs = useMemo(() => {
    if (!data?.pages) {
      return undefined;
    }
    if (data.pages.length === 0) {
      return [];
    }

    const out = [];
    for (const page of data.pages) {
      out.push(...page.edges);
    }
    return out;
  }, [data?.pages]);

  const cancelRun = useCancelRun();
  const rerun = useRerun();
  const rerunFromStep = useRerunFromStep();
  const getTraceResult = useGetTraceResult();
  const getTrigger = useGetTrigger();
  const getRun = useGetRun();

  const onScroll: React.ComponentProps<typeof RunsPage>['onScroll'] = useCallback(
    (event) => {
      if (runs && runs.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isFetching) {
          fetchNextPage();
        }
      }
    },
    [isFetching, runs]
  );

  const onScrollToTop = useCallback(() => {
    // TODO: What should this do?
  }, []);

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }]}
        action={
          <div className="flex flex-row items-center gap-x-1">
            <SendEventButton
              label="Send test event"
              data={JSON.stringify({
                name: '',
                data: {},
                user: {},
              })}
            />
            <RunsActionMenu
              setAutoRefresh={() => setAutoRefresh(!autoRefresh)}
              autoRefresh={autoRefresh}
              intervalSeconds={pollInterval / 1000}
            />
          </div>
        }
      />
      <RunsPage
        apps={appsRes.data?.apps || []}
        cancelRun={cancelRun}
        data={runs ?? []}
        defaultVisibleColumns={['status', 'id', 'trigger', 'function', 'queuedAt', 'endedAt']}
        features={{
          history: Number.MAX_SAFE_INTEGER,
        }}
        hasMore={hasNextPage ?? false}
        isLoadingInitial={isFetching && runs === undefined}
        isLoadingMore={isFetching && runs !== undefined}
        getRun={getRun}
        onScroll={onScroll}
        onScrollToTop={onScrollToTop}
        onRefresh={fetchNextPage}
        getTraceResult={getTraceResult}
        getTrigger={getTrigger}
        rerun={rerun}
        pathCreator={pathCreator}
        pollInterval={pollInterval}
        scope="env"
        totalCount={totalCount}
        traceAIEnabled={false}
      />
    </>
  );
}
