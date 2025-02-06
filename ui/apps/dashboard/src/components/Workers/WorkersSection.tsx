import { WorkersTable } from '@inngest/components/Workers/WorkersTable';

import { useWorkers } from './useWorker';

type Props = {
  envID: string;
  externalAppID: string;
};

export default function WorkersSection({ envID, externalAppID }: Props) {
  const workerRes = useWorkers({
    envID,
    externalAppID,
  });
  if (workerRes.error) {
    if (!workerRes.data) {
      throw workerRes.error;
    }
    console.error(workerRes.error);
  }

  return (
    <div>
      {/* TODO: Add total count in title */}
      <h4 className="text-subtle mb-4 text-xl">Workers</h4>
      <WorkersTable
        isLoading={workerRes.isLoading && !workerRes.data}
        workers={workerRes.data || []}
      />
    </div>
  );
}
