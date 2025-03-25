import { useCallback } from 'react';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';

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
    const startTime = getTimestampDaysAgo({ currentDate: new Date(), days: 1 }).toISOString();
    const data: CountWorkerConnectionsQuery = await client.request(CountWorkerConnectionsDocument, {
      appID: appID,
      status,
      startTime,
    });

    const workersData = data.workerConnections;

    return workersData.totalCount;
  }, []);
}
