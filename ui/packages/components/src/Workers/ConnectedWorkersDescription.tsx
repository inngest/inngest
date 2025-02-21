import { useCallback, useMemo } from 'react';
import DescriptionListItem from '@inngest/components/Apps/DescriptionListItem';
import { Skeleton } from '@inngest/components/Skeleton';
import WorkersCounter from '@inngest/components/Workers/WorkersCounter';
import { workerStatuses } from '@inngest/components/types/workers';
import { keepPreviousData, useQuery } from '@tanstack/react-query';

type Props = {
  appID: string;
  getWorkerCount: ({
    appID,
    status,
  }: {
    appID: string;
    status: any[]; // TODO: Converge to workerStatuses
  }) => Promise<number>;
};

const refreshInterval = 5000;

export default function WorkerCounter({ appID, getWorkerCount }: Props) {
  const {
    isPending: pendingReadyWorkers,
    error: errorReadyWorkers,
    data: countReadyWorkersData,
  } = useQuery({
    queryKey: ['worker-counter-ready', { appID, status: [workerStatuses.Ready] }],
    queryFn: useCallback(() => {
      return getWorkerCount({ appID, status: [workerStatuses.Ready] });
    }, [getWorkerCount, appID]),
    placeholderData: keepPreviousData,
    refetchInterval: refreshInterval,
  });

  const {
    isPending: pendingInactiveWorkers,
    error: errorInactiveWorkers,
    data: countInactiveWorkersData,
  } = useQuery({
    queryKey: [
      'worker-counter-inactive',
      {
        appID,
        status: [workerStatuses.Connected, workerStatuses.Disconnecting, workerStatuses.Draining],
      },
    ],
    queryFn: useCallback(() => {
      return getWorkerCount({
        appID,
        status: [workerStatuses.Connected, workerStatuses.Disconnecting, workerStatuses.Draining],
      });
    }, [getWorkerCount, appID]),
    placeholderData: keepPreviousData,
    refetchInterval: refreshInterval,
  });

  const isLoading = pendingReadyWorkers || pendingInactiveWorkers;

  // Absorve errors
  if (errorReadyWorkers || errorInactiveWorkers) {
    console.error(errorReadyWorkers || errorInactiveWorkers);
  }

  const workerCounts = useMemo(
    () => ({
      ACTIVE: countReadyWorkersData || 0,
      INACTIVE: countInactiveWorkersData || 0,
      DISCONNECTED: null,
    }),
    [countReadyWorkersData, countInactiveWorkersData]
  );

  return (
    <DescriptionListItem
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
