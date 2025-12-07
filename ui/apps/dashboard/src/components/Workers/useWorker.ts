import { useCallback } from "react";
import { getTimestampDaysAgo } from "@inngest/components/utils/date";
import { convertWorkerStatus } from "@inngest/components/utils/workerParser";
import { useClient } from "urql";

import { useEnvironment } from "@/components/Environments/environment-context";
import { graphql } from "@/gql";
import {
  ConnectV1ConnectionStatus,
  ConnectV1WorkerConnectionsOrderByField,
  type ConnectV1WorkerConnectionsOrderBy,
} from "@/gql/graphql";

const query = graphql(`
  query GetWorkerConnections(
    $envID: ID!
    $appID: UUID!
    $startTime: Time
    $status: [ConnectV1ConnectionStatus!]
    $timeField: ConnectV1WorkerConnectionsOrderByField!
    $cursor: String = null
    $orderBy: [ConnectV1WorkerConnectionsOrderBy!] = []
    $first: Int!
  ) {
    environment: workspace(id: $envID) {
      workerConnections(
        first: $first
        filter: { appIDs: [$appID], from: $startTime, status: $status, timeField: $timeField }
        orderBy: $orderBy
        after: $cursor
      ) {
        edges {
          node {
            id
            gatewayId
            instanceID: instanceId
            workerIp
            maxWorkerConcurrency
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
            appVersion
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

type QueryVariables = {
  appID: string;
  orderBy: ConnectV1WorkerConnectionsOrderBy[];
  cursor: string | null;
  pageSize: number;
  status: ConnectV1ConnectionStatus[];
};

export function useWorkers() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ appID, orderBy, cursor, pageSize, status }: QueryVariables) => {
      const result = await client
        .query(
          query,
          {
            timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
            orderBy,
            startTime: getTimestampDaysAgo({
              currentDate: new Date(),
              days: 30,
            }).toISOString(),
            appID: appID,
            status,
            cursor,
            first: pageSize,
            envID,
          },
          { requestPolicy: "network-only" },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error("no data returned");
      }

      const workersData = result.data.environment.workerConnections;

      const workers = workersData.edges.map((e) => ({
        ...e.node,
        status: convertWorkerStatus(e.node.status),
      }));

      return {
        workers,
        pageInfo: workersData.pageInfo,
        totalCount: workersData.totalCount,
      };
    },
    [client, envID],
  );
}

type CountQueryVariables = {
  appID: string;
  status?: ConnectV1ConnectionStatus[];
};

const countQuery = graphql(`
  query GetWorkerCountConnections(
    $envID: ID!
    $appID: UUID!
    $startTime: Time
    $status: [ConnectV1ConnectionStatus!] = []
    $timeField: ConnectV1WorkerConnectionsOrderByField!
  ) {
    environment: workspace(id: $envID) {
      workerConnections(
        filter: { appIDs: [$appID], from: $startTime, status: $status, timeField: $timeField }
        orderBy: [{ field: $timeField, direction: DESC }]
      ) {
        totalCount
      }
    }
  }
`);

export function useWorkersCount() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ appID, status }: CountQueryVariables) => {
      const result = await client
        .query(
          countQuery,
          {
            timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
            startTime: getTimestampDaysAgo({
              currentDate: new Date(),
              days: 30,
            }).toISOString(),
            appID: appID,
            envID,
            status,
          },
          { requestPolicy: "network-only" },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error("no data returned");
      }

      const workersData = result.data.environment.workerConnections;

      return workersData.totalCount;
    },
    [client, envID],
  );
}
