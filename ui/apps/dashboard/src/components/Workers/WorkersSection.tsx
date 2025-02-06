import { WorkersTable } from '@inngest/components/Workers/WorkersTable';

import { useWorkers } from './useWorker';

type Props = {
  envID: string;
  appID: string;
};

export default function WorkersSection({ envID, appID }: Props) {
  const workerRes = useWorkers({
    envID,
    appID,
  });
  if (workerRes.error) {
    if (!workerRes.data) {
      throw workerRes.error;
    }
    console.error(workerRes.error);
  }

  return (
    <div>
      <h4 className="text-subtle mb-4 text-xl">Workers ({workerRes.data?.total})</h4>
      <WorkersTable
        isLoading={workerRes.isLoading && !workerRes.data}
        workers={workerRes.data?.workers || []}
      />
    </div>
  );
}
