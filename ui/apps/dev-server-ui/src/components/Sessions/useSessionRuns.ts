import { useMemo } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';

import { client } from '@/store/baseApi';

export type SessionRun = {
  id: string;
  functionSlug: string;
  eventName: string | null;
  status: string;
  queuedAt: string;
  startedAt: string | null;
  endedAt: string | null;
};

const sessionRunsQuery = `
  query SessionRuns($sessionKey: String!, $sessionId: String!, $timeRange: TimeRangeInput, $after: String, $withTotal: Boolean!) {
    sessionRuns(sessionKey: $sessionKey, sessionId: $sessionId, timeRange: $timeRange, after: $after) {
      nodes {
        id
        functionSlug
        eventName
        status
        queuedAt
        startedAt
        endedAt
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount @include(if: $withTotal)
    }
  }
`;

type SessionRunsQueryResult = {
  sessionRuns: {
    nodes: Array<{
      id: string;
      functionSlug: string;
      eventName: string | null;
      status: string;
      queuedAt: string;
      startedAt: string | null;
      endedAt: string | null;
    }>;
    pageInfo: { hasNextPage: boolean; endCursor: string | null };
    // Only requested on the first page, since it doesn't change per page.
    totalCount?: number;
  };
};

export function useSessionRuns({
  sessionKey,
  sessionId,
  startTime,
  endTime,
}: {
  sessionKey: string;
  sessionId: string;
  startTime: string;
  endTime: string;
}) {
  const query = useInfiniteQuery({
    queryKey: ['session-runs', sessionKey, sessionId, startTime, endTime],
    queryFn: async ({ pageParam }: { pageParam: string | null }) => {
      const result = await client.request<SessionRunsQueryResult>(
        sessionRunsQuery,
        {
          sessionKey,
          sessionId,
          timeRange: { from: startTime, until: endTime },
          after: pageParam,
          withTotal: pageParam === null,
        },
      );
      return result.sessionRuns;
    },
    initialPageParam: null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage
        ? lastPage.pageInfo.endCursor ?? undefined
        : undefined,
    refetchOnWindowFocus: false,
  });

  const data = useMemo(
    (): SessionRun[] | undefined =>
      query.data?.pages.flatMap((page) =>
        page.nodes.map((run) => ({
          id: run.id,
          functionSlug: run.functionSlug,
          eventName: run.eventName ?? null,
          status: run.status.toUpperCase(),
          queuedAt: run.queuedAt,
          startedAt: run.startedAt ?? null,
          endedAt: run.endedAt ?? null,
        })),
      ),
    [query.data],
  );

  // The whole session's run count (not just the loaded pages).
  const totalCount = query.data?.pages[0]?.totalCount;

  return { ...query, data, totalCount };
}
