import { useMemo } from 'react';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';

import { ConnectV1ConnectionStatus } from '@/gql/graphql';
import { useWorkers } from './useWorker';

type Props = {
  envID: string;
  appID: string;
};

export default function WorkersSection({ envID, appID }: Props) {
  // We return the last 7 days of active and inactive workers, but only the last day of disconnected workers
  const workerActiveAndInactiveRes = useWorkers({
    envID,
    appID,
    status: [
      ConnectV1ConnectionStatus.Ready,
      ConnectV1ConnectionStatus.Connected,
      ConnectV1ConnectionStatus.Disconnecting,
      ConnectV1ConnectionStatus.Draining,
    ],
  });

  if (workerActiveAndInactiveRes.error) {
    if (!workerActiveAndInactiveRes.data) {
      throw workerActiveAndInactiveRes.error;
    }
    console.error(workerActiveAndInactiveRes.error);
  }

  const workerDisconnectedRes = useWorkers({
    envID,
    appID,
    status: [ConnectV1ConnectionStatus.Disconnected],
    startTime: getTimestampDaysAgo({ currentDate: new Date(), days: 1 }).toISOString(),
  });

  if (workerActiveAndInactiveRes.error || workerDisconnectedRes.error) {
    console.error(workerActiveAndInactiveRes.error || workerDisconnectedRes.error);
  }

  const workerRes = useMemo(() => {
    const isLoading = workerActiveAndInactiveRes.isLoading || workerDisconnectedRes.isLoading;
    const workers = [
      ...(workerActiveAndInactiveRes.data?.workers || []),
      ...(workerDisconnectedRes.data?.workers || []),
    ];
    const total = workers.length;

    return {
      isLoading,
      data: { workers, total },
    };
  }, [workerActiveAndInactiveRes, workerDisconnectedRes]);

  return (
    <div>
      <h4 className="text-subtle mb-4 text-xl">Workers ({workerRes.data.total})</h4>
      <WorkersTable
        isLoading={workerRes.isLoading && !workerRes.data}
        workers={workerRes.data.workers}
      />
    </div>
  );
}
