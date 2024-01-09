'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { useSearchParam } from '@/utils/useSearchParam';
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
    return null;
  }
  const firstSync = syncsRes.data[0];
  if (!firstSync) {
    // TODO: Make pretty
    return 'No syncs found';
  }

  return (
    <div className="flex min-h-full">
      <SyncList
        onClick={setSelectedSyncID}
        selectedSyncID={selectedSyncID ?? firstSync.id}
        syncs={syncsRes.data}
      />

      <Sync externalAppID={externalAppID} syncID={selectedSyncID ?? firstSync.id} />
    </div>
  );
}
