import { useCallback } from 'react';
import { type Session } from '@inngest/components/types/session';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const sessionsQuery = graphql(`
  query Sessions(
    $workspaceID: ID!
    $sessionKey: String!
    $timeRange: TimeRangeInput
  ) {
    sessions(
      workspaceID: $workspaceID
      sessionKey: $sessionKey
      timeRange: $timeRange
    ) {
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
`);

type GetSessionsParams = {
  sessionKey: string;
  startTime: string;
  endTime: string;
};

export function useSessions() {
  const client = useClient();
  const envID = useEnvironment().id;

  return useCallback(
    async (params: GetSessionsParams): Promise<Session[]> => {
      const { sessionKey, startTime, endTime } = params;

      const result = await client
        .query(
          sessionsQuery,
          {
            workspaceID: envID,
            sessionKey,
            timeRange: { from: startTime, to: endTime },
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) throw result.error;

      return (result.data?.sessions ?? []).map((s) => ({
        sessionKey: s.sessionKey,
        sessionId: s.sessionId,
        runCount: s.runCount,
        failedRunCount: s.failedRunCount,
        failureRate: s.failureRate,
        lastActiveAt: s.lastActiveAt,
        functions: s.functions,
      }));
    },
    [client, envID],
  );
}
