import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import {
  ConnectV1ConnectionStatus,
  CountWorkerConnectionsDocument,
  type CountWorkerConnectionsQuery,
} from '@/store/generated';

type QueryVariables = {
  appID: string;
  status?: ConnectV1ConnectionStatus[];
};

export function useGetWorkerCount() {
  return useCallback(async ({ appID, status }: QueryVariables) => {
    const data: CountWorkerConnectionsQuery = await client.request(CountWorkerConnectionsDocument, {
      appID: appID,
      status,
    });

    const workersData = data.workerConnections;

    return workersData.totalCount;
  }, []);
}
