import { useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { createFileRoute } from '@tanstack/react-router';

import { SyncList } from '@/components/Apps/Syncs/SyncList';
import { useEnvironment } from '@/components/Environments/environment-context';
import { Sync } from '@/components/UnattachedSyncs/Sync';
import { useSyncs } from '@/components/UnattachedSyncs/useSyncs';

export const Route = createFileRoute('/_authed/env/$envSlug/unattached-syncs/')(
  {
    component: UnattachedSyncsPage,
  },
);

function UnattachedSyncsPage() {
  const env = useEnvironment();
  const [selectedSyncID, setSelectedSyncID] = useSearchParam('sync-id');

  const syncsRes = useSyncs({ envID: env.id });

  if (syncsRes.error) {
    throw syncsRes.error;
  }

  if (syncsRes.isLoading) {
    return (
      <div className="flex h-full min-h-0">
        <SyncList onClick={setSelectedSyncID} loading />
      </div>
    );
  }

  const firstSync = syncsRes.data[0];
  if (!firstSync) {
    return (
      <div className="h-full w-full overflow-y-auto">
        <div className="mx-auto mt-16 w-full max-w-[1200px] p-4">
          <p className="bg-canvasMuted text-basis rounded-md p-4 text-center">
            No syncs found
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full">
      <SyncList
        onClick={setSelectedSyncID}
        selectedSyncID={selectedSyncID ?? firstSync.id}
        syncs={syncsRes.data}
      />

      <Sync syncID={selectedSyncID ?? firstSync.id} />
    </div>
  );
}
