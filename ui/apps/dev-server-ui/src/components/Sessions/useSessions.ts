import { useCallback } from 'react';
import type { SessionsPage } from '@inngest/components/types/session';

import { client } from '@/store/baseApi';

const sessionsQuery = `
  query Sessions($sessionKey: String!, $sessionIdSearch: String, $timeRange: TimeRangeInput, $after: String) {
    sessions(sessionKey: $sessionKey, sessionIdSearch: $sessionIdSearch, timeRange: $timeRange, after: $after) {
      nodes {
        sessionKey
        sessionId
        runCount
        failedRunCount
        failureRate
        lastActiveAt
        functions {
          slug
          name
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
`;

type SessionsQueryResult = {
  sessions: {
    nodes: Array<{
      sessionKey: string;
      sessionId: string;
      runCount: number;
      failedRunCount: number;
      failureRate: number;
      lastActiveAt: string;
      functions: Array<{ slug: string; name: string }>;
    }>;
    pageInfo: { hasNextPage: boolean; endCursor: string | null };
  };
};

type GetSessionsParams = {
  sessionKey: string;
  sessionIdSearch?: string;
  startTime: string;
  endTime: string;
  cursor?: string | null;
};

export function useSessions() {
  return useCallback(
    async (params: GetSessionsParams): Promise<SessionsPage> => {
      const { sessionKey, sessionIdSearch, startTime, endTime, cursor } =
        params;
      const result = await client.request<SessionsQueryResult>(sessionsQuery, {
        sessionKey,
        sessionIdSearch: sessionIdSearch || null,
        timeRange: { from: startTime, until: endTime },
        after: cursor ?? null,
      });

      return {
        sessions: result.sessions.nodes.map((session) => ({
          sessionKey: session.sessionKey,
          sessionId: session.sessionId,
          runCount: session.runCount,
          failedRunCount: session.failedRunCount,
          failureRate: session.failureRate,
          lastActiveAt: session.lastActiveAt,
          functions: session.functions,
        })),
        nextCursor: result.sessions.pageInfo.hasNextPage
          ? result.sessions.pageInfo.endCursor
          : null,
      };
    },
    [],
  );
}
