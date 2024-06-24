'use client';

import { useCallback, useMemo } from 'react';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
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
import { getTimestampDaysAgo, toMaybeDate } from '@inngest/components/utils/date';
import { useInfiniteQuery } from '@tanstack/react-query';

import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useGetTrigger } from '@/hooks/useGetTrigger';
import { useRerun } from '@/hooks/useRerun';
import { client } from '@/store/baseApi';
import { GetRunsDocument, type GetRunsQuery } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export default function Page({
  params,
}: {
  params: {
    slug: string;
  };
}) {
  const functionSlug = decodeURIComponent(params.slug);

  const [filteredStatus] = useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);
  const [timeField = FunctionRunTimeField.QueuedAt] = useValidatedSearchParam(
    'timeField',
    isFunctionTimeField
  );
  const [lastDays = '3'] = useSearchParam('last');
  const startTime = useMemo(() => {
    return getTimestampDaysAgo({
      currentDate: new Date(),
      days: parseInt(lastDays),
    });
  }, []);

  const queryFn = useCallback(
    async ({ pageParam }: { pageParam: string | null }) => {
      const data: GetRunsQuery = await client.request(GetRunsDocument, {
        functionRunCursor: pageParam,
        startTime,
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

      return {
        ...data.runs,
        edges,
      };
    },
    [filteredStatus, startTime, timeField]
  );

  const { data, fetchNextPage, isFetching } = useInfiniteQuery({
    queryKey: ['runs'],
    queryFn,
    initialPageParam: null,
    getNextPageParam: (lastPage) => {
      if (!lastPage) {
        return undefined;
      }

      return lastPage.pageInfo.endCursor;
    },
  });

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
      functionSlug={functionSlug}
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
    />
  );
}
