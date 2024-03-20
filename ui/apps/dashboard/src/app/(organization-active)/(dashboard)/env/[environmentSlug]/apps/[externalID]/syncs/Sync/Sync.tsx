'use client';

import ExclamationTriangleIcon from '@heroicons/react/20/solid/ExclamationTriangleIcon';
import { Alert } from '@inngest/components/Alert';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { SyncErrorCard } from '@/components/SyncErrorCard';
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
      return (
        <div className="h-full w-full overflow-y-auto">
          <div className="mx-auto w-full max-w-[1200px] p-4">
            <div className="flex items-center gap-2.5 rounded-lg border border-red-500 bg-red-100 px-8 py-4 text-red-500">
              <ExclamationTriangleIcon className="h-5 w-5" />
              Sync not found
            </div>
          </div>
        </div>
      );
    }
    throw syncRes.error;
  }
  if (syncRes.isLoading) {
    return (
      <div className="h-full w-full overflow-y-auto">
        <div className="mx-auto w-full max-w-[1200px] p-4">
          <AppInfoCard className="mb-4" loading />
        </div>
      </div>
    );
  }

  const { app } = syncRes.data.environment;
  const { sync } = syncRes.data;

  return (
    <div className="h-full w-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] p-4">
        {sync.error && <SyncErrorCard className="mb-4" error={sync.error} />}

        {sync.status === 'duplicate' && (
          <Alert className="mb-4" severity="info">
            Function configurations have not changed since the last successful sync. Logic in
            function handlers may have changed, but they are not inspecting when syncing.
          </Alert>
        )}

        <AppInfoCard app={app} className="mb-4" sync={sync} linkToSyncs />
        <AppGitCard className="mb-4" sync={sync} />

        <FunctionList
          removedFunctions={sync.removedFunctions}
          syncedFunctions={sync.syncedFunctions}
        />
      </div>
    </div>
  );
}
