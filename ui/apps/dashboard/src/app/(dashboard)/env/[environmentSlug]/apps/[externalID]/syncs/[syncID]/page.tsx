'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { FunctionList } from './FunctionList';
import { useSync } from './useSync';

type Props = {
  params: {
    externalID: string;
    syncID: string;
  };
};

export default function Page({ params }: Props) {
  const externalAppID = decodeURIComponent(params.externalID);
  const syncID = params.syncID;
  const env = useEnvironment();

  const syncRes = useSync({ envID: env.id, externalAppID, syncID });
  if (syncRes.error) {
    if (syncRes.error.message.includes('no rows')) {
      // TODO: Make prettier.
      return <div>Sync not found</div>;
    }
    throw syncRes.error;
  }
  if (syncRes.isLoading) {
    // TODO: Make prettier.
    return 'Loading...';
  }

  const { app } = syncRes.data.environment;
  const { sync } = syncRes.data;

  return (
    <div className="flex w-full justify-center p-4">
      <div className="w-full max-w-[1200px]">
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
