import { useMemo } from 'react';
import { Description } from '@inngest/components/Apps/AppCard';
import { Skeleton } from '@inngest/components/Skeleton';
import WorkersCounter from '@inngest/components/Workers/WorkersCounter';

import { ConnectV1ConnectionStatus, useCountWorkerConnectionsQuery } from '@/store/generated';

type Props = {
  appID: string;
};

const refreshInterval = 5000;

export default function WorkerCounter({ appID }: Props) {
  const {
    data: countReadyWorkersData,
    isLoading: loadingReadyWorkers,
    error: errorReadyWorkers,
  } = useCountWorkerConnectionsQuery(
    {
      appID,
      status: [ConnectV1ConnectionStatus.Ready],
    },
    { pollingInterval: refreshInterval }
  );
  const {
    data: countInactiveWorkersData,
    isLoading: loadingInactiveWorkers,
    error: errorInactiveWorkers,
  } = useCountWorkerConnectionsQuery(
    {
      appID,
      status: [
        ConnectV1ConnectionStatus.Connected,
        ConnectV1ConnectionStatus.Disconnecting,
        ConnectV1ConnectionStatus.Draining,
      ],
    },
    { pollingInterval: refreshInterval }
  );
  const {
    data: countDisconnectedWorkersData,
    isLoading: loadingDisconnectedWorkers,
    error: errorDisconnectedWorkers,
  } = useCountWorkerConnectionsQuery(
    {
      appID,
      status: [ConnectV1ConnectionStatus.Disconnected],
    },
    { pollingInterval: refreshInterval }
  );

  const isLoading = loadingReadyWorkers || loadingInactiveWorkers || loadingDisconnectedWorkers;

  // Absorve errors
  if (errorReadyWorkers || errorInactiveWorkers || errorDisconnectedWorkers) {
    console.error(errorReadyWorkers || errorInactiveWorkers || errorDisconnectedWorkers);
  }

  const workerCounts = useMemo(
    () => ({
      ACTIVE: countReadyWorkersData?.workerConnections.totalCount || 0,
      INACTIVE: countInactiveWorkersData?.workerConnections.totalCount || 0,
      DISCONNECTED: countDisconnectedWorkersData?.workerConnections.totalCount || 0,
    }),
    [
      countReadyWorkersData?.workerConnections.totalCount,
      countInactiveWorkersData?.workerConnections.totalCount,
      countDisconnectedWorkersData?.workerConnections.totalCount,
    ]
  );

  return (
    <Description
      term="Connected workers"
      detail={
        isLoading &&
        (!countReadyWorkersData || !countInactiveWorkersData || !countDisconnectedWorkersData) ? (
          <Skeleton className="block h-5 w-36" />
        ) : (
          <WorkersCounter counts={workerCounts} />
        )
      }
    />
  );
}
