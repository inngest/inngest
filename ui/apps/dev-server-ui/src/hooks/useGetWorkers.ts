import { useCallback } from 'react';
import { convertWorkerStatus } from '@inngest/components/types/workers';

import { client } from '@/store/baseApi';
import {
  ConnectV1WorkerConnectionsOrderByField,
  GetWorkerConnectionsDocument,
  type ConnectV1WorkerConnectionsOrderBy,
  type GetWorkerConnectionsQuery,
} from '@/store/generated';

type QueryVariables = {
  appID: string;
  orderBy: ConnectV1WorkerConnectionsOrderBy[];
  cursor: string | null;
  pageSize: number;
};

export function useGetWorkers() {
  return useCallback(async ({ appID, orderBy, cursor, pageSize }: QueryVariables) => {
    const data: GetWorkerConnectionsQuery = await client.request(GetWorkerConnectionsDocument, {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      orderBy,
      startTime: null,
      appID: appID,
      status: [],
      cursor,
      first: pageSize,
    });

    const workersData = data.workerConnections;

    const workers = workersData.edges
      .filter((e) => e.node !== null)
      .map((e) => {
        return {
          ...e.node,
          status: convertWorkerStatus(e.node.status),
          instanceID: e.node.instanceId,
          appVersion: e.node.buildId || 'unknown',
        };
      });

    return {
      workers,
      pageInfo: workersData.pageInfo,
    };
  }, []);
}
