import { ReplayStatus } from '@inngest/components/types/replay';
import { differenceInMilliseconds } from '@inngest/components/utils/date';
import { useQuery } from '@tanstack/react-query';
import { decodeTime } from 'ulid';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const getReplayQuery = graphql(`
  query GetReplay($envID: ID!, $replayID: ID!) {
    environment: workspace(id: $envID) {
      replay(id: $replayID) {
        id
        name
        createdAt
        endedAt
        functionRunsScheduledCount
        fromRange
        toRange
        functionRunsProcessedCount
        filtersV2 {
          statuses
        }
      }
    }
  }
`);

export function useGetReplay(replayID: string) {
  const envID = useEnvironment().id;
  const client = useClient();
  return useQuery({
    queryKey: ['replay', envID, replayID],
    queryFn: async () => {
      const result = await client
        .query(getReplayQuery, { envID, replayID })
        .toPromise();

      if (result.error) {
        throw result.error;
      }
      if (!result.data?.environment.replay) {
        throw new Error('Replay not found');
      }

      const replay = result.data.environment.replay;

      const baseReplay = {
        ...replay,
        createdAt: new Date(replay.createdAt),
        runsCount: replay.functionRunsScheduledCount,
        runsSkippedCount:
          replay.functionRunsScheduledCount - replay.functionRunsProcessedCount,
        fromRange: replay.fromRange
          ? new Date(decodeTime(replay.fromRange))
          : undefined,
        toRange: replay.toRange
          ? new Date(decodeTime(replay.toRange))
          : undefined,
        filters: replay.filtersV2,
      };

      if (replay.endedAt) {
        return {
          ...baseReplay,
          status: ReplayStatus.Ended,
          endedAt: new Date(replay.endedAt),
          duration: differenceInMilliseconds(
            new Date(replay.endedAt),
            new Date(replay.createdAt),
          ),
        };
      }

      return {
        ...baseReplay,
        status: ReplayStatus.Created,
        endedAt: undefined, // Convert from `null` to `undefined` to match the expected type
        duration: undefined,
      };
    },
    refetchInterval: 5000,
  });
}

const GetReplaysDocument = graphql(`
  query GetReplays($environmentID: ID!, $functionSlug: String!) {
    environment: workspace(id: $environmentID) {
      id
      function: workflowBySlug(slug: $functionSlug) {
        id
        replays {
          id
          name
          createdAt
          endedAt
          functionRunsScheduledCount
          functionRunsProcessedCount
        }
      }
    }
  }
`);

export function useGetReplays(functionSlug: string) {
  const envID = useEnvironment().id;
  const client = useClient();
  return useQuery({
    queryKey: ['replays', envID, functionSlug],
    queryFn: async () => {
      const result = await client
        .query(GetReplaysDocument, { environmentID: envID, functionSlug })
        .toPromise();

      if (result.error) {
        throw result.error;
      }
      // Map and transform into Replay[]
      const replays =
        result.data?.environment.function?.replays.map((replay) => {
          const baseReplay = {
            ...replay,
            createdAt: new Date(replay.createdAt),
            runsCount: replay.functionRunsScheduledCount,
            runsSkippedCount:
              replay.functionRunsScheduledCount -
              replay.functionRunsProcessedCount,
          };

          if (replay.endedAt) {
            return {
              ...baseReplay,
              status: ReplayStatus.Ended,
              endedAt: new Date(replay.endedAt),
              duration: differenceInMilliseconds(
                new Date(replay.endedAt),
                new Date(replay.createdAt),
              ),
            };
          }

          return {
            ...baseReplay,
            status: ReplayStatus.Created,
            endedAt: undefined, // Convert from `null` to `undefined` to match the expected type
          };
        }) ?? [];

      return replays;
    },
    refetchInterval: 5000,
  });
}
