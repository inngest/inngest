import { useCallback } from 'react';
import type { Session } from '@inngest/components/types/session';

import { client } from '@/store/baseApi';

const sessionsQuery = `
  query Sessions($sessionKey: String!, $sessionIdSearch: String, $timeRange: TimeRangeInput) {
    sessions(sessionKey: $sessionKey, sessionIdSearch: $sessionIdSearch, timeRange: $timeRange) {
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
  }
`;

type SessionsQueryResult = {
  sessions: Array<{
    sessionKey: string;
    sessionId: string;
    runCount: number;
    failedRunCount: number;
    failureRate: number;
    lastActiveAt: string;
    functions: Array<{ slug: string; name: string }>;
  }>;
};

type GetSessionsParams = {
  sessionKey: string;
  sessionIdSearch?: string;
  startTime: string;
  endTime: string;
};

export function useSessions() {
  return useCallback(async (params: GetSessionsParams): Promise<Session[]> => {
    const { sessionKey, sessionIdSearch, startTime, endTime } = params;
    const result = await client.request<SessionsQueryResult>(sessionsQuery, {
      sessionKey,
      sessionIdSearch: sessionIdSearch || null,
      timeRange: { from: startTime, until: endTime },
    });

    return result.sessions.map((session) => ({
      sessionKey: session.sessionKey,
      sessionId: session.sessionId,
      runCount: session.runCount,
      failedRunCount: session.failedRunCount,
      failureRate: session.failureRate,
      lastActiveAt: session.lastActiveAt,
      functions: session.functions,
    }));
  }, []);
}
