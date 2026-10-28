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
  query SessionRuns($sessionKey: String!, $sessionId: String!, $timeRange: TimeRangeInput, $after: String) {
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

  return { ...query, data };
}
