import { convertWorkerStatus } from '@inngest/components/types/workers';

import { graphql } from '@/gql';
import { ConnectV1WorkerConnectionsOrderByField } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetWorkerConnections(
    $envID: ID!
    $externalAppID: UUID!
    $startTime: Time!
    $status: [ConnectV1ConnectionStatus!]
    $timeField: ConnectV1WorkerConnectionsOrderByField!
    $connectionCursor: String = null
  ) {
    environment: workspace(id: $envID) {
      workerConnections(
        filter: { appIDs: [$externalAppID], from: $startTime, status: $status }
        orderBy: [{ field: $timeField, direction: DESC }]
        after: $connectionCursor
      ) {
        edges {
          node {
            id
            gatewayId
            instanceID: instanceId
            workerIp
            app {
              id
            }
            connectedAt
            lastHeartbeatAt
            disconnectedAt
            disconnectReason
            status
            sdkLang
            sdkVersion
            sdkPlatform
            appVersion: buildId
            functionCount
            cpuCores
            memBytes
            os
          }
        }
        pageInfo {
          hasNextPage
          hasPreviousPage
          startCursor
          endCursor
        }
      }
    }
  }
`);

export function useWorkers({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: {
      envID,
      externalAppID,
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      status: [],
      startTime: new Date().toISOString(),
    },
  });

  if (res.data) {
    const workers = res.data.environment.workerConnections.edges.map((e) => ({
      ...e.node,
      status: convertWorkerStatus(e.node.status),
      appVersion: e.node.appVersion || 'unknown',
    }));

    return {
      ...res,
      data: workers,
    };
  }

  return { ...res, data: undefined };
}
