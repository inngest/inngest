'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
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

import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useGetTrigger } from '@/hooks/useGetTrigger';
import { useRerun } from '@/hooks/useRerun';
import { client } from '@/store/baseApi';
import {
  CountRunsDocument,
  GetRunsDocument,
  type CountRunsQuery,
  type GetRunsQuery,
} from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

const pollInterval = 2500;

export default function Page() {
  const [totalCount, setTotalCount] = useState<number>();
  const [filteredStatus] = useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);
  const [timeField = FunctionRunTimeField.QueuedAt] = useValidatedSearchParam(
    'timeField',
    isFunctionTimeField
  );
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });
  console.log('calculatedStartTime', calculatedStartTime);
  console.log('lastDays', lastDays);
  console.log('startTime', startTime);

  const queryFn = useCallback(
    async ({ pageParam }: { pageParam: string | null }) => {
      const data: GetRunsQuery = await client.request(GetRunsDocument, {
        functionRunCursor: pageParam,
        startTime: calculatedStartTime,
        endTime: endTime,
        status: filteredStatus,
        timeField,
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
      console.log(edges.length);
      return {
        ...data.runs,
        edges,
      };
    },
    [filteredStatus, calculatedStartTime, timeField]
  );

  const { data, fetchNextPage, isFetching } = useInfiniteQuery({
    queryKey: ['runs'],
    queryFn,
    refetchInterval: pollInterval,
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
      });
      setTotalCount(data.runs.totalCount);
    })();
  }, [calculatedStartTime, endTime, filteredStatus, timeField]);

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
    <RunsPage
      cancelRun={cancelRun}
      data={runs ?? []}
      features={{
        history: Number.MAX_SAFE_INTEGER,
      }}
      hasMore={false}
      isLoadingInitial={isFetching && runs === undefined}
      isLoadingMore={isFetching && runs !== undefined}
      getRun={getRun}
      onScroll={onScroll}
      onScrollToTop={onScrollToTop}
      getTraceResult={getTraceResult}
      getTrigger={getTrigger}
      rerun={rerun}
      pathCreator={pathCreator}
      apps={[]}
      functions={[]}
      pollInterval={pollInterval}
      scope="env"
      totalCount={totalCount}
    />
  );
}
