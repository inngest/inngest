import { useState } from 'react';
import { convertWorkerStatus } from '@inngest/components/types/workers';

import { graphql } from '@/gql';
import { ConnectV1WorkerConnectionsOrderByField } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetWorkerConnections(
    $envID: ID!
    $appID: UUID!
    $startTime: Time!
    $status: [ConnectV1ConnectionStatus!]
    $timeField: ConnectV1WorkerConnectionsOrderByField!
    $connectionCursor: String = null
  ) {
    environment: workspace(id: $envID) {
      workerConnections(
        filter: { appIDs: [$appID], from: $startTime, status: $status }
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
        totalCount
      }
    }
  }
`);

export function useWorkers({ envID, appID }: { envID: string; appID: string }) {
  const [startTime] = useState(() => new Date().toISOString());
  const res = useGraphQLQuery({
    query,
    variables: {
      envID,
      appID,
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      status: [],
      startTime: startTime,
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
      data: {
        workers: workers,
        total: res.data.environment.workerConnections.totalCount,
        pageInfo: res.data.environment.workerConnections.pageInfo,
      },
    };
  }

  return { ...res, data: undefined };
}
