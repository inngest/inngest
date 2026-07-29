import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

export type SessionRun = {
  id: string;
  functionSlug: string;
  eventName: string | null;
  status: string;
  queuedAt: string;
  startedAt: string | null;
  endedAt: string | null;
};

const sessionRunsQuery = graphql(`
  query SessionRuns(
    $workspaceID: ID!
    $sessionKey: String!
    $sessionId: String!
    $timeRange: TimeRangeInput
  ) {
    sessionRuns(
      workspaceID: $workspaceID
      sessionKey: $sessionKey
      sessionId: $sessionId
      timeRange: $timeRange
    ) {
      id
      functionSlug
      eventName
      status
      queuedAt
      startedAt
      endedAt
    }
  }
`);

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
  const client = useClient();
  const envID = useEnvironment().id;

  return useQuery({
    queryKey: ['session-runs', envID, sessionKey, sessionId, startTime, endTime],
    queryFn: async (): Promise<SessionRun[]> => {
      const result = await client
        .query(
          sessionRunsQuery,
          {
            workspaceID: envID,
            sessionKey,
            sessionId,
            timeRange: { from: startTime, to: endTime },
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) throw result.error;

      return (result.data?.sessionRuns ?? []).map((r) => ({
        id: r.id,
        functionSlug: r.functionSlug,
        eventName: r.eventName ?? null,
        // The schema types status as a plain string; uppercase it so it keys
        // into the shared FunctionRunStatus color classes and status filter.
        status: r.status.toUpperCase(),
        queuedAt: r.queuedAt,
        startedAt: r.startedAt ?? null,
        endedAt: r.endedAt ?? null,
      }));
    },
  });
}
