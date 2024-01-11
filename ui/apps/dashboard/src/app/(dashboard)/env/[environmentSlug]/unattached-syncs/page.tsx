'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { useSearchParam } from '@/utils/useSearchParam';
import { SyncList } from '../apps/[externalID]/syncs/SyncList';
import { Sync } from './Sync';
import { useSyncs } from './useSyncs';

export default function Page() {
  const env = useEnvironment();
  const [selectedSyncID, setSelectedSyncID] = useSearchParam('sync-id');

  const syncsRes = useSyncs({ envID: env.id });
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
