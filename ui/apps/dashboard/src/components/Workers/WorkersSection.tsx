import { WorkersTable } from '@inngest/components/Workers/WorkersTable';

import { useWorkers, useWorkersCount } from './useWorker';

export default function WorkersSection({ appID }: { appID: string }) {
  const getWorkers = useWorkers();
  const getWorkerCount = useWorkersCount();

  return <WorkersTable appID={appID} getWorkers={getWorkers} getWorkerCount={getWorkerCount} />;
}
