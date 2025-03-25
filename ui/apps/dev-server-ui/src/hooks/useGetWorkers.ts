import { useCallback } from 'react';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { convertWorkerStatus } from '@inngest/components/utils/workerParser';

import { client } from '@/store/baseApi';
import {
  ConnectV1ConnectionStatus,
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
  status: ConnectV1ConnectionStatus[];
};

export function useGetWorkers() {
  return useCallback(async ({ appID, orderBy, cursor, pageSize, status }: QueryVariables) => {
    const startTime = getTimestampDaysAgo({ currentDate: new Date(), days: 1 }).toISOString();
    const data: GetWorkerConnectionsQuery = await client.request(GetWorkerConnectionsDocument, {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      orderBy,
      startTime,
      appID: appID,
      status,
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
        };
      });

    return {
      workers,
      pageInfo: workersData.pageInfo,
      totalCount: workersData.totalCount,
    };
  }, []);
}
