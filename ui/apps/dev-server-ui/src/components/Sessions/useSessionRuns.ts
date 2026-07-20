import { useQuery } from '@tanstack/react-query';

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
  query SessionRuns($sessionKey: String!, $sessionId: String!, $timeRange: TimeRangeInput) {
    sessionRuns(sessionKey: $sessionKey, sessionId: $sessionId, timeRange: $timeRange) {
      id
      functionSlug
      eventName
      status
      queuedAt
      startedAt
      endedAt
    }
  }
`;

type SessionRunsQueryResult = {
  sessionRuns: Array<{
    id: string;
    functionSlug: string;
    eventName: string | null;
    status: string;
    queuedAt: string;
    startedAt: string | null;
    endedAt: string | null;
  }>;
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
  return useQuery({
    queryKey: ['session-runs', sessionKey, sessionId, startTime, endTime],
    queryFn: async (): Promise<SessionRun[]> => {
      const result = await client.request<SessionRunsQueryResult>(
        sessionRunsQuery,
        {
          sessionKey,
          sessionId,
          timeRange: { from: startTime, until: endTime },
        },
      );

      return result.sessionRuns.map((run) => ({
        id: run.id,
        functionSlug: run.functionSlug,
        eventName: run.eventName ?? null,
        status: run.status.toUpperCase(),
        queuedAt: run.queuedAt,
        startedAt: run.startedAt ?? null,
        endedAt: run.endedAt ?? null,
      }));
    },
    refetchOnWindowFocus: false,
  });
}
