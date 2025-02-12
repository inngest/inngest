import { useState } from 'react';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import {
  ConnectV1WorkerConnectionsOrderByDirection,
  ConnectV1WorkerConnectionsOrderByField,
  type ConnectV1WorkerConnectionsOrderBy,
} from '@inngest/components/types/workers';

import { useWorkers } from './useWorker';

type Props = {
  envID: string;
  appID: string;
};

export default function WorkersSection({ envID, appID }: Props) {
  const [orderBy, setOrderBy] = useState<ConnectV1WorkerConnectionsOrderBy[]>([
    {
      field: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      direction: ConnectV1WorkerConnectionsOrderByDirection.Asc,
    },
  ]);

  const workerRes = useWorkers({
    envID,
    appID,
    status: [],
    orderBy,
  });

  if (workerRes.error) {
    if (!workerRes.data) {
      throw workerRes.error;
    }
    console.error(workerRes.error);
  }

  if (workerRes.error) {
    console.error(workerRes.error);
  }

  return (
    <div>
      <h4 className="text-subtle mb-4 text-xl">Workers ({workerRes.data?.total})</h4>
      <WorkersTable
        isLoading={workerRes.isLoading && !workerRes.data}
        workers={workerRes.data?.workers || []}
        onSortingChange={setOrderBy}
      />
    </div>
  );
}
