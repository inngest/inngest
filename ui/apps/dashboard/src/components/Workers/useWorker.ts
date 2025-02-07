import { useEffect, useState } from 'react';
import { convertWorkerStatus } from '@inngest/components/types/workers';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';

import { graphql } from '@/gql';
import { ConnectV1ConnectionStatus, ConnectV1WorkerConnectionsOrderByField } from '@/gql/graphql';
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

export function useWorkers({
  envID,
  appID,
  startTime,
  status = [],
}: {
  envID: string;
  appID: string;
  startTime?: string;
  status?: ConnectV1ConnectionStatus[];
}) {
  const [defaultStartTime] = useState(() =>
    getTimestampDaysAgo({ currentDate: new Date(), days: 7 }).toISOString()
  );
  const res = useGraphQLQuery({
    query,
    variables: {
      envID,
      appID,
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      status,
      startTime: startTime || defaultStartTime,
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

const countQuery = graphql(`
  query GetWorkerCountConnections(
    $envID: ID!
    $appID: UUID!
    $startTime: Time!
    $status: [ConnectV1ConnectionStatus!]
    $timeField: ConnectV1WorkerConnectionsOrderByField!
  ) {
    environment: workspace(id: $envID) {
      workerConnections(
        filter: { appIDs: [$appID], from: $startTime, status: $status }
        orderBy: [{ field: $timeField, direction: DESC }]
      ) {
        totalCount
      }
    }
  }
`);

export function useWorkerCount({
  envID,
  appID,
  status,
}: {
  envID: string;
  appID: string;
  status: ConnectV1ConnectionStatus[];
}) {
  const [startTime, setStartTime] = useState(() =>
    getTimestampDaysAgo({ currentDate: new Date(), days: 7 }).toISOString()
  );

  const pollIntervalInMilliseconds = 2_000;

  useEffect(() => {
    const updateStartTime = () => {
      setStartTime(getTimestampDaysAgo({ currentDate: new Date(), days: 7 }).toISOString());
    };

    const intervalId = setInterval(updateStartTime, pollIntervalInMilliseconds);

    return () => clearInterval(intervalId);
  }, []);

  const res = useGraphQLQuery({
    query: countQuery,
    pollIntervalInMilliseconds: pollIntervalInMilliseconds,
    variables: {
      envID,
      appID,
      status,
      startTime,
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
    },
  });

  if (res.data) {
    return {
      ...res,
      data: {
        total: res.data.environment.workerConnections.totalCount,
      },
    };
  }

  return { ...res, data: undefined };
}
