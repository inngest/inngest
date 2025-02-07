import { useMemo } from 'react';
import { Description } from '@inngest/components/Apps/AppCard';
import { Skeleton } from '@inngest/components/Skeleton';
import WorkersCounter from '@inngest/components/Workers/WorkersCounter';

import { ConnectV1ConnectionStatus } from '@/gql/graphql';
import { useWorkerCount } from './useWorker';

type Props = {
  envID: string;
  appID: string;
};

export default function WorkerCounter({ envID, appID }: Props) {
  const {
    data: countReadyWorkersData,
    isLoading: loadingReadyWorkers,
    error: errorReadyWorkers,
  } = useWorkerCount({
    appID,
    envID,
    status: [ConnectV1ConnectionStatus.Ready],
  });

  const {
    data: countInactiveWorkersData,
    isLoading: loadingInactiveWorkers,
    error: errorInactiveWorkers,
  } = useWorkerCount({
    appID,
    envID,
    status: [
      ConnectV1ConnectionStatus.Connected,
      ConnectV1ConnectionStatus.Disconnecting,
      ConnectV1ConnectionStatus.Draining,
    ],
  });

  const isLoading = loadingReadyWorkers || loadingInactiveWorkers;

  // Absorve errors
  if (errorReadyWorkers || errorInactiveWorkers) {
    console.error(errorReadyWorkers || errorInactiveWorkers);
  }

  const workerCounts = useMemo(
    () => ({
      ACTIVE: countReadyWorkersData?.total || 0,
      INACTIVE: countInactiveWorkersData?.total || 0,
      DISCONNECTED: null,
    }),
    [countReadyWorkersData?.total, countInactiveWorkersData?.total]
  );

  return (
    <Description
      term="Connected workers"
      detail={
        isLoading && (!countReadyWorkersData || !countInactiveWorkersData) ? (
          <Skeleton className="block h-5 w-36" />
        ) : (
          <WorkersCounter counts={workerCounts} />
        )
      }
    />
  );
}
