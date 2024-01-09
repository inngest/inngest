'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { FunctionList } from './FunctionList';
import { useSync } from './useSync';

type Props = {
  externalAppID: string;
  syncID: string;
};

export function Sync({ externalAppID, syncID }: Props) {
  const env = useEnvironment();

  const syncRes = useSync({ envID: env.id, externalAppID, syncID });
  if (syncRes.error) {
    if (syncRes.error.message.includes('no rows')) {
      // TODO: Make pretty
      return <div>Sync not found</div>;
    }
    throw syncRes.error;
  }
  if (syncRes.isLoading) {
    // TODO: Make pretty
    return 'Loading...';
  }

  const { app } = syncRes.data.environment;
  const { sync } = syncRes.data;

  return (
    <div className="h-full w-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] p-4">
        <AppInfoCard app={app} className="mb-4" sync={sync} />
        <AppGitCard className="mb-4" sync={sync} />

        <FunctionList
          removedFunctions={sync.removedFunctions}
          syncedFunctions={sync.syncedFunctions}
        />
      </div>
    </div>
  );
}
