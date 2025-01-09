'use client';

import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

import { useEnvironment } from '@/components/Environments/environment-context';
import { Sync } from './Sync';
import { SyncList } from './SyncList';
import { useSyncs } from './useSyncs';

type Props = {
  params: {
    externalID: string;
  };
};

export default function Page({ params }: Props) {
  const externalAppID = decodeURIComponent(params.externalID);
  const env = useEnvironment();
  const [selectedSyncID, setSelectedSyncID] = useSearchParam('sync-id');

  const syncsRes = useSyncs({ envID: env.id, externalAppID });
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
          <p className="bg-canvasMuted text-basis rounded-md p-4 text-center">No syncs found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0">
      <SyncList
        onClick={setSelectedSyncID}
        selectedSyncID={selectedSyncID ?? firstSync.id}
        syncs={syncsRes.data}
      />

      <Sync externalAppID={externalAppID} syncID={selectedSyncID ?? firstSync.id} />
    </div>
  );
}
